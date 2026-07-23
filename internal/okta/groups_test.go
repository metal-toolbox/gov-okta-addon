package okta

import (
	"context"
	"errors"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockGroupClient struct {
	t   *testing.T
	err error

	apps []okta.ListApplications200ResponseInner

	group  *okta.Group
	groups []okta.Group

	users []okta.User
}

func (m *mockGroupClient) CreateGroup(_ context.Context, _ okta.AddGroupRequest) (*okta.Group, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.group, nil
}

func (m *mockGroupClient) UpdateGroup(_ context.Context, _ string, _ okta.AddGroupRequest) (*okta.Group, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.group, nil
}

func (m *mockGroupClient) DeleteGroup(_ context.Context, _ string) error {
	return m.err
}

func (m *mockGroupClient) ListGroups(_ context.Context, _ string) ([]okta.Group, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.groups, nil
}

func (m *mockGroupClient) AddUserToGroup(_ context.Context, _, _ string) error {
	return m.err
}

func (m *mockGroupClient) RemoveUserFromGroup(_ context.Context, _, _ string) error {
	return m.err
}

func (m *mockGroupClient) ListGroupUsers(_ context.Context, _ string) ([]okta.User, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.users, nil
}

func (m *mockGroupClient) ListAssignedApplicationsForGroup(_ context.Context, _ string) ([]okta.ListApplications200ResponseInner, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.apps, nil
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
			group: &okta.Group{Id: okta.PtrString("11111111")},
			args: args{
				name:    "testgroup",
				desc:    "my test group",
				profile: map[string]interface{}{"governor_id": "abc123"},
			},
			want: "11111111",
		},
		{
			name:  "okta error",
			group: &okta.Group{Id: okta.PtrString("11111111")},
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
		groups  []okta.Group
		err     error
		want    string
		wantErr bool
	}{
		{
			name: "example create group",
			groups: []okta.Group{
				{Id: okta.PtrString("11111111")},
			},
			id:   "2222222",
			want: "11111111",
		},
		{
			name: "okta error",
			groups: []okta.Group{
				{Id: okta.PtrString("11111111")},
			},
			id:      "2222222",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
		{
			name:    "empty list",
			groups:  []okta.Group{},
			id:      "2222222",
			wantErr: true,
		},
		{
			name: "more than one group returned",
			groups: []okta.Group{
				{Id: okta.PtrString("11111111")},
				{Id: okta.PtrString("33333333")},
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
		users   []okta.User
		err     error
		gid     string
		want    []*okta.User
		wantErr bool
	}{
		{
			name: "example",
			users: []okta.User{
				{Id: okta.PtrString("user-01")},
				{Id: okta.PtrString("user-02")},
				{Id: okta.PtrString("user-03")},
			},
			gid: "group-01",
			want: []*okta.User{
				{Id: okta.PtrString("user-01")},
				{Id: okta.PtrString("user-02")},
				{Id: okta.PtrString("user-03")},
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
		if g.GetId() == "skipMe" {
			return nil, nil
		}

		return g, nil
	}

	errMe := func(_ context.Context, _ *okta.Group) (*okta.Group, error) {
		return nil, errors.New("boomsauce") //nolint:err113
	}

	tests := []struct {
		name    string
		f       GroupModifierFunc
		err     error
		groups  []okta.Group
		want    []*okta.Group
		wantErr bool
	}{
		{
			name: "example skip user",
			f:    skipGroup,
			groups: []okta.Group{
				{Id: okta.PtrString("heyThere")},
				{Id: okta.PtrString("skipMe")},
			},
			want: []*okta.Group{{Id: okta.PtrString("heyThere")}},
		},
		{
			name: "okta error",
			f:    skipGroup,
			groups: []okta.Group{
				{Id: okta.PtrString("heyThere")},
				{Id: okta.PtrString("skipMe")},
			},
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
		{
			name: "func error",
			f:    errMe,
			groups: []okta.Group{
				{Id: okta.PtrString("heyThere")},
				{Id: okta.PtrString("skipMe")},
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
				},
			}

			got, err := c.ListGroupsWithModifier(context.TODO(), tt.f, "")
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
			name: "example group (user profile)",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					OktaUserGroupProfile: &okta.OktaUserGroupProfile{
						Name:        okta.PtrString("example"),
						Description: okta.PtrString("an example group"),
						AdditionalProperties: map[string]interface{}{
							GroupProfileGovernorIDKey: "some-governor-id",
						},
					},
				},
			},
			want: "some-governor-id",
		},
		{
			name: "example group (active directory profile)",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					OktaActiveDirectoryGroupProfile: &okta.OktaActiveDirectoryGroupProfile{
						Name:        okta.PtrString("example"),
						Description: okta.PtrString("an example group"),
						AdditionalProperties: map[string]interface{}{
							GroupProfileGovernorIDKey: "some-governor-id",
						},
					},
				},
			},
			want: "some-governor-id",
		},
		{
			name: "non string governor id",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					OktaUserGroupProfile: &okta.OktaUserGroupProfile{
						Name:        okta.PtrString("example"),
						Description: okta.PtrString("an example group"),
						AdditionalProperties: map[string]interface{}{
							GroupProfileGovernorIDKey: 12345,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty governor id",
			group: &okta.Group{
				Profile: &okta.GroupProfile{
					OktaUserGroupProfile: &okta.OktaUserGroupProfile{
						Name:        okta.PtrString("example"),
						Description: okta.PtrString("an example group"),
						AdditionalProperties: map[string]interface{}{
							GroupProfileGovernorIDKey: "",
						},
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
		err     error
		apps    []okta.ListApplications200ResponseInner
		want    []okta.ListApplications200ResponseInner
		wantErr bool
	}{
		{
			name:    "example app list",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", nil),
				samlAppWithGithubOrg("app-02", nil),
			},
			want: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", nil),
				samlAppWithGithubOrg("app-02", nil),
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
				},
				logger: zap.NewNop(),
			}

			got, err := c.listAssignedApplicationsForGroup(context.TODO(), tt.groupID)
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
		apps    []okta.ListApplications200ResponseInner
		err     error
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "example app list",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", "test-org-01"),
				samlAppWithGithubOrg("app-02", nil),
			},
			want: map[string]string{"test-org-01": "app-01"},
		},
		{
			name:    "non-string githubOrg",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", 1234),
				samlAppWithGithubOrg("app-02", nil),
			},
			want: map[string]string{},
		},
		{
			name:    "example app list without github",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps: []okta.ListApplications200ResponseInner{
				samlAppWithGithubOrg("app-01", nil),
				samlAppWithGithubOrg("app-02", nil),
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
