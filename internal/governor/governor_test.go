package governor

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type mockHTTPDoer struct {
	t          *testing.T
	statusCode int
	resp       []byte
}

type mockTokener struct {
	t     *testing.T
	err   error
	token *oauth2.Token
}

func (m *mockHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	resp := http.Response{
		StatusCode: m.statusCode,
	}

	resp.Body = io.NopCloser(bytes.NewReader(m.resp))

	return &resp, nil
}

func (m *mockTokener) Token(ctx context.Context) (*oauth2.Token, error) {
	if m.err != nil {
		return nil, m.err
	}

	if m.token != nil {
		return m.token, nil
	}

	return &oauth2.Token{Expiry: time.Now().Add(5 * time.Second)}, nil
}

var (
	testGroupsResponse = []byte(`
[
    {
        "id": "70674d51-43e0-4539-b6be-b030c0f9e6aa",
        "name": "Streets and Sanitation",
        "slug": "streets-and-sanitation",
        "description": "Keepin it clean",
        "created_at": "2022-08-11T14:38:33.027346Z",
        "updated_at": "2022-08-11T14:38:33.027346Z",
        "deleted_at": null
    },
	{
        "id": "6a603c55-4787-4916-9934-70dbeb8467f7",
        "name": "Arts and Culture",
        "slug": "arts-and-culture",
        "description": "Keepin it classy",
        "created_at": "2022-08-11T14:38:33.027346Z",
        "updated_at": "2022-08-11T14:38:33.027346Z",
        "deleted_at": null
    },
	{
        "id": "6a603c55-4787-4916-9934-70dbeb8467f7",
        "name": "Budget Office",
        "slug": "budget-office",
        "description": "Keepin it real",
        "created_at": "2022-08-11T14:38:33.027346Z",
        "updated_at": "2022-08-11T14:38:33.027346Z",
        "deleted_at": null
    }
]
`)

	testGroupResponse = []byte(`
{
	"id": "8923e54d-0df6-407a-832d-2917915a3ff7",
	"name": "Parks and Public Works",
	"slug": "parks-and-public-works",
	"description": "Go out and play",
	"created_at": "2022-08-11T14:38:33.027346Z",
	"updated_at": "2022-08-11T14:38:33.027346Z",
	"deleted_at": null
}
`)

	testOrganizationResponse = []byte(`
{
	"id": "186c5a52-4421-4573-8bbf-78d85d3c277e",
	"name": "Green Party",
	"created_at": "2001-04-11T15:19:00.668476Z",
	"updated_at": "2001-04-11T15:19:00.668476Z",
	"slug": "green-party"
}
`)

	testOrganizationsResponse = []byte(`
[
	{
		"id": "186c5a52-4421-4573-8bbf-78d85d3c277e",
		"name": "Green Party",
		"created_at": "2001-04-11T15:19:00.668476Z",
		"updated_at": "2001-04-11T15:19:00.668476Z",
		"slug": "green-party"
	},
	{
		"id": "916f580b-01ae-4070-982f-95bf36124c95",
		"name": "Libertarian Party",
		"created_at": "1971-12-11T16:20:00.668476Z",
		"created_at": "1971-12-11T16:20:00.668476Z",
		"slug": "libertarian-party"
	},
	{
		"id": "613b190a-c5d2-4739-8f65-da37080b16cc",
		"name": "Independent Party",
		"created_at": "1967-07-08T07:08:00.668476Z",
		"created_at": "1967-07-08T07:08:00.668476Z",
		"slug": "independent-party"
	},
	{
		"id": "6a0594a9-6cb4-4fa1-b04f-7b2e6b8d17b8",
		"name": "Working Families Party",
		"created_at": "1998-06-19T07:13:00.668476Z",
		"created_at": "1998-06-19T07:13:00.668476Z",
		"slug": "working-families-party"
	}
]
`)

	testUsersResponse = []byte(`
[
    {
        "id": "9fd9408e-08a7-4572-b694-0541fdb80574",
        "external_id": "000089",
        "name": "Ned Lamont",
        "email": "nlamont@ct.gov",
        "login_count": 2,
        "avatar_url": "https://bit.ly/3QISvfa",
        "last_login_at": "2022-05-24T20:26:58.590207Z",
        "created_at": "2018-11-04T23:59:59.999999Z",
        "updated_at": "2018-11-04T23:59:59.999999Z",
        "github_id": 12345678,
        "github_username": "neddy"
    },
    {
        "id": "c5095b8c-9109-4b31-a7ce-ca779aae13de",
        "external_id": "000088",
        "name": "Dannel Malloy",
        "email": "dmalloy@ct.gov",
        "login_count": 7,
        "avatar_url": "https://bit.ly/3woIRpY",
        "last_login_at": "2018-12-30T18:29:27.372569Z",
        "created_at": "2010-11-04T23:59:59.999999Z",
        "updated_at": "2014-11-04T23:59:59.999999Z",
        "github_id": 11223344,
        "github_username": "dantheman"
    },
    {
        "id": "41f0e5a6-8c68-4693-a86b-37f4447fef57",
        "external_id": "000087",
        "name": "Mary Rell",
        "email": "mcrell@ct.gov",
        "login_count": 13,
        "avatar_url": "https://bit.ly/3R1BNHw",
        "last_login_at": "2010-12-28T19:52:45.539714Z",
        "created_at": "2004-07-11T00:00:00.00000Z",
        "updated_at": "2006-11-04T23:59:59.99999Z",
        "github_id": 44332211,
        "github_username": "jodi"
    }
]`)

	testUserResponse = []byte(`
{
	"id": "18d4f247-cb23-47fc-9c84-e624294027ec",
	"external_id": "000016",
	"name": "John Trumbull",
	"email": "jtrumbull@ct.gov",
	"login_count": 27,
	"avatar_url": "https://bit.ly/3pGBA0E",
	"last_login_at": "1775-08-17T20:26:58.590207Z",
	"created_at": "1769-11-04T23:59:59.999999Z",
	"updated_at": "1783-11-04T23:59:59.999999Z",
	"github_id": 10000001,
	"github_username": "johnnyTog"
}
`)
)

