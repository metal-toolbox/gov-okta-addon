package okta

import (
	"context"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

// Client is a client that can talk to Okta
type Client struct {
	groupIface GroupInterface
	userIface  UserInterface
	logger     *zap.Logger

	url   string
	token string
}

// GroupInterface is the interface for managing groups in Okta
type GroupInterface interface {
	CreateGroup(ctx context.Context, body okta.Group) (*okta.Group, *okta.Response, error)
	UpdateGroup(ctx context.Context, groupID string, body okta.Group) (*okta.Group, *okta.Response, error)
	DeleteGroup(ctx context.Context, groupID string) (*okta.Response, error)
	ListGroups(ctx context.Context, qp *query.Params) ([]*okta.Group, *okta.Response, error)
	AddUserToGroup(ctx context.Context, groupID string, userID string) (*okta.Response, error)
	RemoveUserFromGroup(ctx context.Context, groupID string, userID string) (*okta.Response, error)
}

// UserInterface is the interface for managing users in Okta
type UserInterface interface {
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
	)
	if err != nil {
		return nil, err
	}

	client.groupIface = c.Group
	client.userIface = c.User

	return &client, nil
}
