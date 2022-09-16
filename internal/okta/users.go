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

// DeactivateUser deactivates a user in Okta
func (c *Client) DeactivateUser(ctx context.Context, id string) error {
	c.logger.Info("deactivating okta user", zap.String("okta.user.id", id))

	if _, err := c.userIface.DeactivateUser(ctx, id, &query.Params{}); err != nil {
		return err
	}

	c.logger.Debug("deactivated okta user", zap.String("okta.user.id", id))

	return nil
}

// DeleteUser deletes a user in Okta
// since Okta requires that a user must be first deactivated before being deleted, we do this in two steps
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	c.logger.Info("deleting okta user", zap.String("okta.user.id", id))

	// look up the user in okta so we can get their status
	user, _, err := c.userIface.GetUser(ctx, id)
	if err != nil {
		return err
	}

	c.logger.Debug("got okta user status", zap.String("okta.user.status", user.Status))

	// if a user is already deprovisioned this will delete the user, otherwise, it will deprovision them
	if _, err := c.userIface.DeactivateOrDeleteUser(ctx, id, &query.Params{}); err != nil {
		return err
	}

	// run this again to delete the user, unless they were already deprovisioned
	if user.Status != "DEPROVISIONED" {
		if _, err := c.userIface.DeactivateOrDeleteUser(ctx, id, &query.Params{}); err != nil {
			return err
		}
	}

	// TODO: do we need to clear any sessions in Okta?

	c.logger.Debug("deleted okta user", zap.String("okta.user.id", id))

	return nil
}

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

// ListUsers lists all okta users
func (c *Client) ListUsers(ctx context.Context) ([]*okta.User, error) {
	c.logger.Info("listing users")

	users, resp, err := c.userIface.ListUsers(ctx, &query.Params{})
	if err != nil {
		return nil, err
	}

	userResp := users

	for {
		if !resp.HasNextPage() {
			break
		}

		nextPage := []*okta.User{}

		resp, err = resp.Next(ctx, &nextPage)
		if err != nil {
			return nil, err
		}

		userResp = append(userResp, nextPage...)
	}

	c.logger.Debug("returning list of users", zap.Int("num.okta.users", len(userResp)))

	return userResp, nil
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
