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
}

func (m *mockUserClient) DeactivateUser(ctx context.Context, userID string, qp *query.Params) (*okta.Response, error) {
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
		return m.users[0], nil, m.err
	}

	return nil, m.resp, nil
}

func (m *mockUserClient) ListUsers(ctx context.Context, qp *query.Params) ([]*okta.User, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.users, m.resp, nil
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
