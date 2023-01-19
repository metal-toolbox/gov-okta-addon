package reconciler

import (
	"context"

	"go.equinixmetal.net/gov-okta-addon/internal/auctx"
	"go.uber.org/zap"
)

// GroupMembership performs a full reconciliation on the membership of a group in okta
func (r *Reconciler) GroupMembership(ctx context.Context, gid, oktaGID string) error {
	group, err := r.governorClient.Group(ctx, gid, false)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return err
	}

	logger := r.logger.With(
		zap.String("governor.group.id", gid),
		zap.String("okta.group.id", oktaGID),
	)

	oktaGroupMembers, err := r.oktaClient.ListGroupMembership(ctx, oktaGID)
	if err != nil {
		logger.Error("error getting group membership for okta group")
	}

	oktaGroupMemberIDs := make([]string, len(oktaGroupMembers))
	for i, g := range oktaGroupMembers {
		oktaGroupMemberIDs[i] = g.Id
	}

	// keep a map of okta uids to governor uids for quick lookup and less calls
	oktaUserMap := make(map[string]string)

	for _, uid := range group.Members {
		user, err := r.governorClient.User(ctx, uid, false)
		if err != nil {
			logger.Error("error getting governor user", zap.Error(err))
			continue
		}

		if user.Status.String == "pending" {
			logger.Debug("skipping user with pending status",
				zap.String("governor.user.email", user.Email),
				zap.String("governor.user.id", user.ID),
			)

			continue
		} else if user.ExternalID.String == "" {
			logger.Debug("skipping user with missing external id",
				zap.String("governor.user.email", user.Email),
				zap.String("governor.user.id", user.ID),
			)

			continue
		}

		// NOTE: we are skipping group members if the external id is empty and then
		// assuming the external id is an okta ID.  This works for now, but may need
		// to be updated if external_id could be am ID in a different system or missing
		// for valid okta users.
		oktaUID := user.ExternalID.String
		oktaUserMap[oktaUID] = uid

		// if the okta group already contains the uid, continue
		if contains(oktaGroupMemberIDs, oktaUID) {
			logger.Debug("okta group already contains member, not adding")
			continue
		}

		// otherwise add the member
		if !r.dryrun {
			if err := r.oktaClient.AddGroupUser(ctx, oktaGID, oktaUID); err != nil {
				logger.Error("failed to add user to okta group",
					zap.String("user.email", user.Email),
					zap.String("okta.user.id", oktaUID),
					zap.Error(err),
				)

				continue
			}

			groupMembershipCreatedCounter.Inc()

			if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupMemberAdd", map[string]string{
				"governor.group.slug": group.Slug,
				"governor.group.id":   group.ID,
				"governor.user.email": user.Email,
				"governor.user.id":    user.ID,
				"okta.group.id":       oktaGID,
				"okta.user.id":        oktaUID,
			}); err != nil {
				logger.Error("error writing audit event", zap.Error(err))
			}
		} else {
			logger.Info("SKIP adding user to okta group",
				zap.String("user.email", user.Email),
				zap.String("okta.user.id", oktaUID),
			)
		}
	}

	for _, oktaUID := range oktaGroupMemberIDs {
		// if the governor group contains the uid, continue
		if contains(group.Members, oktaUserMap[oktaUID]) {
			logger.Debug("governor group contains member, not removing")
			continue
		}

		// otherwise remove the member
		if !r.dryrun && !r.skipDelete {
			if err := r.oktaClient.RemoveGroupUser(ctx, oktaGID, oktaUID); err != nil {
				logger.Error("failed to remove user from okta group",
					zap.String("okta.user.id", oktaUID),
					zap.Error(err),
				)

				continue
			}

			groupMembershipDeletedCounter.Inc()

			if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupMemberRemove", map[string]string{
				"governor.group.slug": group.Slug,
				"governor.group.id":   group.ID,
				"okta.group.id":       oktaGID,
				"okta.user.id":        oktaUID,
			}); err != nil {
				logger.Error("error writing audit event", zap.Error(err))
			}
		} else {
			logger.Info("SKIP removing user from okta group",
				zap.String("okta.user.id", oktaUID),
			)
		}
	}

	return nil
}

