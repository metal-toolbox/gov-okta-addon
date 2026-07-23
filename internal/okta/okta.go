package okta

import (
	"net/url"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"go.uber.org/zap"
)

// Client is a client that can talk to Okta
type Client struct {
	client *okta.APIClient
	logger *zap.Logger

	url          string
	token        string
	cacheEnabled bool
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

	client.client = okta.NewAPIClient(config)

	return &client, nil
}

// collectPages accumulates all pages of a paginated okta list endpoint.  It is meant to be
// called directly with the result of an Execute() call, e.g.
//
//	collectPages(c.client.GroupAPI.ListGroups(ctx).Execute())
func collectPages[T any](first []T, resp *okta.APIResponse, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}

	all := first

	for resp.HasNextPage() {
		var page []T

		resp, err = resp.Next(&page)
		if err != nil {
			return nil, err
		}

		all = append(all, page...)
	}

	return all, nil
}

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
