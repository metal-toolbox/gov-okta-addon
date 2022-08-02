package governor

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/goccy/go-json"
	"go.equinixmetal.net/governor/pkg/api/v1alpha"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	governorAPIVersion = "v1alpha1"
)

// HTTPDoer implements the standard http.Client interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a governor API client
type Client struct {
	url                    string
	clientCredentialConfig *clientcredentials.Config
	logger                 *zap.Logger
	token                  *oauth2.Token
	httpClient             HTTPDoer
}

// Option is a functional configuration option
type Option func(r *Client)

// WithURL sets the governor API URL
func WithURL(u string) Option {
	return func(r *Client) {
		r.url = u
	}
}

// WithClientCredentialConfig sets the oauth client credential config
func WithClientCredentialConfig(c *clientcredentials.Config) Option {
	return func(r *Client) {
		r.clientCredentialConfig = c
	}
}

// WithLogger sets logger
func WithLogger(l *zap.Logger) Option {
	return func(r *Client) {
		r.logger = l
	}
}

// WithHTTPClient overrides the default http client
func WithHTTPClient(c HTTPDoer) Option {
	return func(r *Client) {
		r.httpClient = c
	}
}

// NewClient returns a new governor client
func NewClient(opts ...Option) (*Client, error) {
	client := Client{
		logger:     zap.NewNop(),
		httpClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(&client)
	}

	t, err := client.auth(context.TODO())
	if err != nil {
		return nil, err
	}

	client.token = t

	return &client, nil
}

func (c *Client) auth(ctx context.Context) (*oauth2.Token, error) {
	c.logger.Debug("authenticating governor client", zap.Any("clientcredentialconfig", c.clientCredentialConfig))
	return c.clientCredentialConfig.Token(ctx)
}

func (c *Client) refreshAuth(ctx context.Context) error {
	c.logger.Debug("refreshing governor client authentication")
	// TODO
	return nil
}

func (c *Client) newGovernorRequest(ctx context.Context, method, u string) (*http.Request, error) {
	if err := c.refreshAuth(ctx); err != nil {
		return nil, err
	}

	queryURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	bearer := "Bearer " + c.token.AccessToken
	req.Header.Add("Authorization", bearer)

	return req, nil
}

// Groups gets the list of groups from governor
func (c *Client) Groups(ctx context.Context) ([]*v1alpha.Group, error) {
	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/groups", c.url, governorAPIVersion))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-success listing groups (%s)", resp.Status)
	}

	out := []*v1alpha.Group{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// Group gets the details of a group from governor
func (c *Client) Group(ctx context.Context, id string) (*v1alpha.Group, error) {
	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/groups/%s", c.url, governorAPIVersion, id))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-success getting group from governor (%s)", resp.Status)
	}

	out := v1alpha.Group{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Organization gets the details of an org from governor
func (c *Client) Organization(ctx context.Context, id string) (*v1alpha.Organization, error) {
	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/organizations/%s", c.url, governorAPIVersion, id))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-success getting org from governor (%s)", resp.Status)
	}

	out := v1alpha.Organization{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
