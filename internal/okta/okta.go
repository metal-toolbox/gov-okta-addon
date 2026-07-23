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

// paginate accumulates all pages of a paginated okta list endpoint.  fetch is invoked once
// per page with the "after" cursor for that page ("" on the first call) and must issue the
// request with the caller's context, so cancellation/deadlines propagate across pages.  We
// re-issue the request builder with .After() rather than using the SDK's resp.Next() helper,
// which reuses the client's background context instead of the caller's.
func paginate[T any](fetch func(after string) ([]T, *okta.APIResponse, error)) ([]T, error) {
	var all []T

	after := ""

	for {
		page, resp, err := fetch(after)
		if err != nil {
			return nil, err
		}

		all = append(all, page...)

		after = nextAfter(resp)
		if after == "" {
			break
		}
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
