package okta

import (
	"context"
	"fmt"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"go.uber.org/zap"
)

const (
	// GroupProfileGovernorIDKey is the map key for the governor ID in an Okta group profile
	GroupProfileGovernorIDKey = "governor_id"
)

// GroupModifierFunc modifies a an okta group response
type GroupModifierFunc func(context.Context, *okta.Group) (*okta.Group, error)

// groupProfileFields returns the name, description and additional (custom) attributes from
// an okta group profile.  The v6 SDK models the group profile as a oneOf union; okta populates
// the Active Directory variant for API responses, so that variant is checked first.  ok is
// false when the group or its profile is nil.
func groupProfileFields(group *okta.Group) (name, description string, additional map[string]interface{}, ok bool) {
	if group == nil || group.Profile == nil {
		return "", "", nil, false
	}

	if p := group.Profile.OktaActiveDirectoryGroupProfile; p != nil {
		return p.GetName(), p.GetDescription(), p.AdditionalProperties, true
	}

	if p := group.Profile.OktaUserGroupProfile; p != nil {
		return p.GetName(), p.GetDescription(), p.AdditionalProperties, true
	}

	return "", "", nil, true
}

// GroupProfileName returns the name from an okta group profile
func GroupProfileName(group *okta.Group) string {
	name, _, _, _ := groupProfileFields(group)
	return name
}

// GroupProfileDescription returns the description from an okta group profile
func GroupProfileDescription(group *okta.Group) string {
	_, description, _, _ := groupProfileFields(group)
	return description
}

// groupProfileRequest builds an okta group profile request from a name, description and
// extended schema profile.
func groupProfileRequest(name, desc string, profile map[string]interface{}) okta.AddGroupRequest {
	return okta.AddGroupRequest{
		Profile: &okta.OktaUserGroupProfile{
			Name:                 okta.PtrString(name),
			Description:          okta.PtrString(desc),
			AdditionalProperties: profile,
		},
	}
}

// CreateGroup creates a simple group in Okta with a name, description and an extended schema profile
func (c *Client) CreateGroup(ctx context.Context, name, desc string, profile map[string]interface{}) (string, error) {
	c.logger.Info("creating Okta group",
		zap.String("okta.group.name", name),
		zap.String("okta.group.description", desc),
		zap.Any("okta.group.profile", profile),
	)

	group, err := c.groupIface.CreateGroup(ctx, groupProfileRequest(name, desc, profile))
	if err != nil {
		return "", err
	}

	c.logger.Debug("created okta group", zap.String("okta.group.id", group.GetId()))

	return group.GetId(), nil
}

// UpdateGroup updates a group in Okta and returns the updated group
func (c *Client) UpdateGroup(ctx context.Context, id, name, desc string, profile map[string]interface{}) (*okta.Group, error) {
	c.logger.Info("updating Okta group",
		zap.String("okta.group.id", id),
		zap.String("okta.group.name", name),
		zap.String("okta.group.description", desc),
		zap.Any("okta.group.profile", profile),
	)

	group, err := c.groupIface.UpdateGroup(ctx, id, groupProfileRequest(name, desc, profile))
	if err != nil {
		return nil, err
	}

	c.logger.Debug("updated okta group", zap.String("okta.group.id", id))

	return group, nil
}

// DeleteGroup deletes a group in Okta
func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	c.logger.Info("deleting Okta group", zap.String("okta.group.id", id))

	if err := c.groupIface.DeleteGroup(ctx, id); err != nil {
		return err
	}

	c.logger.Debug("deleted okta group", zap.String("okta.group.id", id))

	return nil
}

// GetGroupByGovernorID gets an okta group ID from the governor id by searching for the profile field
func (c *Client) GetGroupByGovernorID(ctx context.Context, id string) (string, error) {
	c.logger.Debug("getting okta group by governor id", zap.String("governor.id", id))

	f := fmt.Sprintf("profile.governor_id eq \"%s\"", id)

	groups, err := c.groupIface.ListGroups(ctx, f)
	if err != nil {
		return "", err
	}

	if len(groups) == 0 {
		return "", ErrGroupsNotFound
	} else if len(groups) > 1 {
		return "", ErrUnexpectedGroupsCount
	}

	gid := groups[0].GetId()

	c.logger.Debug("found okta group by governor id", zap.String("governor.id", id), zap.String("okta.group.id", gid))

	return gid, nil
}

