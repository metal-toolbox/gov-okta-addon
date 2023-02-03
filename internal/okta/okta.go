package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
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
	ListApplications(context.Context, *query.Params) ([]okta.App, *okta.Response, error)
	CreateApplicationGroupAssignment(context.Context, string, string, okta.ApplicationGroupAssignment) (*okta.ApplicationGroupAssignment, *okta.Response, error)
	DeleteApplicationGroupAssignment(context.Context, string, string) (*okta.Response, error)
	GetApplicationGroupAssignment(context.Context, string, string, *query.Params) (*okta.ApplicationGroupAssignment, *okta.Response, error)
	ListApplicationGroupAssignments(context.Context, string, *query.Params) ([]*okta.ApplicationGroupAssignment, *okta.Response, error)
}

// GroupInterface is the interface for managing groups in Okta
type GroupInterface interface {
	CreateGroup(context.Context, okta.Group) (*okta.Group, *okta.Response, error)
	UpdateGroup(context.Context, string, okta.Group) (*okta.Group, *okta.Response, error)
	DeleteGroup(context.Context, string) (*okta.Response, error)
	ListGroups(context.Context, *query.Params) ([]*okta.Group, *okta.Response, error)
	AddUserToGroup(context.Context, string, string) (*okta.Response, error)
	RemoveUserFromGroup(context.Context, string, string) (*okta.Response, error)
	ListGroupUsers(context.Context, string, *query.Params) ([]*okta.User, *okta.Response, error)
	ListAssignedApplicationsForGroup(context.Context, string, *query.Params) ([]okta.App, *okta.Response, error)
}

// UserInterface is the interface for managing users in Okta
type UserInterface interface {
	ClearUserSessions(context.Context, string, *query.Params) (*okta.Response, error)
	DeactivateUser(context.Context, string, *query.Params) (*okta.Response, error)
	DeactivateOrDeleteUser(context.Context, string, *query.Params) (*okta.Response, error)
	GetUser(context.Context, string) (*okta.User, *okta.Response, error)
	ListUsers(context.Context, *query.Params) ([]*okta.User, *okta.Response, error)
	SuspendUser(context.Context, string) (*okta.Response, error)
	UnsuspendUser(context.Context, string) (*okta.Response, error)
}

// LogEventInterface is the interface for getting log events from okta
type LogEventInterface interface {
	GetLogs(ctx context.Context, qp *query.Params) ([]*okta.LogEvent, *okta.Response, error)
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

	_, c, err := okta.NewClient(
		context.TODO(),
		okta.WithOrgUrl(client.url),
		okta.WithToken(client.token),
		okta.WithCache(client.cacheEnabled),
	)
	if err != nil {
		return nil, err
	}

	client.appIface = c.Application
	client.groupIface = c.Group
	client.userIface = c.User
	client.logEventIface = c.LogEvent

	return &client, nil
}
