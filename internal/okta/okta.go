package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"go.uber.org/zap"
)

// Client is a client that can talk to Okta
type Client struct {
	appIface      ApplicationInterface
	groupIface    GroupInterface
	logEventIface LogEventInterface
	userIface     UserInterface
	logger        *zap.Logger

	url          string
	token        string
	cacheEnabled bool
}

// ApplicationInterface abstracts the interactions with okta applications
type ApplicationInterface interface {
	ListApplications(ctx context.Context, filter string) ([]okta.ListApplications200ResponseInner, error)
	CreateApplicationGroupAssignment(ctx context.Context, appID, groupID string) (*okta.ApplicationGroupAssignment, error)
	DeleteApplicationGroupAssignment(ctx context.Context, appID, groupID string) error
	GetApplicationGroupAssignment(ctx context.Context, appID, groupID string) (*okta.ApplicationGroupAssignment, error)
	ListApplicationGroupAssignments(ctx context.Context, appID string) ([]okta.ApplicationGroupAssignment, error)
}

// GroupInterface is the interface for managing groups in Okta
type GroupInterface interface {
	CreateGroup(ctx context.Context, group okta.AddGroupRequest) (*okta.Group, error)
	UpdateGroup(ctx context.Context, id string, group okta.AddGroupRequest) (*okta.Group, error)
	DeleteGroup(ctx context.Context, id string) error
	ListGroups(ctx context.Context, search string) ([]okta.Group, error)
	AddUserToGroup(ctx context.Context, groupID, userID string) error
	RemoveUserFromGroup(ctx context.Context, groupID, userID string) error
	ListGroupUsers(ctx context.Context, groupID string) ([]okta.User, error)
	ListAssignedApplicationsForGroup(ctx context.Context, groupID string) ([]okta.ListApplications200ResponseInner, error)
}

// UserInterface is the interface for managing users in Okta
type UserInterface interface {
	ClearUserSessions(ctx context.Context, id string) error
	DeactivateUser(ctx context.Context, id string) error
	DeactivateOrDeleteUser(ctx context.Context, id string) error
	GetUser(ctx context.Context, id string) (*okta.User, error)
	ListUsers(ctx context.Context, search string) ([]okta.User, error)
	SuspendUser(ctx context.Context, id string) error
	UnsuspendUser(ctx context.Context, id string) error
}

// LogEventInterface is the interface for getting log events from okta
type LogEventInterface interface {
	// GetLogs returns a page of log events along with the "after" cursor for the next
	// page (empty when there are no more pages).  Passing a non-empty after cursor
	// requests the next page following a previous call.
	GetLogs(ctx context.Context, since, until, after, filter string, limit int32) ([]okta.LogEvent, string, error)
}

// Option is a functional configuration option
type Option func(c *Client)

// WithURL sets the endpoint for okta
func WithURL(u string) Option {
	return func(c *Client) {
		c.url = u
	}
}

// WithToken sets the okta token
func WithToken(t string) Option {
	return func(c *Client) {
		c.token = t
	}
}

// WithCache enabled the okta client memory cache, default enabled.
func WithCache(t bool) Option {
	return func(c *Client) {
		c.cacheEnabled = t
	}
}

// WithLogger sets logger
func WithLogger(l *zap.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// NewClient returns a new Okta client
func NewClient(opts ...Option) (*Client, error) {
	client := Client{
		logger: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(&client)
	}

	config, err := okta.NewConfiguration(
		okta.WithOrgUrl(client.url),
		okta.WithToken(client.token),
		okta.WithCache(client.cacheEnabled),
	)
	if err != nil {
		return nil, err
	}

	c := okta.NewAPIClient(config)

	client.appIface = &applicationService{client: c}
	client.groupIface = &groupService{client: c}
	client.userIface = &userService{client: c}
	client.logEventIface = &logEventService{client: c}

	return &client, nil
}
