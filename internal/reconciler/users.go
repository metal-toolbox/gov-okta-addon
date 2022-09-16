package reconciler

import (
	"context"
	"time"

	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	"go.uber.org/zap"
)

// UserDelete deletes an okta user that has already been deleted in governor
// an error will be returned if the user still exists in governor
func (r *Reconciler) UserDelete(ctx context.Context, id string) (string, error) {
	logger := r.logger.With(zap.String("user.id", id))

	// get details about this user and verify it was actually deleted in governor
	user, err := r.governorClient.User(ctx, id, true)
	if err != nil {
		logger.Error("failed to get user from governor", zap.Error(err))
		return "", err
	}

	logger.Debug("got governor user response", zap.Any("user details", user))

	if !userDeleted(user) {
		logger.Error("user still exists in governor")
		return "", ErrUserStillExists
	}

	if err := r.oktaClient.DeleteUser(ctx, user.ExternalID); err != nil {
		r.logger.Error("error deleting user", zap.Error(err))
		return "", err
	}

	return user.ExternalID, nil
}

// userDeleted returns true if the given user has been deleted in governor within the specified time period, currently 24h
// the function will also perform some basic user validation and will return false if anything with the user doesn't look right
func userDeleted(user *v1alpha1.User) bool {
	if user == nil {
		return false
	}

	// these fields should always be defined for a user
	if user.ID == "" || user.ExternalID == "" || user.Name == "" || user.Email == "" {
		return false
	}

	cutoff := time.Now().Add(-24 * time.Hour)

	if user.DeletedAt.IsZero() {
		return false
	}

	if user.DeletedAt.Time.After(cutoff) {
		return true
	}

	return false
}
