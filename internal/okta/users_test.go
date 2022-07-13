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
}

func (m *mockUserClient) ListUsers(ctx context.Context, qp *query.Params) ([]*okta.User, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	return m.users, nil, nil
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
			name: "example create group",
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
