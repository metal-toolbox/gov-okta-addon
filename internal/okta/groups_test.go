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

type mockGroupClient struct {
	t   *testing.T
	err error

	apps []okta.App

	group  *okta.Group
	groups []*okta.Group

	users []*okta.User

	resp *okta.Response
}

func (m *mockGroupClient) CreateGroup(_ context.Context, _ okta.Group) (*okta.Group, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.group, m.resp, nil
}

func (m *mockGroupClient) UpdateGroup(_ context.Context, _ string, _ okta.Group) (*okta.Group, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.group, m.resp, nil
}

func (m *mockGroupClient) DeleteGroup(_ context.Context, _ string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockGroupClient) ListGroups(_ context.Context, _ *query.Params) ([]*okta.Group, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.groups, m.resp, nil
}

func (m *mockGroupClient) AddUserToGroup(_ context.Context, _, _ string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockGroupClient) RemoveUserFromGroup(_ context.Context, _, _ string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockGroupClient) ListGroupUsers(_ context.Context, _ string, _ *query.Params) ([]*okta.User, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.users, m.resp, nil
}

func (m *mockGroupClient) ListAssignedApplicationsForGroup(_ context.Context, _ string, _ *query.Params) ([]okta.App, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.apps, m.resp, nil
}

func TestClient_CreateGroup(t *testing.T) {
	type args struct {
		name    string
		desc    string
		profile map[string]interface{}
	}

	tests := []struct {
		name    string
		err     error
		args    args
		group   *okta.Group
		want    string
		wantErr bool
	}{
		{
			name:  "example create group",
			group: &okta.Group{Id: "11111111"},
			args: args{
				name:    "testgroup",
				desc:    "my test group",
				profile: map[string]interface{}{"governor_id": "abc123"},
			},
			want: "11111111",
		},
		{
			name:  "okta error",
			group: &okta.Group{Id: "11111111"},
			args: args{
				name:    "testgroup",
				desc:    "my test group",
				profile: map[string]interface{}{"governor_id": "abc123"},
			},
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:     t,
					err:   tt.err,
					group: tt.group,
				},
				logger: zap.NewNop(),
			}

			got, err := c.CreateGroup(context.TODO(), tt.args.name, tt.args.desc, tt.args.profile)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_UpdateGroup(t *testing.T) {
	type args struct {
		id      string
		name    string
		desc    string
		profile map[string]interface{}
	}

	tests := []struct {
		name    string
		args    args
		err     error
		wantErr bool
	}{
		{
			name: "example update group",
			args: args{
				id:      "11111111",
				name:    "testgroup",
				desc:    "my test group",
				profile: map[string]interface{}{"governor_id": "abc123"},
			},
		},
		{
			name: "okta error",
			args: args{
				id:      "11111111",
				name:    "testgroup",
				desc:    "my test group",
				profile: map[string]interface{}{"governor_id": "abc123"},
			},
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:   t,
					err: tt.err,
				},
				logger: zap.NewNop(),
			}

			_, err := c.UpdateGroup(context.TODO(), tt.args.id, tt.args.name, tt.args.desc, tt.args.profile)
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
		name    string
		id      string
		err     error
		wantErr bool
	}{
		{
			name: "example update group",
			id:   "11111111",
		},
		{
			name:    "okta error",
			id:      "11111111",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:   t,
					err: tt.err,
				},
				logger: zap.NewNop(),
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

func TestClient_GetGroupByGovernorID(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		groups  []*okta.Group
		err     error
		want    string
		wantErr bool
	}{
		{
			name: "example create group",
			groups: []*okta.Group{
				{Id: "11111111"},
			},
			id:   "2222222",
			want: "11111111",
		},
		{
			name: "okta error",
			groups: []*okta.Group{
				{Id: "11111111"},
			},
			id:      "2222222",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
		{
			name:    "empty list",
			groups:  []*okta.Group{},
			id:      "2222222",
			wantErr: true,
		},
		{
			name: "more than one group returned",
			groups: []*okta.Group{
				{Id: "11111111"},
				{Id: "33333333"},
			},
			id:      "2222222",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:      t,
					err:    tt.err,
					groups: tt.groups,
				},
				logger: zap.NewNop(),
			}

			got, err := c.GetGroupByGovernorID(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_AddGroupUser(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		groupID string
		userID  string
		wantErr bool
	}{
		{
			name:    "example add user to group",
			groupID: "11111111",
			userID:  "22222222",
		},
		{
			name:    "okta error",
			groupID: "11111111",
			userID:  "22222222",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:   t,
					err: tt.err,
				},
				logger: zap.NewNop(),
			}

			err := c.AddGroupUser(context.TODO(), tt.groupID, tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_RemoveGroupUser(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
		userID  string
		err     error
		wantErr bool
	}{
		{
			name:    "example add user to group",
			groupID: "11111111",
			userID:  "22222222",
		},
		{
			name:    "okta error",
			groupID: "11111111",
			userID:  "22222222",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:   t,
					err: tt.err,
				},
				logger: zap.NewNop(),
			}

			err := c.RemoveGroupUser(context.TODO(), tt.groupID, tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_ListGroupMembership(t *testing.T) {
	tests := []struct {
		name    string
		users   []*okta.User
		err     error
		gid     string
		want    []*okta.User
		wantErr bool
	}{
		{
			name: "example",
			users: []*okta.User{
				{Id: "user-01"},
				{Id: "user-02"},
				{Id: "user-03"},
			},
			gid: "group-01",
			want: []*okta.User{
				{Id: "user-01"},
				{Id: "user-02"},
				{Id: "user-03"},
			},
		},
		{
			name:    "error",
			gid:     "group-02",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:     t,
					err:   tt.err,
					users: tt.users,
					resp:  &okta.Response{},
				},
				logger: zap.NewNop(),
			}

			got, err := c.ListGroupMembership(context.TODO(), tt.gid)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_ListGroupsWithModifier(t *testing.T) {
	skipGroup := func(_ context.Context, g *okta.Group) (*okta.Group, error) {
		if g.Id == "skipMe" {
			return nil, nil
		}

		return g, nil
	}

	errMe := func(_ context.Context, _ *okta.Group) (*okta.Group, error) {
		return nil, errors.New("boomsauce") //nolint:err113
	}

	type args struct {
		f GroupModifierFunc
		q *query.Params
	}

	tests := []struct {
		name    string
		args    args
		err     error
		groups  []*okta.Group
		want    []*okta.Group
		wantErr bool
	}{
		{
			name: "example skip user",
			args: args{
				f: skipGroup,
				q: &query.Params{},
			},
			groups: []*okta.Group{
				{Id: "heyThere"},
				{Id: "skipMe"},
			},
			want: []*okta.Group{{Id: "heyThere"}},
		},
		{
			name: "okta error",
			args: args{
				f: skipGroup,
				q: &query.Params{},
			},
			groups: []*okta.Group{
				{Id: "heyThere"},
				{Id: "skipMe"},
			},
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
		{
			name: "func error",
			args: args{
				f: errMe,
				q: &query.Params{},
			},
			groups: []*okta.Group{
				{Id: "heyThere"},
				{Id: "skipMe"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				groupIface: &mockGroupClient{
					t:      t,
					err:    tt.err,
					groups: tt.groups,
					resp:   &okta.Response{},
				},
			}

			got, err := c.ListGroupsWithModifier(context.TODO(), tt.args.f, tt.args.q)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGroupGovernorID(t *testing.T) {
	tests := []struct {
		name    string
		group   *okta.Group
		want    string
		wantErr bool
	}{
		{
			name: "example group",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					Name:        "example",
					Description: "an example group",
					GroupProfileMap: okta.GroupProfileMap{
						GroupProfileGovernorIDKey: "some-governor-id",
					},
				},
			},
			want: "some-governor-id",
		},
		{
			name: "non string governor id",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					Name:        "example",
					Description: "an example group",
					GroupProfileMap: okta.GroupProfileMap{
						GroupProfileGovernorIDKey: 12345,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty governor id",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					Name:        "example",
					Description: "an example group",
					GroupProfileMap: okta.GroupProfileMap{
						GroupProfileGovernorIDKey: "",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GroupGovernorID(tt.group)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_listAssignedApplicationsForGroup(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
		qp      *query.Params
		err     error
		apps    []okta.App
		want    []okta.App
		wantErr bool
	}{
		{
			name:    "example app list",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.App{
				&okta.Application{
					Id:   "app-01",
					Name: "App 01",
				},
				&okta.Application{
					Id:   "app-02",
					Name: "App 02",
				},
			},
			want: []okta.App{
				&okta.Application{
					Id:   "app-01",
					Name: "App 01",
				},
				&okta.Application{
					Id:   "app-02",
					Name: "App 02",
				},
			},
		},
		{
			name:    "list error",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			err:     errors.New("boomsauce"), //nolint:err113
			wantErr: true,
		},
		{
			name:    "empty groupid error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:    t,
					err:  tt.err,
					apps: tt.apps,
					resp: &okta.Response{},
				},
				logger: zap.NewNop(),
			}

			got, err := c.listAssignedApplicationsForGroup(context.TODO(), tt.groupID, tt.qp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_GroupGithubCloudApplications(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
		qp      *query.Params
		apps    []okta.App
		err     error
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "example app list",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.App{
				&okta.Application{
					Id:   "app-01",
					Name: "App 01",
					Settings: &okta.ApplicationSettings{
						App: &okta.ApplicationSettingsApplication{
							"githubOrg": "test-org-01",
						},
					},
				},
				&okta.Application{
					Id:   "app-02",
					Name: "App 02",
				},
			},
			want: map[string]string{"test-org-01": "app-01"},
		},
		{
			name:    "non-string githubOrg",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.App{
				&okta.Application{
					Id:   "app-01",
					Name: "App 01",
					Settings: &okta.ApplicationSettings{
						App: &okta.ApplicationSettingsApplication{
							"githubOrg": 1234,
						},
					},
				},
				&okta.Application{
					Id:   "app-02",
					Name: "App 02",
				},
			},
			want: map[string]string{},
		},
		{
			name:    "example app list without github",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.App{
				&okta.Application{
					Id:   "app-01",
					Name: "App 01",
				},
				&okta.Application{
					Id:   "app-02",
					Name: "App 02",
				},
			},
			want: map[string]string{},
		},
		{
			name:    "list error",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			err:     errors.New("boomsauce"), //nolint:err113
			wantErr: true,
		},
		{
			name:    "empty groupid error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				groupIface: &mockGroupClient{
					t:    t,
					err:  tt.err,
					apps: tt.apps,
					resp: &okta.Response{},
				},
				logger: zap.NewNop(),
			}

			got, err := c.GroupGithubCloudApplications(context.TODO(), tt.groupID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
