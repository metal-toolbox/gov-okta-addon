package okta

import (
	"context"
	"fmt"

	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

// GetUserIDByEmail gets an okta user id from the user's email address
func (c *Client) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	c.logger.Info("getting user by email address", zap.String("user.email", email))

	f := fmt.Sprintf("profile.email eq %s", email)

	users, _, err := c.userIface.ListUsers(ctx, &query.Params{Search: f})
	if err != nil {
		return "", err
	}

	if len(users) != 1 {
		return "", ErrUnexpectedUsersCount
	}

	uid := users[0].Id

	c.logger.Debug("found okta user by email", zap.String("user.email", email), zap.String("okta.user.id", uid))

	return uid, nil
}
