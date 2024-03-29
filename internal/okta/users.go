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

// UserDetails contains the details of an Okta user
type UserDetails struct {
	ID     string
	Name   string
	Email  string
	Status string
}

// GetUser gets an okta user by id
func (c *Client) GetUser(ctx context.Context, id string) (*okta.User, error) {
	c.logger.Debug("getting okta user", zap.String("okta.user.id", id))

	user, _, err := c.userIface.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("returning okta user", zap.Any("okta.user", user))

	return user, nil
}

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

	// make sure the user is deactivated first
	if user.Status != "DEPROVISIONED" {
		c.logger.Debug("deactivating user", zap.String("okta.user.id", id))

		if _, err := c.userIface.DeactivateUser(ctx, id, &query.Params{}); err != nil {
			return err
		}
	}

	if _, err := c.userIface.DeactivateOrDeleteUser(ctx, id, &query.Params{}); err != nil {
		return err
	}

	// TODO clear any sessions in Okta

	c.logger.Debug("deleted okta user", zap.String("okta.user.id", id))

	return nil
}

// ClearUserSessions removes all active idp sessiosn and forces the user to reauthenticate.
func (c *Client) ClearUserSessions(ctx context.Context, id string) error {
	c.logger.Info("clearing user sessions", zap.String("okta.user.id", id))

	if _, err := c.userIface.ClearUserSessions(ctx, id, &query.Params{}); err != nil {
		return err
	}

	c.logger.Debug("cleared user sessions", zap.String("okta.user.id", id))

	return nil
}

// GetUserIDByEmail gets an okta user id from the user's email address
func (c *Client) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	c.logger.Debug("getting okta user by email", zap.String("user.email", email))

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
	c.logger.Debug("listing users")

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
	c.logger.Debug("listing users with func")

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

// SuspendUser suspends an active user in Okta
func (c *Client) SuspendUser(ctx context.Context, id string) error {
	c.logger.Info("suspending okta user", zap.String("okta.user.id", id))

	if _, err := c.userIface.SuspendUser(ctx, id); err != nil {
		return err
	}

	c.logger.Debug("suspended okta user", zap.String("okta.user.id", id))

	return nil
}

// UnsuspendUser un-suspends a user in Okta and returns them to active state
func (c *Client) UnsuspendUser(ctx context.Context, id string) error {
	c.logger.Info("un-suspending okta user", zap.String("okta.user.id", id))

	if _, err := c.userIface.UnsuspendUser(ctx, id); err != nil {
		return err
	}

	c.logger.Debug("un-suspended okta user", zap.String("okta.user.id", id))

	return nil
}

// EmailFromUserProfile parses the email from the okta user profile
func EmailFromUserProfile(u *okta.User) (string, error) {
	// get the email from the user profile
	for k, v := range *u.Profile {
		if k == "email" {
			if fv, ok := v.(string); ok {
				return fv, nil
			}

			return "", ErrOktaUserEmailNotString
		}
	}

	return "", fmt.Errorf("email not found for user %s", u.Id) //nolint:goerr113
}

// FirstNameFromUserProfile parses the firstName from the okta user profile
func FirstNameFromUserProfile(u *okta.User) (string, error) {
	// get the firstName from the user profile
	for k, v := range *u.Profile {
		if k == "firstName" {
			if fv, ok := v.(string); ok {
				return fv, nil
			}

			return "", ErrOktaUserFirstNameNotString
		}
	}

	return "", fmt.Errorf("firstName not found for user %s", u.Id) //nolint:goerr113
}

// LastNameFromUserProfile parses the lastName from the okta user profile
func LastNameFromUserProfile(u *okta.User) (string, error) {
	// get the lastName from the user profile
	for k, v := range *u.Profile {
		if k == "lastName" {
			if fv, ok := v.(string); ok {
				return fv, nil
			}

			return "", ErrOktaUserLastNameNotString
		}
	}

	return "", fmt.Errorf("lastName not found for user %s", u.Id) //nolint:goerr113
}

// UserDetailsFromOktaUser parses the relevant user details from the okta user object
func UserDetailsFromOktaUser(u *okta.User) (*UserDetails, error) {
	d := &UserDetails{
		ID:     u.Id,
		Status: u.Status,
	}

	var firstName, lastName string

	for k, v := range *u.Profile {
		if k == "firstName" {
			fn, ok := v.(string)
			if !ok {
				return nil, ErrOktaUserLastNameNotString
			}

			firstName = fn
		}

		if k == "lastName" {
			ln, ok := v.(string)
			if !ok {
				return nil, ErrOktaUserFirstNameNotString
			}

			lastName = ln
		}

		if k == "email" {
			e, ok := v.(string)
			if !ok {
				return nil, ErrOktaUserEmailNotString
			}

			d.Email = e
		}
	}

	if firstName == "" {
		return nil, fmt.Errorf("firstName not found for user %s", u.Id) //nolint:goerr113
	}

	if lastName == "" {
		return nil, fmt.Errorf("lastName not found for user %s", u.Id) //nolint:goerr113
	}

	if d.Email == "" {
		return nil, fmt.Errorf("email not found for user %s", u.Id) //nolint:goerr113
	}

	d.Name = fmt.Sprintf("%s %s", firstName, lastName)

	return d, nil
}