// GroupMembershipCreate reconciles the existence of a user in an okta group based on the given governor user and group ids
func (r *Reconciler) GroupMembershipCreate(ctx context.Context, gid, uid string) (string, string, error) {
	group, err := r.governorClient.Group(ctx, gid, false)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return "", "", err
	}

	r.logger.Debug("got group response", zap.Any("group details", group))

	user, err := r.governorClient.User(ctx, uid, false)
	if err != nil {
		r.logger.Error("error getting governor user", zap.Error(err))
		return "", "", err
	}

	logger := r.logger.With(
		zap.String("governor.group.id", group.ID),
		zap.String("governor.group.slug", group.Slug),
		zap.String("governor.user.id", user.ID),
		zap.String("governor.user.email", user.Email),
	)

	if user.Status.String == "pending" {
		logger.Info("skipping pending user")
		return "", "", ErrGovernorUserPendingStatus
	}

	if !contains(group.Members, user.ID) {
		logger.Error("governor group does not contain requested membership")
		return "", "", ErrGroupMembershipNotFound
	}

	oktaUID, err := r.oktaClient.GetUserIDByEmail(ctx, user.Email)
	if err != nil {
		logger.Error("error getting okta user by email", zap.Error(err))
		return "", "", err
	}

	oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, gid)
	if err != nil {
		logger.Error("error getting group by governor id", zap.String("governor.group.id", gid), zap.Error(err))
		return "", "", err
	}

	if r.dryrun {
		logger.Info("SKIP adding user to okta group",
			zap.String("user.email", user.Email),
			zap.String("okta.user.id", oktaUID),
			zap.String("okta.group.id", oktaGID),
		)

		return oktaGID, oktaUID, nil
	}

	if err := r.oktaClient.AddGroupUser(ctx, oktaGID, oktaUID); err != nil {
		logger.Error("failed to add user to group",
			zap.String("user.email", user.Email),
			zap.String("okta.user.id", oktaUID),
			zap.String("okta.group.id", oktaGID),
			zap.Error(err),
		)

		return "", "", err
	}

	groupMembershipCreatedCounter.Inc()

	if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupMemberAdd", map[string]string{
		"governor.group.slug": group.Slug,
		"governor.group.id":   group.ID,
		"governor.user.email": user.Email,
		"governor.user.id":    user.ID,
		"okta.group.id":       oktaGID,
		"okta.user.id":        oktaUID,
	}); err != nil {
		logger.Error("error writing audit event", zap.Error(err))
	}

	return oktaGID, oktaUID, nil
}

// GroupMembershipDelete reconciles the removal a user from an okta group based on the given governor group and user ids
func (r *Reconciler) GroupMembershipDelete(ctx context.Context, gid, uid string) (string, string, error) {
	group, err := r.governorClient.Group(ctx, gid, false)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return "", "", err
	}

	r.logger.Debug("got group response", zap.Any("group details", group))

	user, err := r.governorClient.User(ctx, uid, false)
	if err != nil {
		r.logger.Error("error getting governor user", zap.Error(err))
		return "", "", err
	}

	logger := r.logger.With(
		zap.String("governor.group.id", group.ID),
		zap.String("governor.group.slug", group.Slug),
		zap.String("governor.user.id", user.ID),
		zap.String("governor.user.email", user.Email),
	)

	if user.Status.String == "pending" {
		logger.Info("skipping pending user")
		return "", "", ErrGovernorUserPendingStatus
	}

	if contains(group.Members, user.ID) {
		logger.Error("governor group contains requested membership delete")
		return "", "", ErrGroupMembershipFound
	}

	oktaUID, err := r.oktaClient.GetUserIDByEmail(ctx, user.Email)
	if err != nil {
		logger.Error("error getting okta user by email", zap.Error(err))
		return "", "", err
	}

	oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, gid)
	if err != nil {
		logger.Error("error getting group by governor id", zap.String("governor.group.id", gid), zap.Error(err))
		return "", "", err
	}

	if r.dryrun {
		logger.Info("SKIP removing user from okta group",
			zap.String("user.email", user.Email),
			zap.String("okta.user.id", oktaUID),
			zap.String("okta.group.id", oktaGID),
		)

		return oktaGID, oktaUID, nil
	}

	if err := r.oktaClient.RemoveGroupUser(context.Background(), oktaGID, oktaUID); err != nil {
		logger.Error("failed to remove user from group",
			zap.String("user.email", user.Email),
			zap.String("okta.user.id", oktaUID),
			zap.String("okta.group.id", oktaGID),
			zap.Error(err),
		)

		return "", "", err
	}

	groupMembershipDeletedCounter.Inc()

	if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "GroupMemberRemove", map[string]string{
		"governor.group.slug": group.Slug,
		"governor.group.id":   group.ID,
		"governor.user.email": user.Email,
		"governor.user.id":    user.ID,
		"okta.group.id":       oktaGID,
		"okta.user.id":        oktaUID,
	}); err != nil {
		logger.Error("error writing audit event", zap.Error(err))
	}

	return oktaGID, oktaUID, nil
}
