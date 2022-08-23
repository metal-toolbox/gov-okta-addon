package okta

import (
	"context"
	"errors"
	"testing"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockApplicationClient struct {
	t                   *testing.T
	err                 error
	resp                *okta.Response
	apps                []okta.App
	appGroupAssignments []*okta.ApplicationGroupAssignment
}

func (m *mockApplicationClient) ListApplications(context.Context, *query.Params) ([]okta.App, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.apps, m.resp, nil
}

func (m *mockApplicationClient) CreateApplicationGroupAssignment(ctx context.Context, appID string, groupID string, body okta.ApplicationGroupAssignment) (*okta.ApplicationGroupAssignment, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return nil, m.resp, nil
}

func (m *mockApplicationClient) DeleteApplicationGroupAssignment(ctx context.Context, appID string, groupID string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockApplicationClient) GetApplicationGroupAssignment(ctx context.Context, appID string, groupID string, qp *query.Params) (*okta.ApplicationGroupAssignment, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return nil, m.resp, nil
}

func (m *mockApplicationClient) ListApplicationGroupAssignments(ctx context.Context, appID string, qp *query.Params) ([]*okta.ApplicationGroupAssignment, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.appGroupAssignments, m.resp, nil
}

type otherApplication struct{}

func (o *otherApplication) IsApplicationInstance() bool {
	return false
}

func TestClient_AssignGroupToApplication(t *testing.T) {
	tests := []struct {
		name    string
		appID   string
		groupID string
		err     error
		wantErr bool
	}{
		{
			name:    "example",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
		},
		{
			name:    "error",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				appIface: &mockApplicationClient{
					t:   t,
					err: tt.err,
				},
			}
			err := c.AssignGroupToApplication(context.TODO(), tt.appID, tt.groupID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_RemoveApplicationGroupAssignment(t *testing.T) {
	tests := []struct {
		name    string
		appID   string
		groupID string
		err     error
		wantErr bool
	}{
		{
			name:    "example",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
		},
		{
			name:    "error",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				appIface: &mockApplicationClient{
					t:   t,
					err: tt.err,
				},
			}
			err := c.RemoveApplicationGroupAssignment(context.TODO(), tt.appID, tt.groupID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_ListGroupApplicationAssignment(t *testing.T) {
	tests := []struct {
		name        string
		appID       string
		err         error
		assignments []*okta.ApplicationGroupAssignment
		resp        *okta.Response
		want        []string
		wantErr     bool
	}{
		{
			name:  "example",
			appID: "47819d20-70e5-4ab9-b008-898be42adde7",
			assignments: []*okta.ApplicationGroupAssignment{
				{
					Id: "group-001",
				},
				{
					Id: "group-002",
				},
			},
			resp: &okta.Response{},
			want: []string{"group-001", "group-002"},
		},
		{
			name:    "api error",
			appID:   "47819d20-70e5-4ab9-b008-898be42adde7",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				appIface: &mockApplicationClient{
					t:                   t,
					err:                 tt.err,
					appGroupAssignments: tt.assignments,
					resp:                tt.resp,
				},
			}
			got, err := c.ListGroupApplicationAssignment(context.TODO(), tt.appID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_listApplications(t *testing.T) {
	tests := []struct {
		name    string
		qp      *query.Params
		resp    *okta.Response
		err     error
		apps    []okta.App
		want    []okta.App
		wantErr bool
	}{
		{
			name: "example",
			resp: &okta.Response{},
			apps: []okta.App{
				&okta.Application{Id: "app-01"},
				&okta.Application{Id: "app-02"},
				&okta.Application{Id: "app-03"},
			},
			want: []okta.App{
				&okta.Application{Id: "app-01"},
				&okta.Application{Id: "app-02"},
				&okta.Application{Id: "app-03"},
			},
		},
		{
			name:    "api error",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				appIface: &mockApplicationClient{
					t:    t,
					err:  tt.err,
					resp: tt.resp,
					apps: tt.apps,
				},
			}
			got, err := c.listApplications(context.TODO(), tt.qp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_GithubCloudApplications(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		resp    *okta.Response
		apps    []okta.App
		want    map[string]string
		wantErr bool
	}{
		{
			name: "example apps",
			resp: &okta.Response{},
			apps: []okta.App{
				&okta.Application{
					Id: "app-01",
					Settings: &okta.ApplicationSettings{
						App: &okta.ApplicationSettingsApplication{
							"githubOrg": "testorg01",
						},
					},
				},
				&okta.Application{
					Id: "app-02",
					Settings: &okta.ApplicationSettings{
						App: &okta.ApplicationSettingsApplication{
							"githubOrg": "testorg02",
						},
					},
				},
				&okta.Application{
					Id:       "app-03",
					Settings: &okta.ApplicationSettings{},
				},
				&okta.Application{
					Id: "app-02",
					Settings: &okta.ApplicationSettings{
						App: &okta.ApplicationSettingsApplication{
							"githubOrg": []string{"some", "not", "string"},
						},
					},
				},
				&okta.Application{Id: "app-04"},
				&otherApplication{},
			},
			want: map[string]string{
				"testorg01": "app-01",
				"testorg02": "app-02",
			},
		},
		{
			name: "nil settings",
			resp: &okta.Response{},
			apps: []okta.App{
				&okta.Application{Id: "app-01"},
				&okta.Application{Id: "app-02"},
				&okta.Application{Id: "app-03"},
				&otherApplication{},
			},
			want: map[string]string{},
		},
		{
			name: "nil settings app",
			resp: &okta.Response{},
			apps: []okta.App{
				&okta.Application{
					Id:       "app-01",
					Settings: &okta.ApplicationSettings{},
				},
				&okta.Application{
					Id:       "app-02",
					Settings: &okta.ApplicationSettings{},
				},
				&okta.Application{
					Id:       "app-03",
					Settings: &okta.ApplicationSettings{},
				},
				&otherApplication{},
			},
			want: map[string]string{},
		},
		{
			name:    "error",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				appIface: &mockApplicationClient{
					t:    t,
					err:  tt.err,
					resp: tt.resp,
					apps: tt.apps,
				},
			}
			got, err := c.GithubCloudApplications(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
