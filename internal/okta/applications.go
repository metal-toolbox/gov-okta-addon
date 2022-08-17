package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

const (
	defaultPageLimit = 200
)

// GithubCloudApplications returns a map of all Okta Github cloud applications with org name as the key and the okta ID as the value
func (c *Client) GithubCloudApplications(ctx context.Context) (map[string]string, error) {
	applications, err := c.listApplications(ctx, &query.Params{Filter: "name eq \"githubcloud\"", Limit: defaultPageLimit})
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

// listApplications returns all of the applications modified by the query parameters
func (c *Client) listApplications(ctx context.Context, qp *query.Params) ([]okta.App, error) {
	apps, resp, err := c.appIface.ListApplications(ctx, qp)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing applications", zap.Any("okta.application", apps), zap.Any("response", resp))

	list := make([]okta.App, len(apps))
	copy(list, apps)

	for {
		if !resp.HasNextPage() {
			break
		}

		resp, err = resp.Next(ctx, &apps)
		if err != nil {
			return nil, err
		}

		list = append(list, apps...)
	}

	return list, nil
}

// AssignGroupToApplication assigns a group to an okta application
func (c *Client) AssignGroupToApplication(ctx context.Context, appID, groupID string) error {
	assignment, _, err := c.appIface.CreateApplicationGroupAssignment(ctx, appID, groupID, okta.ApplicationGroupAssignment{})
	if err != nil {
		return err
	}

	c.logger.Debug("output from application group assignment", zap.Any("okta.assignment", assignment))

	return nil
}

// RemoveApplicationGroupAssignment removes an application group assignment
func (c *Client) RemoveApplicationGroupAssignment(ctx context.Context, appID, groupID string) error {
	if _, err := c.appIface.DeleteApplicationGroupAssignment(ctx, appID, groupID); err != nil {
		return err
	}

	c.logger.Debug("deleted application group assignment", zap.String("okta.app.id", appID), zap.String("okta.group.id", groupID))

	return nil
}

// GetGroupApplicationAssignment gets details about an application group assignment
func (c *Client) GetGroupApplicationAssignment(ctx context.Context, appID, groupID string) error {
	assignment, _, err := c.appIface.GetApplicationGroupAssignment(ctx, appID, groupID, &query.Params{})
	if err != nil {
		return err
	}

	c.logger.Debug("output from application group assignment", zap.Any("okta.assignment", assignment))

	return nil
}

// ListGroupApplicationAssignment returns a list of the groups assigned to an application
func (c *Client) ListGroupApplicationAssignment(ctx context.Context, appID string) ([]string, error) {
	groups := []string{}

	assignments, resp, err := c.appIface.ListApplicationGroupAssignments(ctx, appID, &query.Params{Limit: defaultPageLimit})
	if err != nil {
		return nil, err
	}

	c.logger.Debug("output from listing application group assignments", zap.Any("okta.assignment", assignments))

	for _, a := range assignments {
		groups = append(groups, a.Id)
	}

	for {
		if !resp.HasNextPage() {
			break
		}

		resp, err = resp.Next(ctx, &assignments)
		if err != nil {
			return nil, err
		}

		for _, a := range assignments {
			groups = append(groups, a.Id)
		}
	}

	return groups, nil
}
