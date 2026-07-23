package okta

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/okta/okta-sdk-golang/v6/okta"
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

// userFromGetSingleton converts the okta UserGetSingleton returned by GetUser into a
// *okta.User via a JSON round-trip so callers can work with a single user type.
func userFromGetSingleton(singleton *okta.UserGetSingleton) (*okta.User, error) {
	if singleton == nil {
		return nil, nil
	}

	b, err := json.Marshal(singleton)
	if err != nil {
		return nil, err
	}

	user := &okta.User{}
	if err := json.Unmarshal(b, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetUser gets an okta user by id
func (c *Client) GetUser(ctx context.Context, id string) (*okta.User, error) {
	c.logger.Debug("getting okta user", zap.String("okta.user.id", id))

	singleton, _, err := c.client.UserAPI.GetUser(ctx, id).Execute()
	if err != nil {
		return nil, err
	}

	user, err := userFromGetSingleton(singleton)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("returning okta user", zap.Any("okta.user", user))

	return user, nil
}

// DeactivateUser deactivates a user in Okta
func (c *Client) DeactivateUser(ctx context.Context, id string) error {
	c.logger.Info("deactivating okta user", zap.String("okta.user.id", id))

	if _, err := c.client.UserLifecycleAPI.DeactivateUser(ctx, id).Execute(); err != nil {
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
	user, err := c.GetUser(ctx, id)
	if err != nil {
		return err
	}

	c.logger.Debug("got okta user status", zap.String("okta.user.status", user.GetStatus()))

	// make sure the user is deactivated first
	if user.GetStatus() != "DEPROVISIONED" {
		c.logger.Debug("deactivating user", zap.String("okta.user.id", id))

		if _, err := c.client.UserLifecycleAPI.DeactivateUser(ctx, id).Execute(); err != nil {
			return err
		}
	}

	if _, err := c.client.UserAPI.DeleteUser(ctx, id).Execute(); err != nil {
		return err
	}

	// TODO clear any sessions in Okta

	c.logger.Debug("deleted okta user", zap.String("okta.user.id", id))

	return nil
}

// ClearUserSessions removes all active idp sessiosn and forces the user to reauthenticate.
func (c *Client) ClearUserSessions(ctx context.Context, id string) error {
	c.logger.Info("clearing user sessions", zap.String("okta.user.id", id))

	if _, err := c.client.UserSessionsAPI.RevokeUserSessions(ctx, id).Execute(); err != nil {
		return err
	}

	c.logger.Debug("cleared user sessions", zap.String("okta.user.id", id))

	return nil
}

// GetUserIDByEmail gets an okta user id from the user's email address
func (c *Client) GetUserIDByEmail(ctx context.Context, email string) (string, error) {
	c.logger.Debug("getting okta user by email", zap.String("user.email", email))

	f := fmt.Sprintf("profile.email eq \"%s\"", email)

	users, err := paginate(func(after string) ([]okta.User, *okta.APIResponse, error) {
		req := c.client.UserAPI.ListUsers(ctx).Search(f).Limit(defaultPageLimit)
		if after != "" {
			req = req.After(after)
		}

		return req.Execute()
	})
	if err != nil {
		return "", err
	}

	if len(users) != 1 {
		return "", ErrUnexpectedUsersCount
	}

	uid := users[0].GetId()

	c.logger.Debug("found okta user by email", zap.String("user.email", email), zap.String("okta.user.id", uid))

	return uid, nil
}

// ListUsers lists all okta users
func (c *Client) ListUsers(ctx context.Context) ([]*okta.User, error) {
	c.logger.Debug("listing users")

	users, err := paginate(func(after string) ([]okta.User, *okta.APIResponse, error) {
		req := c.client.UserAPI.ListUsers(ctx).Limit(defaultPageLimit)
		if after != "" {
			req = req.After(after)
		}

		return req.Execute()
	})
	if err != nil {
		return nil, err
	}

	userResp := make([]*okta.User, len(users))
	for i := range users {
		userResp[i] = &users[i]
	}

	c.logger.Debug("returning list of users", zap.Int("num.okta.users", len(userResp)))

	return userResp, nil
}

// ListUsersWithModifier lists okta users and modifies the user response with the given UserModifierFunc.  If nil is
// returned from the UserModifierFunc, the user will not be returned in the response.
func (c *Client) ListUsersWithModifier(ctx context.Context, f UserModifierFunc, search string) ([]*okta.User, error) {
	c.logger.Debug("listing users with func")

	users, err := paginate(func(after string) ([]okta.User, *okta.APIResponse, error) {
		req := c.client.UserAPI.ListUsers(ctx).Limit(defaultPageLimit)
		if search != "" {
			req = req.Search(search)
		}

		if after != "" {
			req = req.After(after)
		}

		return req.Execute()
	})
	if err != nil {
		return nil, err
	}

	userResp := []*okta.User{}

	for i := range users {
		u := &users[i]

		c.logger.Debug("running function on user", zap.Any("user", u))

		user, err := f(ctx, u)
		if err != nil {
			return nil, err
		}

		if user != nil {
			userResp = append(userResp, user)
		}
	}

	c.logger.Debug("returning list of users", zap.Int("num.okta.users", len(userResp)))

	return userResp, nil
}

// SuspendUser suspends an active user in Okta
func (c *Client) SuspendUser(ctx context.Context, id string) error {
	c.logger.Info("suspending okta user", zap.String("okta.user.id", id))

	if _, err := c.client.UserLifecycleAPI.SuspendUser(ctx, id).Execute(); err != nil {
		return err
	}

	c.logger.Debug("suspended okta user", zap.String("okta.user.id", id))

	return nil
}

// UnsuspendUser un-suspends a user in Okta and returns them to active state
func (c *Client) UnsuspendUser(ctx context.Context, id string) error {
	c.logger.Info("un-suspending okta user", zap.String("okta.user.id", id))

	if _, err := c.client.UserLifecycleAPI.UnsuspendUser(ctx, id).Execute(); err != nil {
		return err
	}

	c.logger.Debug("un-suspended okta user", zap.String("okta.user.id", id))

	return nil
}

// EmailFromUserProfile parses the email from the okta user profile
func EmailFromUserProfile(u *okta.User) (string, error) {
	if u.Profile != nil && u.Profile.Email != nil {
		return *u.Profile.Email, nil
	}

	return "", fmt.Errorf("email not found for user %s", u.GetId()) //nolint:err113
}

// FirstNameFromUserProfile parses the firstName from the okta user profile
func FirstNameFromUserProfile(u *okta.User) (string, error) {
	if u.Profile != nil && u.Profile.FirstName.Get() != nil {
		return *u.Profile.FirstName.Get(), nil
	}

	return "", fmt.Errorf("firstName not found for user %s", u.GetId()) //nolint:err113
}

// LastNameFromUserProfile parses the lastName from the okta user profile
func LastNameFromUserProfile(u *okta.User) (string, error) {
	if u.Profile != nil && u.Profile.LastName.Get() != nil {
		return *u.Profile.LastName.Get(), nil
	}

	return "", fmt.Errorf("lastName not found for user %s", u.GetId()) //nolint:err113
}

// UserDetailsFromOktaUser parses the relevant user details from the okta user object
func UserDetailsFromOktaUser(u *okta.User) (*UserDetails, error) {
	d := &UserDetails{
		ID:     u.GetId(),
		Status: u.GetStatus(),
	}

	var firstName, lastName string

	if u.Profile != nil {
		if u.Profile.FirstName.Get() != nil {
			firstName = *u.Profile.FirstName.Get()
		}

		if u.Profile.LastName.Get() != nil {
			lastName = *u.Profile.LastName.Get()
		}

		if u.Profile.Email != nil {
			d.Email = *u.Profile.Email
		}
	}

	if firstName == "" {
		return nil, fmt.Errorf("firstName not found for user %s", u.GetId()) //nolint:err113
	}

	if lastName == "" {
		return nil, fmt.Errorf("lastName not found for user %s", u.GetId()) //nolint:err113
	}

	if d.Email == "" {
		return nil, fmt.Errorf("email not found for user %s", u.GetId()) //nolint:err113
	}

	d.Name = fmt.Sprintf("%s %s", firstName, lastName)

	return d, nil
}
