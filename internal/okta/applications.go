package okta

import (
	"context"
	"encoding/json"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"go.uber.org/zap"
)

const (
	defaultPageLimit = 200
)

// appGithubOrg extracts the okta application id and the raw githubOrg app setting from an
// okta application.  The v6 SDK models applications as a oneOf union of typed variants with
// non-string app settings living in the settings "app" object, so we marshal the concrete
// instance and read the fields back generically.  ok is false when the application has no
// githubOrg setting.
func appGithubOrg(a okta.ListApplications200ResponseInner) (id string, githubOrg interface{}, ok bool) {
	b, err := json.Marshal(a)
	if err != nil || len(b) == 0 || string(b) == "null" {
		return "", nil, false
	}

	var parsed struct {
		ID       string `json:"id"`
		Settings struct {
			App map[string]interface{} `json:"app"`
		} `json:"settings"`
	}

	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", nil, false
	}

	v, ok := parsed.Settings.App["githubOrg"]

	return parsed.ID, v, ok
}

// GithubCloudApplications returns a map of all Okta Github cloud applications with org name as the key and the okta ID as the value
func (c *Client) GithubCloudApplications(ctx context.Context) (map[string]string, error) {
	c.logger.Debug("listing okta githubcloud application")

	applications, err := c.listApplications(ctx, "name eq \"githubcloud\"")
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

// listApplications returns all of the applications matching the given filter
func (c *Client) listApplications(ctx context.Context, filter string) ([]okta.ListApplications200ResponseInner, error) {
	apps, err := paginate(func(after string) ([]okta.ListApplications200ResponseInner, *okta.APIResponse, error) {
		req := c.client.ApplicationAPI.ListApplications(ctx).Limit(defaultPageLimit)
		if filter != "" {
			req = req.Filter(filter)
		}

		if after != "" {
			req = req.After(after)
		}

		return req.Execute()
	})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing applications", zap.Any("okta.application", apps))

	return apps, nil
}

// AssignGroupToApplication assigns a group to an okta application
func (c *Client) AssignGroupToApplication(ctx context.Context, appID, groupID string) error {
	if appID == "" || groupID == "" {
		return ErrApplicationBadParameters
	}

	c.logger.Info("adding okta application group assignments", zap.Any("okta.application.id", appID), zap.Any("okta.group.id", groupID))

	assignment, _, err := c.client.ApplicationGroupsAPI.AssignGroupToApplication(ctx, appID, groupID).Execute()
	if err != nil {
		return err
	}

	c.logger.Debug("output from application group assignment", zap.Any("okta.assignment", assignment))

	return nil
}

// RemoveApplicationGroupAssignment removes an application group assignment
func (c *Client) RemoveApplicationGroupAssignment(ctx context.Context, appID, groupID string) error {
	if appID == "" || groupID == "" {
		return ErrApplicationBadParameters
	}

	c.logger.Info("removing okta application group assignments", zap.Any("okta.application.id", appID), zap.Any("okta.group.id", groupID))

	if _, err := c.client.ApplicationGroupsAPI.UnassignApplicationFromGroup(ctx, appID, groupID).Execute(); err != nil {
		return err
	}

	c.logger.Debug("deleted application group assignment", zap.String("okta.app.id", appID), zap.String("okta.group.id", groupID))

	return nil
}

// ListGroupApplicationAssignment returns a list of the groups assigned to an application
func (c *Client) ListGroupApplicationAssignment(ctx context.Context, appID string) ([]string, error) {
	if appID == "" {
		return nil, ErrApplicationBadParameters
	}

	c.logger.Debug("listing okta application group assignments", zap.Any("okta.application.id", appID))

	assignments, err := paginate(func(after string) ([]okta.ApplicationGroupAssignment, *okta.APIResponse, error) {
		req := c.client.ApplicationGroupsAPI.ListApplicationGroupAssignments(ctx, appID).Limit(defaultPageLimit)
		if after != "" {
			req = req.After(after)
		}

		return req.Execute()
	})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing application group assignments", zap.Any("okta.assignment", assignments))

	groups := []string{}

	for _, a := range assignments {
		groups = append(groups, a.GetId())
	}

	return groups, nil
}
