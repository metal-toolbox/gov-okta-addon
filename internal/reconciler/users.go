package reconciler

import (
	"context"
	"time"

	"go.equinixmetal.net/gov-okta-addon/internal/auctx"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	"go.uber.org/zap"
)

// cutoffUserDeleted is used to determine which deleted governor users will be removed from Okta
var cutoffUserDeleted = time.Now().Add(-24 * time.Hour)

// UserDelete deletes an okta user that has already been deleted in governor
// an error will be returned if the user still exists in governor
func (r *Reconciler) UserDelete(ctx context.Context, govID string) (string, error) {
	// get details about this user and verify it was actually deleted in governor
	user, err := r.governorClient.User(ctx, govID, true)
	if err != nil {
		r.logger.Error("failed to get user from governor", zap.Error(err))
		return "", err
	}

	r.logger.Debug("got governor user response", zap.Any("user details", user))

	extID := user.ExternalID.String

	logger := r.logger.With(
		zap.String("governor.user.id", user.ID),
		zap.String("governor.external_id", extID),
		zap.String("governor.user.email", user.Email),
	)

	if !userDeleted(user) {
		logger.Error("user still exists in governor")
		return "", ErrUserStillExists
	}

	oktaID, err := r.oktaClient.GetUserIDByEmail(ctx, user.Email)
	if err != nil {
		logger.Error("error looking up okta user by email address", zap.Error(err))
		return "", err
	}

	logger = logger.With(zap.String("okta.user.id", oktaID))

	if r.dryrun {
		logger.Info("SKIP deleting okta user")
		return extID, nil
	}

	if err := r.oktaClient.DeleteUser(ctx, oktaID); err != nil {
		logger.Error("error deleting okta user", zap.Error(err))
		return "", err
	}

	usersDeletedCounter.Inc()

	if err := auctx.WriteAuditEvent(ctx, r.auditEventWriter, "UserDelete", map[string]string{
		"governor.user.email": user.Email,
		"governor.user.id":    user.ID,
		"okta.user.id":        oktaID,
	}); err != nil {
		r.logger.Error("error writing audit event", zap.Error(err))
	}

	return oktaID, nil
}

// userDeleted returns true if the given user has been deleted in governor within the specified cutoff time period.
// The function also performs some basic user validation and will return false if anything with the user doesn't look right
func userDeleted(user *v1alpha1.User) bool {
	if user == nil {
		return false
	}

	// these fields should always be defined for a user
	if user.ID == "" || user.Name == "" || user.Email == "" {
		return false
	}

	if user.DeletedAt.IsZero() {
		return false
	}

	if user.DeletedAt.Time.After(cutoffUserDeleted) {
		return true
	}

	return false
}
