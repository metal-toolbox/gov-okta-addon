package okta

import (
	"context"
	"errors"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockApplicationClient struct {
	t                   *testing.T
	err                 error
	apps                []okta.ListApplications200ResponseInner
	appGroupAssignments []okta.ApplicationGroupAssignment
}

func (m *mockApplicationClient) ListApplications(context.Context, string) ([]okta.ListApplications200ResponseInner, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.apps, nil
}

func (m *mockApplicationClient) CreateApplicationGroupAssignment(_ context.Context, _, _ string) (*okta.ApplicationGroupAssignment, error) {
	if m.err != nil {
		return nil, m.err
	}

	return nil, nil
}

func (m *mockApplicationClient) DeleteApplicationGroupAssignment(_ context.Context, _, _ string) error {
	return m.err
}

func (m *mockApplicationClient) GetApplicationGroupAssignment(_ context.Context, _, _ string) (*okta.ApplicationGroupAssignment, error) {
	if m.err != nil {
		return nil, m.err
	}

	return nil, nil
}

func (m *mockApplicationClient) ListApplicationGroupAssignments(_ context.Context, _ string) ([]okta.ApplicationGroupAssignment, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.appGroupAssignments, nil
}

// samlAppWithGithubOrg builds an okta application (as the oneOf response wrapper) with the
// given id and, optionally, a githubOrg app setting.
func samlAppWithGithubOrg(id string, githubOrg interface{}) okta.ListApplications200ResponseInner {
	app := &okta.SamlApplication{
		Application: okta.Application{Id: okta.PtrString(id)},
	}

	if githubOrg != nil {
		app.Settings = &okta.SamlApplicationSettings{
			AdditionalProperties: map[string]interface{}{
				"app": map[string]interface{}{"githubOrg": githubOrg},
			},
		}
	}

	return okta.ListApplications200ResponseInner{SamlApplication: app}
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
			name:    "empty appID",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
			wantErr: true,
		},
		{
			name:    "empty groupID",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			wantErr: true,
		},
		{
			name:    "error",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
			err:     errors.New("boom"), //nolint:err113
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
			name:    "empty appID",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
			wantErr: true,
		},
		{
			name:    "empty groupID",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			wantErr: true,
		},
		{
			name:    "error",
			appID:   "14270ca5-ea9f-43b7-a560-f2014399bddc",
			groupID: "39712500-37a8-4102-bce9-432cbe2c28d2",
			err:     errors.New("boom"), //nolint:err113
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
		assignments []okta.ApplicationGroupAssignment
		want        []string
		wantErr     bool
	}{
		{
			name:  "example",
			appID: "47819d20-70e5-4ab9-b008-898be42adde7",
			assignments: []okta.ApplicationGroupAssignment{
				{Id: okta.PtrString("group-001")},
				{Id: okta.PtrString("group-002")},
			},
			want: []string{"group-001", "group-002"},
		},
		{
			name:    "empty appID",
			wantErr: true,
		},
		{
			name:    "api error",
			appID:   "47819d20-70e5-4ab9-b008-898be42adde7",
			err:     errors.New("boom"), //nolint:err113
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
		err     error
		apps    []okta.ListApplications200ResponseInner
		want    []okta.ListApplications200ResponseInner
		wantErr bool
	}{
		{
			name: "example",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", nil),
				samlAppWithGithubOrg("app-02", nil),
				samlAppWithGithubOrg("app-03", nil),
			},
			want: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", nil),
				samlAppWithGithubOrg("app-02", nil),
				samlAppWithGithubOrg("app-03", nil),
			},
		},
		{
			name: "example empty response",
			apps: []okta.ListApplications200ResponseInner{},
			want: []okta.ListApplications200ResponseInner{},
		},
		{
			name:    "api error",
			err:     errors.New("boom"), //nolint:err113
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
					apps: tt.apps,
				},
			}

			got, err := c.listApplications(context.TODO(), "")
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
		apps    []okta.ListApplications200ResponseInner
		want    map[string]string
		wantErr bool
	}{
		{
			name: "example apps",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", "testorg01"),
				samlAppWithGithubOrg("app-02", "testorg02"),
				samlAppWithGithubOrg("app-03", nil),
				samlAppWithGithubOrg("app-05", []string{"some", "not", "string"}),
				samlAppWithGithubOrg("app-06", nil),
			},
			want: map[string]string{
				"testorg01": "app-01",
				"testorg02": "app-02",
			},
		},
		{
			name: "nil settings",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", nil),
				samlAppWithGithubOrg("app-02", nil),
				samlAppWithGithubOrg("app-03", nil),
			},
			want: map[string]string{},
		},
		{
			name:    "error",
			err:     errors.New("boom"), //nolint:err113
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
