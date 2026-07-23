package okta

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/okta/okta-sdk-golang/v6/okta"
)

// The *service types adapt the okta v6 fluent API clients to the local interfaces used
// by Client.  They own request construction and response pagination so that the rest of
// the package (and its test mocks) can deal in plain slices instead of *okta.APIResponse.

// nextAfter parses the "after" pagination cursor from an okta APIResponse.  It returns an
// empty string when there are no more pages.
func nextAfter(resp *okta.APIResponse) string {
	if resp == nil || !resp.HasNextPage() {
		return ""
	}

	u, err := url.Parse(resp.NextPage())
	if err != nil {
		return ""
	}

	return u.Query().Get("after")
}

type applicationService struct {
	client *okta.APIClient
}

func (s *applicationService) ListApplications(ctx context.Context, filter string) ([]okta.ListApplications200ResponseInner, error) {
	req := s.client.ApplicationAPI.ListApplications(ctx).Limit(defaultPageLimit)
	if filter != "" {
		req = req.Filter(filter)
	}

	apps, resp, err := req.Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var page []okta.ListApplications200ResponseInner

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		apps = append(apps, page...)
	}

	return apps, nil
}

func (s *applicationService) CreateApplicationGroupAssignment(ctx context.Context, appID, groupID string) (*okta.ApplicationGroupAssignment, error) {
	assignment, _, err := s.client.ApplicationGroupsAPI.AssignGroupToApplication(ctx, appID, groupID).Execute()
	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (s *applicationService) DeleteApplicationGroupAssignment(ctx context.Context, appID, groupID string) error {
	_, err := s.client.ApplicationGroupsAPI.UnassignApplicationFromGroup(ctx, appID, groupID).Execute()

	return err
}

func (s *applicationService) GetApplicationGroupAssignment(ctx context.Context, appID, groupID string) (*okta.ApplicationGroupAssignment, error) {
	assignment, _, err := s.client.ApplicationGroupsAPI.GetApplicationGroupAssignment(ctx, appID, groupID).Execute()
	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (s *applicationService) ListApplicationGroupAssignments(ctx context.Context, appID string) ([]okta.ApplicationGroupAssignment, error) {
	assignments, resp, err := s.client.ApplicationGroupsAPI.ListApplicationGroupAssignments(ctx, appID).Limit(defaultPageLimit).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var page []okta.ApplicationGroupAssignment

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		assignments = append(assignments, page...)
	}

	return assignments, nil
}

type groupService struct {
	client *okta.APIClient
}

func (s *groupService) CreateGroup(ctx context.Context, group okta.AddGroupRequest) (*okta.Group, error) {
	g, _, err := s.client.GroupAPI.AddGroup(ctx).Group(group).Execute()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *groupService) UpdateGroup(ctx context.Context, id string, group okta.AddGroupRequest) (*okta.Group, error) {
	g, _, err := s.client.GroupAPI.ReplaceGroup(ctx, id).Group(group).Execute()
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *groupService) DeleteGroup(ctx context.Context, id string) error {
	_, err := s.client.GroupAPI.DeleteGroup(ctx, id).Execute()

	return err
}

func (s *groupService) ListGroups(ctx context.Context, search string) ([]okta.Group, error) {
	req := s.client.GroupAPI.ListGroups(ctx).Limit(defaultPageLimit)
	if search != "" {
		req = req.Search(search)
	}

	groups, resp, err := req.Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var page []okta.Group

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		groups = append(groups, page...)
	}

	return groups, nil
}

func (s *groupService) AddUserToGroup(ctx context.Context, groupID, userID string) error {
	_, err := s.client.GroupAPI.AssignUserToGroup(ctx, groupID, userID).Execute()

	return err
}

func (s *groupService) RemoveUserFromGroup(ctx context.Context, groupID, userID string) error {
	_, err := s.client.GroupAPI.UnassignUserFromGroup(ctx, groupID, userID).Execute()

	return err
}

func (s *groupService) ListGroupUsers(ctx context.Context, groupID string) ([]okta.User, error) {
	users, resp, err := s.client.GroupAPI.ListGroupUsers(ctx, groupID).Limit(defaultPageLimit).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var page []okta.User

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		users = append(users, page...)
	}

	return users, nil
}

func (s *groupService) ListAssignedApplicationsForGroup(ctx context.Context, groupID string) ([]okta.ListApplications200ResponseInner, error) {
	apps, resp, err := s.client.GroupAPI.ListAssignedApplicationsForGroup(ctx, groupID).Limit(defaultPageLimit).Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var page []okta.ListApplications200ResponseInner

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		apps = append(apps, page...)
	}

	return apps, nil
}

type userService struct {
	client *okta.APIClient
}

func (s *userService) ClearUserSessions(ctx context.Context, id string) error {
	_, err := s.client.UserSessionsAPI.RevokeUserSessions(ctx, id).Execute()

	return err
}

func (s *userService) DeactivateUser(ctx context.Context, id string) error {
	_, err := s.client.UserLifecycleAPI.DeactivateUser(ctx, id).Execute()

	return err
}

func (s *userService) DeactivateOrDeleteUser(ctx context.Context, id string) error {
	_, err := s.client.UserAPI.DeleteUser(ctx, id).Execute()

	return err
}

func (s *userService) GetUser(ctx context.Context, id string) (*okta.User, error) {
	singleton, _, err := s.client.UserAPI.GetUser(ctx, id).Execute()
	if err != nil {
		return nil, err
	}

	return userFromGetSingleton(singleton)
}

func (s *userService) ListUsers(ctx context.Context, search string) ([]okta.User, error) {
	req := s.client.UserAPI.ListUsers(ctx).Limit(defaultPageLimit)
	if search != "" {
		req = req.Search(search)
	}

	users, resp, err := req.Execute()
	if err != nil {
		return nil, err
	}

	for resp.HasNextPage() {
		var page []okta.User

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		users = append(users, page...)
	}

	return users, nil
}

func (s *userService) SuspendUser(ctx context.Context, id string) error {
	_, err := s.client.UserLifecycleAPI.SuspendUser(ctx, id).Execute()

	return err
}

func (s *userService) UnsuspendUser(ctx context.Context, id string) error {
	_, err := s.client.UserLifecycleAPI.UnsuspendUser(ctx, id).Execute()

	return err
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

type logEventService struct {
	client *okta.APIClient
}

func (s *logEventService) GetLogs(ctx context.Context, since, until, after, filter string, limit int32) ([]okta.LogEvent, string, error) {
	req := s.client.SystemLogAPI.ListLogEvents(ctx)

	if since != "" {
		req = req.Since(since)
	}

	if until != "" {
		req = req.Until(until)
	}

	if after != "" {
		req = req.After(after)
	}

	if filter != "" {
		req = req.Filter(filter)
	}

	if limit > 0 {
		req = req.Limit(limit)
	}

	events, resp, err := req.Execute()
	if err != nil {
		return nil, "", err
	}

	return events, nextAfter(resp), nil
}
