package okta

import (
	"context"
	"fmt"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

const (
	// GroupProfileGovernorIDKey is the map key for the governor ID in an Okta group profile
	GroupProfileGovernorIDKey = "governor_id"
)

// GroupModifierFunc modifies a an okta group response
type GroupModifierFunc func(context.Context, *okta.Group) (*okta.Group, error)

// CreateGroup creates a simple group in Okta with a name, description and an extended schema profile
func (c *Client) CreateGroup(ctx context.Context, name, desc string, profile map[string]interface{}) (string, error) {
	c.logger.Info("creating Okta group",
		zap.String("okta.group.name", name),
		zap.String("okta.group.description", desc),
		zap.Any("okta.group.profile", profile),
	)

	group, _, err := c.groupIface.CreateGroup(ctx, okta.Group{
		Profile: &okta.GroupProfile{
			Name:            name,
			Description:     desc,
			GroupProfileMap: okta.GroupProfileMap(profile),
		},
	})
	if err != nil {
		return "", err
	}

	c.logger.Debug("created okta group", zap.String("okta.group.id", group.Id))

	return group.Id, nil
}

// UpdateGroup updates a group in Okta and returns the updated group
func (c *Client) UpdateGroup(ctx context.Context, id, name, desc string, profile map[string]interface{}) (*okta.Group, error) {
	c.logger.Info("updating Okta group",
		zap.String("okta.group.id", id),
		zap.String("okta.group.name", name),
		zap.String("okta.group.description", desc),
		zap.Any("okta.group.profile", profile),
	)

	group, _, err := c.groupIface.UpdateGroup(ctx, id, okta.Group{
		Profile: &okta.GroupProfile{
			Name:            name,
			Description:     desc,
			GroupProfileMap: okta.GroupProfileMap(profile),
		},
	})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("updated okta group", zap.String("okta.group.id", id))

	return group, nil
}

// DeleteGroup deletes a group in Okta
func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	c.logger.Info("deleting Okta group", zap.String("okta.group.id", id))

	if _, err := c.groupIface.DeleteGroup(ctx, id); err != nil {
		return err
	}

	c.logger.Debug("deleted okta group", zap.String("okta.group.id", id))

	return nil
}

// GetGroupByGovernorID gets an okta group ID from the governor id by searching for the profile field
func (c *Client) GetGroupByGovernorID(ctx context.Context, id string) (string, error) {
	c.logger.Debug("getting okta group by governor id", zap.String("governor.id", id))

	f := fmt.Sprintf("profile.governor_id eq \"%s\"", id)

	groups, _, err := c.groupIface.ListGroups(ctx, &query.Params{Search: f})
	if err != nil {
		return "", err
	}

	if len(groups) == 0 {
		return "", ErrGroupsNotFound
	} else if len(groups) > 1 {
		return "", ErrUnexpectedGroupsCount
	}

	gid := groups[0].Id

	c.logger.Debug("found okta group by governor id", zap.String("governor.id", id), zap.String("okta.group.id", gid))

	return gid, nil
}

// AddGroupUser adds a user to a group by user id and group id
func (c *Client) AddGroupUser(ctx context.Context, groupID, userID string) error {
	c.logger.Info("adding user to okta group", zap.String("okta.user.id", userID), zap.String("okta.group.id", groupID))

	if _, err := c.groupIface.AddUserToGroup(ctx, groupID, userID); err != nil {
		return err
	}

	return nil
}

// RemoveGroupUser removes a user from a group by user id and group id
func (c *Client) RemoveGroupUser(ctx context.Context, groupID, userID string) error {
	c.logger.Info("removing user from okta group", zap.String("okta.user.id", userID), zap.String("okta.group.id", groupID))

	if _, err := c.groupIface.RemoveUserFromGroup(ctx, groupID, userID); err != nil {
		return err
	}

	return nil
}

// ListGroupMembership returns the full list of members of an okta group
func (c *Client) ListGroupMembership(ctx context.Context, gid string) ([]*okta.User, error) {
	c.logger.Debug("listing okta group members", zap.String("okta.group.id", gid))

	users, resp, err := c.groupIface.ListGroupUsers(ctx, gid, &query.Params{Limit: defaultPageLimit})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing group users", zap.Any("okta.group.users", users))

	usersResp := users

	for resp.HasNextPage() {
		resp, err = resp.Next(ctx, &users)
		if err != nil {
			return nil, err
		}

		usersResp = append(usersResp, users...)
	}

	return usersResp, nil
}

