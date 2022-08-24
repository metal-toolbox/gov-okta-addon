package reconciler

import (
	"context"

	"go.uber.org/zap"
)

// GroupMembership performs a full reconciliation on the membership of a group in okta
func (r *Reconciler) GroupMembership(ctx context.Context, gid, oktaGID string) error {
	group, err := r.governorClient.Group(ctx, gid)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return err
	}

	logger := r.logger.With(zap.String("governor.group.id", gid), zap.String("okta.group.id", oktaGID))

	oktaGroupMembers, err := r.oktaClient.ListGroupMembership(ctx, oktaGID)
	if err != nil {
		logger.Error("error getting group membership for okta group")
	}

	// keep a map of okta uids to governor uids for quick lookup and less calls
	oktaUserMap := make(map[string]string)

	for _, uid := range group.Members {
		user, err := r.governorClient.User(ctx, uid)
		if err != nil {
			logger.Error("error getting governor user", zap.Error(err))
			continue
		}

		oktaUID, err := r.oktaClient.GetUserIDByEmail(ctx, user.Email)
		if err != nil {
			logger.Error("error getting user by email address", zap.String("user.email", user.Email), zap.Error(err))
			continue
		}

		oktaUserMap[oktaUID] = uid

		// if the okta group already contains the uid, continue
		if contains(oktaGroupMembers, oktaUID) {
			logger.Debug("okta group already contains member, not adding")
			continue
		}

		// otherwise add the member
		if err := r.oktaClient.AddGroupUser(ctx, oktaGID, oktaUID); err != nil {
			logger.Error("failed to add user to okta group",
				zap.String("user.email", user.Email),
				zap.String("okta.user.id", oktaGID),
				zap.Error(err),
			)

			continue
		}
	}

	for _, oktaUID := range oktaGroupMembers {
		// if the governor group contains the uid, continue
		if contains(group.Members, oktaUserMap[oktaUID]) {
			logger.Debug("governor group contains member, not removing")
			continue
		}

		// otherwise remove the member
		if err := r.oktaClient.RemoveGroupUser(ctx, oktaGID, oktaUID); err != nil {
			logger.Error("failed to remove user from okta group",
				zap.String("okta.user.id", oktaGID),
				zap.String("okta.group.id", oktaUID),
				zap.Error(err),
			)

			continue
		}
	}

	return nil
}

// GroupMembershipCreate reconciles the existence of a user in an okta group based on the given governor user and group ids
func (r *Reconciler) GroupMembershipCreate(ctx context.Context, gid, uid string) (string, string, error) {
	group, err := r.governorClient.Group(ctx, gid)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return "", "", err
	}

	r.logger.Debug("got group response", zap.Any("group details", group))

	user, err := r.governorClient.User(ctx, uid)
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

	if !contains(group.Members, user.ID) {
		logger.Error("governor group does not contain requested membership")
		return "", "", ErrGroupMembershipNotFound
	}

	oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, gid)
	if err != nil {
		logger.Error("error getting group by governor id", zap.String("governor.group.id", gid), zap.Error(err))
		return "", "", err
	}

	oktaUID, err := r.oktaClient.GetUserIDByEmail(ctx, user.Email)
	if err != nil {
		logger.Error("error getting user by email address", zap.String("user.email", user.Email), zap.Error(err))
		return "", "", err
	}

	if err := r.oktaClient.AddGroupUser(ctx, oktaGID, oktaUID); err != nil {
		logger.Error("failed to add user to group",
			zap.String("user.email", user.Email),
			zap.String("okta.user.id", oktaGID),
			zap.String("okta.group.id", oktaUID),
			zap.Error(err),
		)

		return "", "", err
	}

	return oktaGID, oktaUID, nil
}

// GroupMembershipDelete reconciles the removal a user from an okta group based on the given governor group and user ids
func (r *Reconciler) GroupMembershipDelete(ctx context.Context, gid, uid string) (string, string, error) {
	group, err := r.governorClient.Group(ctx, gid)
	if err != nil {
		r.logger.Error("error getting governor group", zap.Error(err))
		return "", "", err
	}

	r.logger.Debug("got group response", zap.Any("group details", group))

	user, err := r.governorClient.User(ctx, uid)
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

	if contains(group.Members, user.ID) {
		logger.Error("governor group contains requested membership delete")
		return "", "", ErrGroupMembershipFound
	}

	oktaGID, err := r.oktaClient.GetGroupByGovernorID(ctx, gid)
	if err != nil {
		logger.Error("error getting group by governor id", zap.String("governor.group.id", gid), zap.Error(err))
		return "", "", err
	}

	oktaUID, err := r.oktaClient.GetUserIDByEmail(ctx, user.Email)
	if err != nil {
		logger.Error("error getting user by email address", zap.String("user.email", user.Email), zap.Error(err))
		return "", "", err
	}

	if err := r.oktaClient.RemoveGroupUser(context.Background(), oktaGID, oktaUID); err != nil {
		logger.Error("failed to remove user from group",
			zap.String("user.email", user.Email),
			zap.String("okta.user.id", oktaGID),
			zap.String("okta.group.id", oktaUID),
			zap.Error(err),
		)

		return "", "", err
	}

	return oktaGID, oktaUID, nil
}
