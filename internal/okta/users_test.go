package okta

import (
	"context"
	"errors"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockUserClient struct {
	t   *testing.T
	err error

	users []okta.User

	deactivatedUser bool
}

func (m *mockUserClient) ClearUserSessions(_ context.Context, _ string) error {
	return m.err
}

func (m *mockUserClient) DeactivateUser(_ context.Context, _ string) error {
	m.deactivatedUser = true

	return m.err
}

func (m *mockUserClient) DeactivateOrDeleteUser(_ context.Context, _ string) error {
	return m.err
}

func (m *mockUserClient) GetUser(_ context.Context, _ string) (*okta.User, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &m.users[0], nil
}

func (m *mockUserClient) ListUsers(_ context.Context, _ string) ([]okta.User, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.users, nil
}

func (m *mockUserClient) SuspendUser(_ context.Context, _ string) error {
	return m.err
}

func (m *mockUserClient) UnsuspendUser(_ context.Context, _ string) error {
	return m.err
}

// userProfile builds an okta user profile from the given fields, leaving unset any field
// passed as an empty string.
func userProfile(email, first, last string) *okta.UserProfile {
	p := &okta.UserProfile{}

	if email != "" {
		p.Email = okta.PtrString(email)
	}

	if first != "" {
		p.FirstName = *okta.NewNullableString(okta.PtrString(first))
	}

	if last != "" {
		p.LastName = *okta.NewNullableString(okta.PtrString(last))
	}

	return p
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
			err:     errors.New("boomsauce"), //nolint:err113
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
			err:     errors.New("boom"), //nolint:err113
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
		users   []okta.User
		err     error
		wantDA  bool
		wantErr bool
	}{
		{
			name: "delete active user",
			id:   "user101",
			users: []okta.User{
				{Id: okta.PtrString("11111111"), Status: okta.PtrString("ACTIVE")},
			},
			wantDA: true,
		},
		{
			name: "delete deactivated user",
			id:   "user101",
			users: []okta.User{
				{Id: okta.PtrString("11111111"), Status: okta.PtrString("DEPROVISIONED")},
			},
			wantDA: false,
		},
		{
			name: "okta error",
			id:   "user101",
			users: []okta.User{
				{Id: okta.PtrString("11111111")},
			},
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockUserClient{
				t:               t,
				err:             tt.err,
				users:           tt.users,
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
			err:     errors.New("boom"), //nolint:err113
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
			err:     errors.New("boom"), //nolint:err113
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
		users   []okta.User
		err     error
		want    string
		wantErr bool
	}{
		{
			name: "example get user by email",
			users: []okta.User{
				{Id: okta.PtrString("11111111")},
			},
			email: "foo@example.com",
			want:  "11111111",
		},
		{
			name: "okta error",
			users: []okta.User{
				{Id: okta.PtrString("11111111")},
			},
			email:   "foo@example.com",
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
		{
			name:    "empty list",
			users:   []okta.User{},
			email:   "foo@example.com",
			wantErr: true,
		},
		{
			name: "more than one group returned",
			users: []okta.User{
				{Id: okta.PtrString("11111111")},
				{Id: okta.PtrString("33333333")},
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
		users   []okta.User
		want    []*okta.User
		wantErr bool
	}{
		{
			name: "successful list users",
			users: []okta.User{
				{Id: okta.PtrString("user1")},
				{Id: okta.PtrString("user2")},
			},
			want: []*okta.User{{Id: okta.PtrString("user1")}, {Id: okta.PtrString("user2")}},
		},
		{
			name:  "empty list users",
			users: []okta.User{},
			want:  []*okta.User{},
		},
		{
			name: "okta error",
			users: []okta.User{
				{Id: okta.PtrString("user1")},
				{Id: okta.PtrString("user1")},
			},
			err:     errors.New("boom"), //nolint:err113
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
	skipUser := func(_ context.Context, u *okta.User) (*okta.User, error) {
		if u.GetId() == "skipMe" {
			return nil, nil
		}

		return u, nil
	}

	errMe := func(_ context.Context, _ *okta.User) (*okta.User, error) {
		return nil, errors.New("boomsauce") //nolint:err113
	}

	tests := []struct {
		name    string
		f       UserModifierFunc
		err     error
		users   []okta.User
		want    []*okta.User
		wantErr bool
	}{
		{
			name: "example skip user",
			f:    skipUser,
			users: []okta.User{
				{Id: okta.PtrString("heyThere")},
				{Id: okta.PtrString("skipMe")},
			},
			want: []*okta.User{{Id: okta.PtrString("heyThere")}},
		},
		{
			name: "okta error",
			f:    skipUser,
			users: []okta.User{
				{Id: okta.PtrString("heyThere")},
				{Id: okta.PtrString("skipMe")},
			},
			err:     errors.New("boom"), //nolint:err113
			wantErr: true,
		},
		{
			name: "func error",
			f:    errMe,
			users: []okta.User{
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
				userIface: &mockUserClient{
					t:     t,
					err:   tt.err,
					users: tt.users,
				},
			}

			got, err := c.ListUsersWithModifier(context.TODO(), tt.f, "")
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
				Profile: userProfile("test1@test.com", "", ""),
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
				Profile: userProfile("", "Test", ""),
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
				Profile: userProfile("", "", "One"),
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
				Id:      okta.PtrString("00u123456789abcde697"),
				Status:  okta.PtrString("ACTIVE"),
				Profile: userProfile("bblaster@gopher.com", "Burrow", "Blaster"),
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
				Profile: userProfile("", "Burrow", "Blaster"),
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
