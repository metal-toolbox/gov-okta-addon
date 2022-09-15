package reconciler

import (
	"context"

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

	if user.DeletedAt.IsZero() {
		logger.Error("user still exists in governor")
		return "", ErrUserStillExists
	}

	if err := r.oktaClient.DeleteUser(ctx, user.ExternalID); err != nil {
		r.logger.Error("error deleting user", zap.Error(err))
		return "", err
	}

	return user.ExternalID, nil
}
