package okta

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
)

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
		apiErr  bool
		wantErr bool
	}{
		{name: "example clear user sessions"},
		{name: "okta error", apiErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusNoContent, nil)
				})
			}

			c := newTestClient(t, h)

			err := c.ClearUserSessions(context.TODO(), "user101")
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
		apiErr  bool
		wantErr bool
	}{
		{name: "example deactivate user"},
		{name: "okta error", apiErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, okta.User{Id: okta.PtrString("user101")})
				})
			}

			c := newTestClient(t, h)

			err := c.DeactivateUser(context.TODO(), "user101")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_DeleteUser(t *testing.T) {
	const id = "user101"

	tests := []struct {
		name    string
		status  string
		apiErr  bool
		wantDA  bool
		wantErr bool
	}{
		{name: "delete active user", status: "ACTIVE", wantDA: true},
		{name: "delete deactivated user", status: "DEPROVISIONED", wantDA: false},
		{name: "okta error", apiErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var deactivated bool

			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch {
					case r.Method == http.MethodGet && r.URL.Path == "/api/v1/users/"+id:
						writeJSON(t, w, http.StatusOK, okta.UserGetSingleton{Id: okta.PtrString(id), Status: okta.PtrString(tt.status)})
					case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/lifecycle/deactivate"):
						deactivated = true

						writeJSON(t, w, http.StatusOK, nil)
					case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/users/"+id:
						writeJSON(t, w, http.StatusNoContent, nil)
					default:
						t.Errorf("unexpected okta api call: %s %s", r.Method, r.URL.Path)
					}
				})
			}

			c := newTestClient(t, h)

			err := c.DeleteUser(context.TODO(), id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantDA, deactivated)
		})
	}
}

func TestClient_SuspendUser(t *testing.T) {
	tests := []struct {
		name    string
		apiErr  bool
		wantErr bool
	}{
		{name: "example suspend user"},
		{name: "okta error", apiErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, nil)
				})
			}

			c := newTestClient(t, h)

			err := c.SuspendUser(context.TODO(), "user101")
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
		apiErr  bool
		wantErr bool
	}{
		{name: "example un-suspend user"},
		{name: "okta error", apiErr: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, nil)
				})
			}

			c := newTestClient(t, h)

			err := c.UnsuspendUser(context.TODO(), "user101")
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
		users   []okta.User
		apiErr  bool
		want    string
		wantErr bool
	}{
		{
			name:  "example get user by email",
			users: []okta.User{{Id: okta.PtrString("11111111")}},
			want:  "11111111",
		},
		{
			name:    "okta error",
			apiErr:  true,
			wantErr: true,
		},
		{
			name:    "empty list",
			users:   []okta.User{},
			wantErr: true,
		},
		{
			name:    "more than one user returned",
			users:   []okta.User{{Id: okta.PtrString("11111111")}, {Id: okta.PtrString("33333333")}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, tt.users)
				})
			}

			c := newTestClient(t, h)

			got, err := c.GetUserIDByEmail(context.TODO(), "foo@example.com")
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
		apiErr  bool
		users   []okta.User
		want    []string
		wantErr bool
	}{
		{
			name:  "successful list users",
			users: []okta.User{{Id: okta.PtrString("user1")}, {Id: okta.PtrString("user2")}},
			want:  []string{"user1", "user2"},
		},
		{
			name:  "empty list users",
			users: []okta.User{},
			want:  []string{},
		},
		{
			name:    "okta error",
			apiErr:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, tt.users)
				})
			}

			c := newTestClient(t, h)

			got, err := c.ListUsers(context.TODO())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, userIDs(got))
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
		return nil, assert.AnError
	}

	users := []okta.User{
		{Id: okta.PtrString("heyThere")},
		{Id: okta.PtrString("skipMe")},
	}

	tests := []struct {
		name    string
		f       UserModifierFunc
		apiErr  bool
		want    []string
		wantErr bool
	}{
		{
			name: "example skip user",
			f:    skipUser,
			want: []string{"heyThere"},
		},
		{
			name:    "okta error",
			f:       skipUser,
			apiErr:  true,
			wantErr: true,
		},
		{
			name:    "func error",
			f:       errMe,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler
			if tt.apiErr {
				h = errorHandler()
			} else {
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, users)
				})
			}

			c := newTestClient(t, h)

			got, err := c.ListUsersWithModifier(context.TODO(), tt.f, "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, userIDs(got))
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
			user: &okta.User{Profile: userProfile("test1@test.com", "", "")},
			want: "test1@test.com",
		},
		{
			name:    "not found",
			user:    &okta.User{Profile: &okta.UserProfile{}},
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
			user: &okta.User{Profile: userProfile("", "Test", "")},
			want: "Test",
		},
		{
			name:    "not found",
			user:    &okta.User{Profile: &okta.UserProfile{}},
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
			user: &okta.User{Profile: userProfile("", "", "One")},
			want: "One",
		},
		{
			name:    "not found",
			user:    &okta.User{Profile: &okta.UserProfile{}},
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
			name:    "empty profile",
			user:    &okta.User{Profile: &okta.UserProfile{}},
			wantErr: true,
		},
		{
			name:    "missing email",
			user:    &okta.User{Profile: userProfile("", "Burrow", "Blaster")},
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