// ListGroupsWithModifier lists okta groups and modifies the group response with the given
// GroupModifierFunc.  If nil is returned from the GroupModifierFunc, the group will not be returned
// in the response.
func (c *Client) ListGroupsWithModifier(ctx context.Context, f GroupModifierFunc, q *query.Params) ([]*okta.Group, error) {
	c.logger.Debug("listing groups with func")

	groups, resp, err := c.groupIface.ListGroups(ctx, q)
	if err != nil {
		return nil, err
	}

	groupResp := []*okta.Group{}

	for _, g := range groups {
		c.logger.Debug("running function on group", zap.Any("group", g))

		group, err := f(ctx, g)
		if err != nil {
			return nil, err
		}

		if group != nil {
			groupResp = append(groupResp, group)
		}
	}

	for resp.HasNextPage() {
		nextPage := []*okta.Group{}

		resp, err = resp.Next(ctx, &nextPage)
		if err != nil {
			return nil, err
		}

		for _, g := range nextPage {
			c.logger.Debug("running function on group", zap.Any("group", g))

			group, err := f(ctx, g)
			if err != nil {
				return nil, err
			}

			if group != nil {
				groupResp = append(groupResp, group)
			}
		}
	}

	c.logger.Debug("returning list of groups", zap.Int("num.okta.groups", len(groupResp)))

	return groupResp, nil
}

// GroupGovernorID gets the governor group id from the okta group profile
func GroupGovernorID(group *okta.Group) (string, error) {
	if group == nil {
		return "", ErrBadOktaGroupParameter
	}

	if group.Profile == nil {
		return "", ErrNilGroupProfile
	}

	for k, v := range group.Profile.GroupProfileMap {
		if k == GroupProfileGovernorIDKey {
			kv, ok := v.(string)
			if !ok {
				return "", ErrGroupGovernorIDNotString
			}

			if kv == "" {
				return "", ErrGroupGovernorIDNotFound
			}

			return kv, nil
		}
	}

	return "", ErrGroupGovernorIDNotFound
}

// GroupGithubCloudApplications returns a map of Okta Github cloud applications assigned to an Okta
// group with org name as the key and the okta ID as the value
func (c *Client) GroupGithubCloudApplications(ctx context.Context, groupID string) (map[string]string, error) {
	c.logger.Debug("listing okta githubcloud application for group", zap.String("okta.group.id", groupID))

	applications, err := c.listAssignedApplicationsForGroup(ctx, groupID, &query.Params{Filter: "name eq \"githubcloud\"", Limit: defaultPageLimit})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("applications list from Okta", zap.Any("okta.apps", applications))

	apps := map[string]string{}

	for _, a := range applications {
		app, ok := a.(*okta.Application)
		if !ok {
			continue
		}

		// trudge through the app settings looking for the github org
		if app.Settings != nil && app.Settings.App != nil {
			for k, v := range *app.Settings.App {
				c.logger.Debug("okta app setting", zap.String("okta.app.setting.key", k), zap.Any("okta.app.setting.value", v))

				if k == "githubOrg" {
					org, ok := v.(string)
					if !ok {
						c.logger.Warn("okta app setting for githubOrg is not a string", zap.Any("okta.app.settings", *app.Settings.App))
						break
					}

					apps[org] = app.Id
				}
			}
		}
	}

	return apps, nil
}

// listAssignedApplicationsForGroup lists the applications that are assigned to a group ID
func (c *Client) listAssignedApplicationsForGroup(ctx context.Context, groupID string, qp *query.Params) ([]okta.App, error) {
	if groupID == "" {
		return nil, ErrApplicationBadParameters
	}

	c.logger.Debug("listing okta applications assigned to group", zap.Any("okta.group.id", groupID))

	apps, resp, err := c.groupIface.ListAssignedApplicationsForGroup(ctx, groupID, qp)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing application group assignments", zap.Any("okta.applications", apps))

	list := make([]okta.App, len(apps))
	copy(list, apps)

	for resp.HasNextPage() {
		resp, err = resp.Next(ctx, &apps)
		if err != nil {
			return nil, err
		}

		list = append(list, apps...)
	}

	return list, nil
}