// AddGroupUser adds a user to a group by user id and group id
func (c *Client) AddGroupUser(ctx context.Context, groupID, userID string) error {
	c.logger.Info("adding user to okta group", zap.String("okta.user.id", userID), zap.String("okta.group.id", groupID))

	if err := c.groupIface.AddUserToGroup(ctx, groupID, userID); err != nil {
		return err
	}

	return nil
}

// RemoveGroupUser removes a user from a group by user id and group id
func (c *Client) RemoveGroupUser(ctx context.Context, groupID, userID string) error {
	c.logger.Info("removing user from okta group", zap.String("okta.user.id", userID), zap.String("okta.group.id", groupID))

	if err := c.groupIface.RemoveUserFromGroup(ctx, groupID, userID); err != nil {
		return err
	}

	return nil
}

// ListGroupMembership returns the full list of members of an okta group
func (c *Client) ListGroupMembership(ctx context.Context, gid string) ([]*okta.User, error) {
	c.logger.Debug("listing okta group members", zap.String("okta.group.id", gid))

	users, err := c.groupIface.ListGroupUsers(ctx, gid)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing group users", zap.Any("okta.group.users", users))

	usersResp := make([]*okta.User, len(users))
	for i := range users {
		usersResp[i] = &users[i]
	}

	return usersResp, nil
}

// ListGroupsWithModifier lists okta groups and modifies the group response with the given
// GroupModifierFunc.  If nil is returned from the GroupModifierFunc, the group will not be returned
// in the response.
func (c *Client) ListGroupsWithModifier(ctx context.Context, f GroupModifierFunc, search string) ([]*okta.Group, error) {
	c.logger.Debug("listing groups with func")

	groups, err := c.groupIface.ListGroups(ctx, search)
	if err != nil {
		return nil, err
	}

	groupResp := []*okta.Group{}

	for i := range groups {
		g := &groups[i]

		c.logger.Debug("running function on group", zap.Any("group", g))

		group, err := f(ctx, g)
		if err != nil {
			return nil, err
		}

		if group != nil {
			groupResp = append(groupResp, group)
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

	_, _, additional, _ := groupProfileFields(group)

	v, found := additional[GroupProfileGovernorIDKey]
	if !found {
		return "", ErrGroupGovernorIDNotFound
	}

	kv, ok := v.(string)
	if !ok {
		return "", ErrGroupGovernorIDNotString
	}

	if kv == "" {
		return "", ErrGroupGovernorIDNotFound
	}

	return kv, nil
}

// GroupGithubCloudApplications returns a map of Okta Github cloud applications assigned to an Okta
// group with org name as the key and the okta ID as the value
func (c *Client) GroupGithubCloudApplications(ctx context.Context, groupID string) (map[string]string, error) {
	c.logger.Debug("listing okta githubcloud application for group", zap.String("okta.group.id", groupID))

	applications, err := c.listAssignedApplicationsForGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("applications list from Okta", zap.Any("okta.apps", applications))

	apps := map[string]string{}

	for _, a := range applications {
		id, v, ok := appGithubOrg(a)
		if !ok {
			continue
		}

		org, ok := v.(string)
		if !ok {
			c.logger.Warn("okta app setting for githubOrg is not a string", zap.Any("okta.app.setting.githubOrg", v))
			continue
		}

		apps[org] = id
	}

	return apps, nil
}

// listAssignedApplicationsForGroup lists the applications that are assigned to a group ID
func (c *Client) listAssignedApplicationsForGroup(ctx context.Context, groupID string) ([]okta.ListApplications200ResponseInner, error) {
	if groupID == "" {
		return nil, ErrApplicationBadParameters
	}

	c.logger.Debug("listing okta applications assigned to group", zap.Any("okta.group.id", groupID))

	apps, err := c.groupIface.ListAssignedApplicationsForGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing application group assignments", zap.Any("okta.applications", apps))

	return apps, nil
}
