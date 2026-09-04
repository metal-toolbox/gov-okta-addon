package okta

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"go.uber.org/zap"
)

const (
	defaultPageLimit = 200
)

// appSummary is a minimal, permissive view of an okta application: just the fields we
// actually need (id and the "app" settings bag that carries githubOrg).  We decode into this
// instead of the SDK's typed oneOf union (okta.ListApplications200ResponseInner) because that
// union enforces required sub-fields (e.g. SamlApplicationSettingsSignOn.allowMultipleAcsEndpoints)
// that some real-world Okta apps omit from their API response, which makes the SDK fail to
// decode the entire page. See decodeAppSummaries.
type appSummary struct {
	ID       string `json:"id"`
	Settings struct {
		App map[string]interface{} `json:"app"`
	} `json:"settings"`
}

// appGithubOrg extracts the okta application id and the raw githubOrg app setting from an
// okta application summary.  ok is false when the application has no githubOrg setting.
func appGithubOrg(a appSummary) (id string, githubOrg interface{}, ok bool) {
	v, ok := a.Settings.App["githubOrg"]
	return a.ID, v, ok
}

// decodeAppSummaries decodes a raw applications list response body into appSummary values.
// It's deliberately more permissive than the SDK's typed models (see appSummary) so it
// succeeds even when the SDK's own oneOf decode would reject the payload.
func decodeAppSummaries(body []byte) ([]appSummary, error) {
	var apps []appSummary

	if err := json.Unmarshal(body, &apps); err != nil {
		return nil, err
	}

	return apps, nil
}

// toAppSummaries converts SDK-decoded applications to appSummary by round-tripping through
// JSON, since the concrete app fields we need (id, settings.app) are common to every variant
// of the oneOf union.
func toAppSummaries(apps []okta.ListApplications200ResponseInner) ([]appSummary, error) {
	b, err := json.Marshal(apps)
	if err != nil {
		return nil, err
	}

	return decodeAppSummaries(b)
}

// listApplicationsFallback recovers from an okta SDK decode error by parsing the raw response
// body directly into appSummary values, working around the SDK's overly strict oneOf decoding
// (see appSummary).  It returns ok=false if the error isn't a decode error we can recover from.
func listApplicationsFallback(err error) (apps []appSummary, ok bool) {
	var oerr *okta.GenericOpenAPIError
	if !errors.As(err, &oerr) || len(oerr.Body()) == 0 {
		return nil, false
	}

	apps, decodeErr := decodeAppSummaries(oerr.Body())
	if decodeErr != nil {
		return nil, false
	}

	return apps, true
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
func (c *Client) listApplications(ctx context.Context, filter string) ([]appSummary, error) {
	apps, err := paginate(func(after string) ([]appSummary, *okta.APIResponse, error) {
		req := c.client.ApplicationAPI.ListApplications(ctx).Limit(defaultPageLimit)
		if filter != "" {
			req = req.Filter(filter)
		}

		if after != "" {
			req = req.After(after)
		}

		raw, resp, err := req.Execute()
		if err != nil {
			if fallback, ok := listApplicationsFallback(err); ok {
				c.logger.Warn("recovered from okta application decode error using raw response fallback", zap.Error(err))
				return fallback, resp, nil
			}

			return nil, resp, err
		}

		summaries, err := toAppSummaries(raw)
		if err != nil {
			return nil, resp, err
		}

		return summaries, resp, nil
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