func TestClient_newGovernorRequest(t *testing.T) {
	testReq := func(m, u, t string) *http.Request {
		queryURL, _ := url.Parse(u)

		req, _ := http.NewRequestWithContext(context.TODO(), m, queryURL.String(), nil)
		req.Header.Add("Authorization", "Bearer "+t)

		return req
	}

	type fields struct {
		url   string
		token *oauth2.Token
	}

	tests := []struct {
		name    string
		fields  fields
		method  string
		url     string
		want    *http.Request
		wantErr bool
	}{
		{
			name: "example GET request",
			fields: fields{
				token: &oauth2.Token{
					AccessToken: "topSekret!!!!!11111",
					Expiry:      time.Now().Add(5 * time.Second),
				},
			},
			method: http.MethodGet,
			url:    "https://foo.example.com/tax",
			want:   testReq(http.MethodGet, "https://foo.example.com/tax", "topSekret!!!!!11111"),
		},
		{
			name:    "example bad method",
			method:  "BREAK IT",
			url:     "https://foo.example.com/zoning",
			wantErr: true,
		},
		{
			name:    "example bad url ",
			url:     "#&^$%^*T@#%",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    tt.fields.url,
				logger:                 zap.NewNop(),
				clientCredentialConfig: &mockTokener{t: t},
				token:                  tt.fields.token,
			}

			got, err := c.newGovernorRequest(context.TODO(), tt.method, tt.url)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Groups(t *testing.T) {
	testResp := func(r []byte) []*v1alpha1.Group {
		resp := []*v1alpha1.Group{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return resp
	}

	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		want    []*v1alpha1.Group
		wantErr bool
	}{
		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testGroupsResponse,
					statusCode: http.StatusOK,
				},
			},
			want: testResp(testGroupsResponse),
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			wantErr: true,
		},
		{
			name: "bad json response",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
					resp:       []byte(`{`),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.Groups(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Group(t *testing.T) {
	testResp := func(r []byte) *v1alpha1.Group {
		resp := v1alpha1.Group{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return &resp
	}

	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		id      string
		want    *v1alpha1.Group
		wantErr bool
	}{

		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testGroupResponse,
					statusCode: http.StatusOK,
				},
			},
			id:   "8923e54d-0df6-407a-832d-2917915a3ff7",
			want: testResp(testGroupResponse),
		},
		{
			name: "not found",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusNotFound,
				},
			},
			id:      "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			id:      "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
		{
			name: "bad json response",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
					resp:       []byte(`{`),
				},
			},
			id:      "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
		{
			name: "missing id in request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testGroupResponse,
					statusCode: http.StatusOK,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.Group(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Organization(t *testing.T) {
	testResp := func(r []byte) *v1alpha1.Organization {
		resp := v1alpha1.Organization{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return &resp
	}

	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		id      string
		want    *v1alpha1.Organization
		wantErr bool
	}{
		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testOrganizationResponse,
					statusCode: http.StatusOK,
				},
			},
			id:   "186c5a52-4421-4573-8bbf-78d85d3c277e",
			want: testResp(testOrganizationResponse),
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			id:      "186c5a52-4421-4573-8bbf-78d85d3c277e",
			wantErr: true,
		},
		{
			name: "bad json response",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
					resp:       []byte(`{`),
				},
			},
			id:      "186c5a52-4421-4573-8bbf-78d85d3c277e",
			wantErr: true,
		},
		{
			name: "missing id in request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testOrganizationResponse,
					statusCode: http.StatusOK,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.Organization(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Users(t *testing.T) {
	testResp := func(r []byte) []*v1alpha1.User {
		resp := []*v1alpha1.User{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return resp
	}

	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		want    []*v1alpha1.User
		wantErr bool
	}{
		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testUsersResponse,
					statusCode: http.StatusOK,
				},
			},
			want: testResp(testUsersResponse),
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			wantErr: true,
		},
		{
			name: "bad json response",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
					resp:       []byte(`{`),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.Users(context.TODO(), false)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_User(t *testing.T) {
	testResp := func(r []byte) *v1alpha1.User {
		resp := v1alpha1.User{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return &resp
	}

	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		id      string
		want    *v1alpha1.User
		wantErr bool
	}{
		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testUserResponse,
					statusCode: http.StatusOK,
				},
			},
			id:   "186c5a52-4421-4573-8bbf-78d85d3c277e",
			want: testResp(testUserResponse),
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			id:      "186c5a52-4421-4573-8bbf-78d85d3c277e",
			wantErr: true,
		},
		{
			name: "bad json response",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
					resp:       []byte(`{`),
				},
			},
			id:      "186c5a52-4421-4573-8bbf-78d85d3c277e",
			wantErr: true,
		},
		{
			name: "missing id in request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.User(context.TODO(), tt.id, false)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_CreateUser(t *testing.T) {
	testResp := func(r []byte) *v1alpha1.User {
		resp := v1alpha1.User{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return &resp
	}

	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		req     *v1alpha1.UserReq
		want    *v1alpha1.User
		wantErr bool
	}{
		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testUserResponse,
					statusCode: http.StatusOK,
				},
			},
			req: &v1alpha1.UserReq{
				ExternalID: "000001",
				Name:       "John Trumbull",
				Email:      "jtrumbull@ct.gov",
			},
			want: testResp(testUserResponse),
		},
		{
			name: "example request status accepted",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testUserResponse,
					statusCode: http.StatusAccepted,
				},
			},
			req: &v1alpha1.UserReq{
				ExternalID: "000001",
				Name:       "John Trumbull",
				Email:      "jtrumbull@ct.gov",
			},
			want: testResp(testUserResponse),
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			req: &v1alpha1.UserReq{
				ExternalID: "999991",
				Name:       "Test One",
				Email:      "test1@test.gov",
			},
			wantErr: true,
		},
		{
			name: "bad json response",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
					resp:       []byte(`{`),
				},
			},
			req: &v1alpha1.UserReq{
				ExternalID: "999992",
				Name:       "Test Two",
				Email:      "test2@test.gov",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.CreateUser(context.TODO(), tt.req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_DeleteUser(t *testing.T) {
	type fields struct {
		httpClient HTTPDoer
	}

	tests := []struct {
		name    string
		fields  fields
		id      string
		wantErr bool
	}{
		{
			name: "example request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testUserResponse,
					statusCode: http.StatusOK,
				},
			},
			id: "186c5a52-4421-4573-8bbf-78d85d3c277e",
		},
		{
			name: "example request accepted",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					resp:       testUserResponse,
					statusCode: http.StatusAccepted,
				},
			},
			id: "186c5a52-4421-4573-8bbf-78d85d3c277e",
		},
		{
			name: "non-success",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusInternalServerError,
				},
			},
			id:      "186c5a52-4421-4573-8bbf-78d85d3c277e",
			wantErr: true,
		},
		{
			name: "missing id in request",
			fields: fields{
				httpClient: &mockHTTPDoer{
					t:          t,
					statusCode: http.StatusOK,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.fields.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			err := c.DeleteUser(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_DeleteGroup(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		httpClient HTTPDoer
		wantErr    bool
	}{
		{
			name: "example ok request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			id: "8923e54d-0df6-407a-832d-2917915a3ff7",
		},
		{
			name: "example accepted request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusAccepted,
			},
			id: "8923e54d-0df6-407a-832d-2917915a3ff7",
		},
		{
			name: "not found",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusNotFound,
			},
			id:      "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
		{
			name: "non-success",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusInternalServerError,
			},
			id:      "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
		{
			name: "missing id in request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			err := c.DeleteGroup(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_AddGroupToOrganization(t *testing.T) {
	tests := []struct {
		name       string
		groupID    string
		orgID      string
		httpClient HTTPDoer
		wantErr    bool
	}{
		{
			name: "example ok request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "example accepted request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusAccepted,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "example no content request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusNoContent,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "not found",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusNotFound,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
			wantErr: true,
		},
		{
			name: "non-success",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusInternalServerError,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
			wantErr: true,
		},
		{
			name: "missing groupID in request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			wantErr: true,
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "missing orgID in request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			err := c.AddGroupToOrganization(context.TODO(), tt.groupID, tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_RemoveGroupFromOrganization(t *testing.T) {
	tests := []struct {
		name       string
		groupID    string
		orgID      string
		httpClient HTTPDoer
		wantErr    bool
	}{
		{
			name: "example ok request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "example accepted request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusAccepted,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "example no content request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusNoContent,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "not found",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusNotFound,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
			wantErr: true,
		},
		{
			name: "non-success",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusInternalServerError,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
			wantErr: true,
		},
		{
			name: "missing groupID in request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			wantErr: true,
			orgID:   "bde11bd6-66b7-4f1b-9d4b-0a8a86b2e097",
		},
		{
			name: "missing orgID in request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testGroupResponse,
				statusCode: http.StatusOK,
			},
			groupID: "8923e54d-0df6-407a-832d-2917915a3ff7",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			err := c.RemoveGroupFromOrganization(context.TODO(), tt.groupID, tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_Organizations(t *testing.T) {
	testResp := func(r []byte) []*v1alpha1.Organization {
		resp := []*v1alpha1.Organization{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return resp
	}

	tests := []struct {
		name       string
		httpClient HTTPDoer
		want       []*v1alpha1.Organization
		wantErr    bool
	}{
		{
			name: "example request",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testOrganizationsResponse,
				statusCode: http.StatusOK,
			},
			want: testResp(testOrganizationsResponse),
		},
		{
			name: "example request status accepted",
			httpClient: &mockHTTPDoer{
				t:          t,
				resp:       testOrganizationsResponse,
				statusCode: http.StatusOK,
			},
			want: testResp(testOrganizationsResponse),
		},
		{
			name: "non-success",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusInternalServerError,
			},
			wantErr: true,
		},
		{
			name: "bad json response",
			httpClient: &mockHTTPDoer{
				t:          t,
				statusCode: http.StatusOK,
				resp:       []byte(`{`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				url:                    "https://the.gov/",
				logger:                 zap.NewNop(),
				httpClient:             tt.httpClient,
				clientCredentialConfig: &mockTokener{t: t},
				token:                  &oauth2.Token{AccessToken: "topSekret"},
			}
			got, err := c.Organizations(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
