package reconciler

import (
	"context"

	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	"go.uber.org/zap"
)

// GroupsApplicationAssignments reconciles application assignments in okta for a list of governor groups
func (r *Reconciler) GroupsApplicationAssignments(ctx context.Context, ids ...string) error {
	groupMap := map[string]*v1alpha1.Group{}

	for _, id := range ids {
		logger := r.logger.With(zap.String("group.id", id))

		// get the details about a governor group
		group, err := r.governorClient.Group(ctx, id)
		if err != nil {
			logger.Error("error getting governor group details", zap.Error(err))
			continue
		}

		logger.Debug("got governor group response", zap.Any("group details", group))

		// get the okta id for the governor group
		oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, id)
		if err != nil {
			logger.Error("error getting okta group by governor id", zap.Error(err))
			continue
		}

		groupMap[oktaGID] = group
	}

	if err := r.reconcileGroupApplicationAssignments(ctx, groupMap); err != nil {
		return err
	}

	return nil
}

// GroupCreate creates a governor group in okta
func (r *Reconciler) GroupCreate(ctx context.Context, id string) (string, error) {
	group, err := r.governorClient.Group(ctx, id)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return "", err
	}

	logger := r.logger.With(zap.String("governor.group.id", group.ID), zap.String("governor.group.slug", group.Slug))

	oktaGID, err := r.oktaClient.CreateGroup(ctx, group.Name, group.Description, map[string]interface{}{"governor_id": group.ID})
	if err != nil {
		logger.Error("error creating okta group", zap.Error(err))
		return "", err
	}

	logger.Info("created okta group", zap.String("okta.group.id", oktaGID))

	return oktaGID, nil
}

// GroupUpdate updates an existing governor group in okta
func (r *Reconciler) GroupUpdate(ctx context.Context, id string) (string, error) {
	group, err := r.governorClient.Group(ctx, id)
	if err != nil {
		r.logger.Error("failed to get group from governor", zap.Error(err))
		return "", err
	}

	logger := r.logger.With(zap.String("governor.group.id", group.ID), zap.String("governor.group.slug", group.Slug))

	gid, err := r.oktaClient.GetGroupByGovernorID(ctx, group.ID)
	if err != nil {
		logger.Error("error getting group by governor id", zap.String("governor.group.id", group.ID), zap.Error(err))
		return "", err
	}

	if _, err := r.oktaClient.UpdateGroup(ctx, gid, group.Name, group.Description, map[string]interface{}{"governor_id": group.ID}); err != nil {
		logger.Error("error updating group", zap.Error(err))
		return "", err
	}

	return gid, nil
}

// GroupDelete deletes an existing governor group in okta
func (r *Reconciler) GroupDelete(ctx context.Context, id string) (string, error) {
	// TODO validate the group is deleted from governor API by ID
	gid, err := r.oktaClient.GetGroupByGovernorID(ctx, id)
	if err != nil {
		r.logger.Error("error getting okta group by governor id", zap.String("governor.group.id", id), zap.Error(err))
		return "", err
	}

	if err := r.oktaClient.DeleteGroup(ctx, gid); err != nil {
		r.logger.Error("error deleting group", zap.Error(err))
		return "", err
	}

	return gid, nil
}

// getGroupOrgSlugs returns the github organization slugs assigned to a governor group
func getGroupOrgSlugs(group *v1alpha1.Group, orgs []*v1alpha1.Organization) []string {
	slugs := []string{}

	for _, g := range group.Organizations {
		for _, o := range orgs {
			if o.ID == g {
				slugs = append(slugs, o.Slug)
				break
			}
		}
	}

	return slugs
}
