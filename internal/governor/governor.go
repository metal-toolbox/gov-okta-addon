package governor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
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

// Tokener implements the token interface
type Tokener interface {
	Token(ctx context.Context) (*oauth2.Token, error)
}

// Client is a governor API client
type Client struct {
	url                    string
	clientCredentialConfig Tokener
	logger                 *zap.Logger
	token                  *oauth2.Token
	httpClient             HTTPDoer
	authMux                sync.Mutex
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

	client.authMux.Lock()
	defer client.authMux.Unlock()

	client.token = t

	return &client, nil
}

func (c *Client) auth(ctx context.Context) (*oauth2.Token, error) {
	c.logger.Debug("authenticating governor client", zap.Any("clientcredentialconfig", c.clientCredentialConfig))
	return c.clientCredentialConfig.Token(ctx)
}

func (c *Client) refreshAuth(ctx context.Context) error {
	if c.token != nil && !time.Now().After(c.token.Expiry) {
		return nil
	}

	t, err := c.auth(ctx)
	if err != nil {
		return err
	}

	c.authMux.Lock()
	defer c.authMux.Unlock()

	c.token = t

	c.logger.Debug("refreshing governor client authentication")

	return nil
}

func (c *Client) newGovernorRequest(ctx context.Context, method, u string) (*http.Request, error) {
	if err := c.refreshAuth(ctx); err != nil {
		return nil, err
	}

	c.logger.Debug("parsing url", zap.String("url", u))

	queryURL, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("creating new http request", zap.String("url", queryURL.String()), zap.String("method", method))

	req, err := http.NewRequestWithContext(ctx, method, queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	bearer := "Bearer " + c.token.AccessToken
	req.Header.Add("Authorization", bearer)

	return req, nil
}

// Groups gets the list of groups from governor
func (c *Client) Groups(ctx context.Context) ([]*v1alpha1.Group, error) {
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
		return nil, ErrRequestNonSuccess
	}

	out := []*v1alpha1.Group{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// Group gets the details of a group from governor
func (c *Client) Group(ctx context.Context, id string) (*v1alpha1.Group, error) {
	if id == "" {
		return nil, ErrMissingGroupID
	}

	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/groups/%s", c.url, governorAPIVersion, id))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	c.logger.Debug("status code", zap.String("status code", resp.Status))

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrGroupNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestNonSuccess
	}

	out := v1alpha1.Group{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// CreateGroup creates a new group in governor
func (c *Client) CreateGroup(ctx context.Context, group *v1alpha1.GroupReq) (*v1alpha1.Group, error) {
	if group == nil {
		return nil, ErrNilGroupRequest
	}

	req, err := c.newGovernorRequest(ctx, http.MethodPost, fmt.Sprintf("%s/api/%s/groups", c.url, governorAPIVersion))
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(group)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewBuffer(b))

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, ErrRequestNonSuccess
	}

	out := v1alpha1.Group{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// DeleteGroup deletes a group from governor
func (c *Client) DeleteGroup(ctx context.Context, id string) error {
	if id == "" {
		return ErrMissingGroupID
	}

	req, err := c.newGovernorRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/api/%s/groups/%s", c.url, governorAPIVersion, id))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	c.logger.Debug("status code", zap.String("status code", resp.Status))

	if resp.StatusCode == http.StatusNotFound {
		return ErrGroupNotFound
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return ErrRequestNonSuccess
	}

	return nil
}

// AddGroupToOrganization links the group to the organization
func (c *Client) AddGroupToOrganization(ctx context.Context, groupID, orgID string) error {
	if groupID == "" {
		return ErrMissingGroupID
	}

	if orgID == "" {
		return ErrMissingOrganizationID
	}

	req, err := c.newGovernorRequest(ctx, http.MethodPut, fmt.Sprintf("%s/api/%s/groups/%s/organizations/%s", c.url, governorAPIVersion, groupID, orgID))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusNoContent {
		return ErrRequestNonSuccess
	}

	return nil
}

// RemoveGroupFromOrganization unlinks the group from the organization
func (c *Client) RemoveGroupFromOrganization(ctx context.Context, groupID, orgID string) error {
	if groupID == "" {
		return ErrMissingGroupID
	}

	if orgID == "" {
		return ErrMissingOrganizationID
	}

	req, err := c.newGovernorRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/api/%s/groups/%s/organizations/%s", c.url, governorAPIVersion, groupID, orgID))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusNoContent {
		return ErrRequestNonSuccess
	}

	return nil
}

// Organizations gets the list of organizations from governor
func (c *Client) Organizations(ctx context.Context) ([]*v1alpha1.Organization, error) {
	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/organizations", c.url, governorAPIVersion))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestNonSuccess
	}

	out := []*v1alpha1.Organization{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// Organization gets the details of an org from governor
func (c *Client) Organization(ctx context.Context, id string) (*v1alpha1.Organization, error) {
	if id == "" {
		return nil, ErrMissingOrganizationID
	}

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
		return nil, ErrRequestNonSuccess
	}

	out := v1alpha1.Organization{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// UsersQuery searches for a user in governor with the passed query
func (c *Client) UsersQuery(ctx context.Context, query map[string][]string) ([]*v1alpha1.User, error) {
	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/users", c.url, governorAPIVersion))
	if err != nil {
		return nil, err
	}

	q := url.Values{}

	for k, vals := range query {
		for _, v := range vals {
			q.Add(k, v)
		}
	}

	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestNonSuccess
	}

	out := []*v1alpha1.User{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// Users gets the list of users from governor
func (c *Client) Users(ctx context.Context) ([]*v1alpha1.User, error) {
	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/users", c.url, governorAPIVersion))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestNonSuccess
	}

	out := []*v1alpha1.User{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// User gets the details of a user from governor
func (c *Client) User(ctx context.Context, id string) (*v1alpha1.User, error) {
	if id == "" {
		return nil, ErrMissingUserID
	}

	req, err := c.newGovernorRequest(ctx, http.MethodGet, fmt.Sprintf("%s/api/%s/users/%s", c.url, governorAPIVersion, id))
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrRequestNonSuccess
	}

	out := v1alpha1.User{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// CreateUser creates a user in governor and returns the user
func (c *Client) CreateUser(ctx context.Context, user *v1alpha1.UserReq) (*v1alpha1.User, error) {
	if user == nil {
		return nil, ErrNilUserRequest
	}

	req, err := c.newGovernorRequest(ctx, http.MethodPost, fmt.Sprintf("%s/api/%s/users", c.url, governorAPIVersion))
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(bytes.NewBuffer(b))

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, ErrRequestNonSuccess
	}

	out := v1alpha1.User{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// DeleteUser deletes a user in governor
func (c *Client) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return ErrMissingUserID
	}

	req, err := c.newGovernorRequest(ctx, http.MethodDelete, fmt.Sprintf("%s/api/%s/users/%s", c.url, governorAPIVersion, id))
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return ErrRequestNonSuccess
	}

	return nil
}
