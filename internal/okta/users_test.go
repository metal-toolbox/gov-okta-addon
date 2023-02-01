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

type mockUserClient struct {
	t   *testing.T
	err error

	users []*okta.User

	resp *okta.Response

	deactivatedUser bool
}

func (m *mockUserClient) ClearUserSessions(ctx context.Context, userID string, qp *query.Params) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockUserClient) DeactivateUser(ctx context.Context, userID string, qp *query.Params) (*okta.Response, error) {
	m.deactivatedUser = true

	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockUserClient) DeactivateOrDeleteUser(ctx context.Context, userID string, qp *query.Params) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockUserClient) GetUser(ctx context.Context, userID string) (*okta.User, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.users[0], m.resp, nil
}

func (m *mockUserClient) ListUsers(ctx context.Context, qp *query.Params) ([]*okta.User, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.users, m.resp, nil
}

func (m *mockUserClient) SuspendUser(ctx context.Context, userID string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func (m *mockUserClient) UnsuspendUser(ctx context.Context, userID string) (*okta.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.resp, nil
}

func TestClient_ClearUserSessions(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		err     error
		wantErr bool
	}{
		{
			name: "example clear user sessions",
			id:   "user101",
		},
		{
			name:    "okta error",
			id:      "user101",
			err:     errors.New("boomsauce"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				userIface: &mockUserClient{
					t:   t,
					err: tt.err,
				},
			}
			err := c.ClearUserSessions(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_DeactivateUser(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		err     error
		wantErr bool
	}{
		{
			name: "example deactivate user",
			id:   "user101",
		},
		{
			name:    "okta error",
			id:      "user101",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				userIface: &mockUserClient{
					t:   t,
					err: tt.err,
				},
			}
			err := c.DeactivateUser(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_DeleteUser(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		users   []*okta.User
		err     error
		wantDA  bool
		wantErr bool
	}{
		{
			name: "delete active user",
			id:   "user101",
			users: []*okta.User{
				{Id: "11111111", Status: "ACTIVE"},
			},
			wantDA: true,
		},
		{
			name: "delete deactivated user",
			id:   "user101",
			users: []*okta.User{
				{Id: "11111111", Status: "DEPROVISIONED"},
			},
			wantDA: false,
		},
		{
			name: "okta error",
			id:   "user101",
			users: []*okta.User{
				{Id: "11111111"},
			},
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockUserClient{
				t:               t,
				err:             tt.err,
				users:           tt.users,
				resp:            &okta.Response{},
				deactivatedUser: false,
			}

			c := &Client{
				logger:    zap.NewNop(),
				userIface: m,
			}

			err := c.DeleteUser(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantDA, m.deactivatedUser)
		})
	}
}

func TestClient_SuspendUser(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		err     error
		wantErr bool
	}{
		{
			name: "example suspend user",
			id:   "user101",
		},
		{
			name:    "okta error",
			id:      "user101",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				userIface: &mockUserClient{
					t:   t,
					err: tt.err,
				},
			}
			err := c.SuspendUser(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_UnsuspendUser(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		err     error
		wantErr bool
	}{
		{
			name: "example un-suspend user",
			id:   "user101",
		},
		{
			name:    "okta error",
			id:      "user101",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				userIface: &mockUserClient{
					t:   t,
					err: tt.err,
				},
			}
			err := c.UnsuspendUser(context.TODO(), tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_GetUserIDByEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		users   []*okta.User
		err     error
		want    string
		wantErr bool
	}{
		{
			name: "example get user by email",
			users: []*okta.User{
				{Id: "11111111"},
			},
			email: "foo@example.com",
			want:  "11111111",
		},
		{
			name: "okta error",
			users: []*okta.User{
				{Id: "11111111"},
			},
			email:   "foo@example.com",
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
		{
			name:    "empty list",
			users:   []*okta.User{},
			email:   "foo@example.com",
			wantErr: true,
		},
		{
			name: "more than one group returned",
			users: []*okta.User{
				{Id: "11111111"},
				{Id: "33333333"},
			},
			email:   "foo@example.com",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				userIface: &mockUserClient{
					t:     t,
					err:   tt.err,
					users: tt.users,
				},
			}
			got, err := c.GetUserIDByEmail(context.TODO(), tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_ListUsers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		users   []*okta.User
		want    []*okta.User
		wantErr bool
	}{
		{
			name: "successful list users",
			users: []*okta.User{
				{Id: "user1"},
				{Id: "user2"},
			},
			want: []*okta.User{{Id: "user1"}, {Id: "user2"}},
		},
		{
			name:  "empty list users",
			users: []*okta.User{},
			want:  []*okta.User{},
		},
		{
			name: "okta error",
			users: []*okta.User{
				{Id: "user1"},
				{Id: "user1"},
			},
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				userIface: &mockUserClient{
					t:     t,
					err:   tt.err,
					users: tt.users,
					resp:  &okta.Response{},
				},
			}
			got, err := c.ListUsers(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_ListUsersWithModifier(t *testing.T) {
	skipUser := func(ctx context.Context, u *okta.User) (*okta.User, error) {
		if u.Id == "skipMe" {
			return nil, nil
		}

		return u, nil
	}

	errMe := func(ctx context.Context, u *okta.User) (*okta.User, error) {
		return nil, errors.New("boomsauce") //nolint:goerr113
	}

	type args struct {
		f UserModifierFunc
		q *query.Params
	}

	tests := []struct {
		name    string
		args    args
		err     error
		users   []*okta.User
		want    []*okta.User
		wantErr bool
	}{
		{
			name: "example skip user",
			args: args{
				f: skipUser,
				q: &query.Params{},
			},
			users: []*okta.User{
				{Id: "heyThere"},
				{Id: "skipMe"},
			},
			want: []*okta.User{{Id: "heyThere"}},
		},
		{
			name: "okta error",
			args: args{
				f: skipUser,
				q: &query.Params{},
			},
			users: []*okta.User{
				{Id: "heyThere"},
				{Id: "skipMe"},
			},
			err:     errors.New("boom"), //nolint:goerr113
			wantErr: true,
		},
		{
			name: "func error",
			args: args{
				f: errMe,
				q: &query.Params{},
			},
			users: []*okta.User{
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
				userIface: &mockUserClient{
					t:     t,
					err:   tt.err,
					users: tt.users,
					resp:  &okta.Response{},
				},
			}
			got, err := c.ListUsersWithModifier(context.TODO(), tt.args.f, tt.args.q)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_EmailFromUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		user    *okta.User
		want    string
		wantErr bool
	}{
		{
			name: "example email",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"email": "test1@test.com",
				},
			},
			want: "test1@test.com",
		},
		{
			name: "not found",
			user: &okta.User{
				Profile: &okta.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"email": 12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EmailFromUserProfile(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_FirstNameFromUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		user    *okta.User
		want    string
		wantErr bool
	}{
		{
			name: "example firstName",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"firstName": "Test",
				},
			},
			want: "Test",
		},
		{
			name: "not found",
			user: &okta.User{
				Profile: &okta.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"firstName": 12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FirstNameFromUserProfile(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_LastNameFromUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		user    *okta.User
		want    string
		wantErr bool
	}{
		{
			name: "example lastName",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"lastName": "One",
				},
			},
			want: "One",
		},
		{
			name: "not found",
			user: &okta.User{
				Profile: &okta.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"lastName": 12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LastNameFromUserProfile(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_UserDetailsFromOktaUser(t *testing.T) {
	tests := []struct {
		name    string
		user    *okta.User
		want    *UserDetails
		wantErr bool
	}{
		{
			name: "successful example",
			user: &okta.User{
				Id:     "00u123456789abcde697",
				Status: "ACTIVE",
				Profile: &okta.UserProfile{
					"firstName": "Burrow",
					"lastName":  "Blaster",
					"email":     "bblaster@gopher.com",
				},
			},
			want: &UserDetails{
				ID:     "00u123456789abcde697",
				Name:   "Burrow Blaster",
				Email:  "bblaster@gopher.com",
				Status: "ACTIVE",
			},
		},
		{
			name: "empty profile",
			user: &okta.User{
				Profile: &okta.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "missing email",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"firstName": "Burrow",
					"lastName":  "Blaster",
				},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okta.User{
				Profile: &okta.UserProfile{
					"firstName": "Burrow",
					"lastName":  12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UserDetailsFromOktaUser(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
