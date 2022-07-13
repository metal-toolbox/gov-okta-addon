package okta

import (
	"context"
	"fmt"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

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

// UpdateGroup updates a group in Okta
func (c *Client) UpdateGroup(ctx context.Context, id, name, desc string, profile map[string]interface{}) error {
	c.logger.Info("updating Okta group",
		zap.String("okta.group.id", id),
		zap.String("okta.group.name", name),
		zap.String("okta.group.description", desc),
		zap.Any("okta.group.profile", profile),
	)

	if _, _, err := c.groupIface.UpdateGroup(ctx, id, okta.Group{
		Profile: &okta.GroupProfile{
			Name:            name,
			Description:     desc,
			GroupProfileMap: okta.GroupProfileMap(profile),
		},
	}); err != nil {
		return err
	}

	c.logger.Debug("updated okta group", zap.String("okta.group.id", id))

	return nil
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
	c.logger.Info("getting group by governor id", zap.String("governor.id", id))

	f := fmt.Sprintf("profile.governor_id eq %s", id)

	groups, _, err := c.groupIface.ListGroups(ctx, &query.Params{Search: f})
	if err != nil {
		return "", err
	}

	if len(groups) != 1 {
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
