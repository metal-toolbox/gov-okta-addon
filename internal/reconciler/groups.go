package reconciler

import (
	"context"

	"github.com/equinixmetal/gov-okta-addon/internal/auctx"
	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	"go.uber.org/zap"
)

// GroupsApplicationAssignments reconciles application assignments in okta for a list of governor groups
func (r *Reconciler) GroupsApplicationAssignments(ctx context.Context, ids ...string) error {
	groupMap := map[string]*v1alpha1.Group{}

	for _, id := range ids {
		logger := r.logger.With(zap.String("group.id", id))

		// get the details about a governor group
		group, err := r.governorClient.Group(ctx, id, false)
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

	return r.reconcileGroupApplicationAssignments(ctx, groupMap)
}

// GroupCreate creates a governor group in okta
func (r *Reconciler) GroupCreate(ctx context.Context, id string) (string, error) {
	group, err := r.governorClient.Group(ctx, id, false)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return "", err
	}

	logger := r.logger.With(zap.String("governor.group.id", group.ID), zap.String("governor.group.slug", group.Slug))

	if r.dryrun {
		logger.Info("SKIP creating okta group")
		return "dryrun", nil
	}

	oktaGID, err := r.oktaClient.CreateGroup(ctx, group.Name, group.Description, map[string]interface{}{"governor_id": group.ID})
	if err != nil {
		logger.Error("error creating okta group", zap.Error(err))
		return "", err
	}

	groupsCreatedCounter.Inc()

	logger.Info("created okta group", zap.String("okta.group.id", oktaGID))

	if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupCreate", map[string]string{
		"governor.group.slug": group.Slug,
		"governor.group.id":   group.ID,
		"okta.group.id":       oktaGID,
	}); err != nil {
		logger.Error("error writing audit event", zap.Error(err))
	}

	return oktaGID, nil
}

// GroupUpdate updates an existing governor group in okta
func (r *Reconciler) GroupUpdate(ctx context.Context, id string) (string, error) {
	group, err := r.governorClient.Group(ctx, id, false)
	if err != nil {
		r.logger.Error("failed to get group from governor", zap.Error(err))
		return "", err
	}

	logger := r.logger.With(zap.String("governor.group.id", group.ID), zap.String("governor.group.slug", group.Slug))

	oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, group.ID)
	if err != nil {
		logger.Error("error getting group by governor id", zap.String("governor.group.id", group.ID), zap.Error(err))
		return "", err
	}

	if r.dryrun {
		logger.Info("SKIP updating okta group")
		return oktaGID, nil
	}

	if _, err := r.oktaClient.UpdateGroup(ctx, oktaGID, group.Name, group.Description, map[string]interface{}{"governor_id": group.ID}); err != nil {
		logger.Error("error updating group", zap.Error(err))
		return "", err
	}

	groupsUpdatedCounter.Inc()

	if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupUpdate", map[string]string{
		"governor.group.slug": group.Slug,
		"governor.group.id":   group.ID,
		"okta.group.id":       oktaGID,
	}); err != nil {
		logger.Error("error writing audit event", zap.Error(err))
	}

	return oktaGID, nil
}

// GroupDelete deletes an existing governor group in okta
func (r *Reconciler) GroupDelete(ctx context.Context, id string) (string, error) {
	// TODO validate the group is deleted from governor API by ID
	oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, id)
	if err != nil {
		r.logger.Error("error getting okta group by governor id", zap.String("governor.group.id", id), zap.Error(err))
		return "", err
	}

	if r.dryrun {
		r.logger.Info("dryrun deleting okta group", zap.String("okta.group.id", oktaGID))
		return oktaGID, nil
	}

	if err := r.oktaClient.DeleteGroup(ctx, oktaGID); err != nil {
		r.logger.Error("error deleting group", zap.Error(err))
		return "", err
	}

	groupsDeletedCounter.Inc()

	if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupDelete", map[string]string{
		"governor.group.id": id,
		"okta.group.id":     oktaGID,
	}); err != nil {
		r.logger.Error("error writing audit event", zap.Error(err))
	}

	return oktaGID, nil
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
