package okta

import (
	"context"
	"fmt"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

// UserModifierFunc modifies a an okta user response
type UserModifierFunc func(context.Context, *okta.User) (*okta.User, error)

// GetUserIDByEmail gets an okta user id from the user's email address
func (c *Client) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	c.logger.Info("getting okta user by email", zap.String("user.email", email))

	f := fmt.Sprintf("profile.email eq \"%s\"", email)

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

// ListUsersWithModifier lists okta users and modifies the user response with the given UserModifierFunc.  If nil is
// returned from the UserModifierFunc, the user will not be returned in the response.
func (c *Client) ListUsersWithModifier(ctx context.Context, f UserModifierFunc, q *query.Params) ([]*okta.User, error) {
	c.logger.Info("listing users with func")

	users, resp, err := c.userIface.ListUsers(ctx, q)
	if err != nil {
		return nil, err
	}

	userResp := []*okta.User{}

	for _, u := range users {
		c.logger.Debug("running function on user", zap.Any("user", u))

		user, err := f(ctx, u)
		if err != nil {
			return nil, err
		}

		if user != nil {
			userResp = append(userResp, user)
		}
	}

	for {
		if !resp.HasNextPage() {
			break
		}

		nextPage := []*okta.User{}

		resp, err = resp.Next(ctx, &nextPage)
		if err != nil {
			return nil, err
		}

		for _, u := range nextPage {
			c.logger.Debug("running function on user", zap.Any("user", u))

			user, err := f(ctx, u)
			if err != nil {
				return nil, err
			}

			if user != nil {
				userResp = append(userResp, user)
			}
		}
	}

	c.logger.Debug("returning list of users", zap.Int("num.okta.users", len(userResp)))

	return userResp, nil
}
