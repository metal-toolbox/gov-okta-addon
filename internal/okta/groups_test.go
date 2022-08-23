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

	group  *okta.Group
	groups []*okta.Group

	users []*okta.User

	resp *okta.Response
}

func (m *mockGroupClient) CreateGroup(ctx context.Context, body okta.Group) (*okta.Group, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.group, m.resp, nil
}

func (m *mockGroupClient) UpdateGroup(ctx context.Context, groupID string, body okta.Group) (*okta.Group, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.group, m.resp, nil
}

func (m *mockGroupClient) DeleteGroup(ctx context.Context, groupID string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockGroupClient) ListGroups(ctx context.Context, qp *query.Params) ([]*okta.Group, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.groups, m.resp, nil
}

func (m *mockGroupClient) AddUserToGroup(ctx context.Context, groupID string, userID string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockGroupClient) RemoveUserFromGroup(ctx context.Context, groupID string, userID string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockGroupClient) ListGroupUsers(ctx context.Context, groupID string, qp *query.Params) ([]*okta.User, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.users, m.resp, nil
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
			err:     errors.New("boom"), //nolint:goerr113
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
			err:     errors.New("boom"), //nolint:goerr113
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
			err := c.UpdateGroup(context.TODO(), tt.args.id, tt.args.name, tt.args.desc, tt.args.profile)
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
			err:     errors.New("boom"), //nolint:goerr113
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
			err:     errors.New("boom"), //nolint:goerr113
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
			err:     errors.New("boom"), //nolint:goerr113
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
			err:     errors.New("boom"), //nolint:goerr113
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
		want    []string
		wantErr bool
	}{
		{
			name: "example",
			users: []*okta.User{
				{Id: "user-01"},
				{Id: "user-02"},
				{Id: "user-03"},
			},
			gid:  "group-01",
			want: []string{"user-01", "user-02", "user-03"},
		},
		{
			name:    "error",
			gid:     "group-02",
			err:     errors.New("boom"), //nolint:goerr113
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
