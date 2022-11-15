package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

// Client is a client that can talk to Okta
type Client struct {
	appIface   ApplicationInterface
	groupIface GroupInterface
	userIface  UserInterface
	logger     *zap.Logger

	url          string
	token        string
	cacheEnabled bool
}

// ApplicationInterface abstracts the interactions with okta applications
type ApplicationInterface interface {
	ListApplications(context.Context, *query.Params) ([]okta.App, *okta.Response, error)
	CreateApplicationGroupAssignment(ctx context.Context, appID, groupID string, body okta.ApplicationGroupAssignment) (*okta.ApplicationGroupAssignment, *okta.Response, error)
	DeleteApplicationGroupAssignment(ctx context.Context, appID, groupID string) (*okta.Response, error)
	GetApplicationGroupAssignment(ctx context.Context, appID, groupID string, qp *query.Params) (*okta.ApplicationGroupAssignment, *okta.Response, error)
	ListApplicationGroupAssignments(ctx context.Context, appID string, qp *query.Params) ([]*okta.ApplicationGroupAssignment, *okta.Response, error)
}

// GroupInterface is the interface for managing groups in Okta
type GroupInterface interface {
	CreateGroup(ctx context.Context, body okta.Group) (*okta.Group, *okta.Response, error)
	UpdateGroup(ctx context.Context, groupID string, body okta.Group) (*okta.Group, *okta.Response, error)
	DeleteGroup(ctx context.Context, groupID string) (*okta.Response, error)
	ListGroups(ctx context.Context, qp *query.Params) ([]*okta.Group, *okta.Response, error)
	AddUserToGroup(ctx context.Context, groupID, userID string) (*okta.Response, error)
	RemoveUserFromGroup(ctx context.Context, groupID, userID string) (*okta.Response, error)
	ListGroupUsers(ctx context.Context, groupID string, qp *query.Params) ([]*okta.User, *okta.Response, error)
	ListAssignedApplicationsForGroup(ctx context.Context, groupID string, qp *query.Params) ([]okta.App, *okta.Response, error)
}

// UserInterface is the interface for managing users in Okta
type UserInterface interface {
	DeactivateUser(ctx context.Context, userID string, qp *query.Params) (*okta.Response, error)
	DeactivateOrDeleteUser(ctx context.Context, userID string, qp *query.Params) (*okta.Response, error)
	GetUser(ctx context.Context, userID string) (*okta.User, *okta.Response, error)
	ListUsers(ctx context.Context, qp *query.Params) ([]*okta.User, *okta.Response, error)
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

	return &client, nil
}
