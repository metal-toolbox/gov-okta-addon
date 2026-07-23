package okta

import (
	"context"
	"net/http"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
)

func TestClient_CreateGroup(t *testing.T) {
	tests := []struct {
		name    string
		apiErr  bool
		want    string
		wantErr bool
	}{
		{
			name: "example create group",
			want: "11111111",
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
					writeJSON(t, w, http.StatusOK, okta.Group{Id: okta.PtrString(tt.want)})
				})
			}

			c := newTestClient(t, h)

			got, err := c.CreateGroup(context.TODO(), "testgroup", "my test group", map[string]interface{}{"governor_id": "abc123"})
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
	tests := []struct {
		name    string
		apiErr  bool
		wantErr bool
	}{
		{
			name: "example update group",
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
					writeJSON(t, w, http.StatusOK, okta.Group{Id: okta.PtrString("11111111")})
				})
			}

			c := newTestClient(t, h)

			_, err := c.UpdateGroup(context.TODO(), "11111111", "testgroup", "my test group", map[string]interface{}{"governor_id": "abc123"})
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
		apiErr  bool
		wantErr bool
	}{
		{
			name: "example delete group",
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
					writeJSON(t, w, http.StatusNoContent, nil)
				})
			}

			c := newTestClient(t, h)

			err := c.DeleteGroup(context.TODO(), "11111111")
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
		apiErr  bool
		groups  []okta.Group
		want    string
		wantErr bool
	}{
		{
			name:   "example get group",
			groups: []okta.Group{{Id: okta.PtrString("11111111")}},
			want:   "11111111",
		},
		{
			name:    "okta error",
			apiErr:  true,
			wantErr: true,
		},
		{
			name:    "empty list",
			groups:  []okta.Group{},
			wantErr: true,
		},
		{
			name:    "more than one group returned",
			groups:  []okta.Group{{Id: okta.PtrString("11111111")}, {Id: okta.PtrString("33333333")}},
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
					writeJSON(t, w, http.StatusOK, tt.groups)
				})
			}

			c := newTestClient(t, h)

			got, err := c.GetGroupByGovernorID(context.TODO(), "2222222")
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
		apiErr  bool
		wantErr bool
	}{
		{name: "example add user to group"},
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

			err := c.AddGroupUser(context.TODO(), "11111111", "22222222")
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
		apiErr  bool
		wantErr bool
	}{
		{name: "example remove user from group"},
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

			err := c.RemoveGroupUser(context.TODO(), "11111111", "22222222")
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
		apiErr  bool
		want    []string
		wantErr bool
	}{
		{
			name: "example",
			users: []okta.User{
				{Id: okta.PtrString("user-01")},
				{Id: okta.PtrString("user-02")},
				{Id: okta.PtrString("user-03")},
			},
			want: []string{"user-01", "user-02", "user-03"},
		},
		{
			name:    "error",
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

			got, err := c.ListGroupMembership(context.TODO(), "group-01")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, userIDs(got))
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
		return nil, assert.AnError
	}

	groups := []okta.Group{
		{Id: okta.PtrString("heyThere")},
		{Id: okta.PtrString("skipMe")},
	}

	tests := []struct {
		name    string
		f       GroupModifierFunc
		apiErr  bool
		want    []string
		wantErr bool
	}{
		{
			name: "example skip group",
			f:    skipGroup,
			want: []string{"heyThere"},
		},
		{
			name:    "okta error",
			f:       skipGroup,
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
					writeJSON(t, w, http.StatusOK, groups)
				})
			}

			c := newTestClient(t, h)

			got, err := c.ListGroupsWithModifier(context.TODO(), tt.f, "")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, groupIDs(got))
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
						Name:                 okta.PtrString("example"),
						Description:          okta.PtrString("an example group"),
						AdditionalProperties: map[string]interface{}{GroupProfileGovernorIDKey: "some-governor-id"},
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
						Name:                 okta.PtrString("example"),
						Description:          okta.PtrString("an example group"),
						AdditionalProperties: map[string]interface{}{GroupProfileGovernorIDKey: "some-governor-id"},
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
						AdditionalProperties: map[string]interface{}{GroupProfileGovernorIDKey: 12345},
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
						AdditionalProperties: map[string]interface{}{GroupProfileGovernorIDKey: ""},
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
		apiErr  bool
		apps    []okta.ListApplications200ResponseInner
		want    []string
		wantErr bool
	}{
		{
			name:    "example app list",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps:    []okta.ListApplications200ResponseInner{samlApp("app-01", nil), samlApp("app-02", nil)},
			want:    []string{"app-01", "app-02"},
		},
		{
			name:    "list error",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apiErr:  true,
			wantErr: true,
		},
		{
			name:    "empty groupid error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler

			switch {
			case tt.groupID == "":
				h = noCallHandler(t)
			case tt.apiErr:
				h = errorHandler()
			default:
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, tt.apps)
				})
			}

			c := newTestClient(t, h)

			got, err := c.listAssignedApplicationsForGroup(context.TODO(), tt.groupID)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, appIDs(got))
		})
	}
}

func TestClient_GroupGithubCloudApplications(t *testing.T) {
	tests := []struct {
		name    string
		groupID string
		apps    []okta.ListApplications200ResponseInner
		apiErr  bool
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "example app list",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps:    []okta.ListApplications200ResponseInner{samlApp("app-01", "test-org-01"), samlApp("app-02", nil)},
			want:    map[string]string{"test-org-01": "app-01"},
		},
		{
			name:    "non-string githubOrg",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps:    []okta.ListApplications200ResponseInner{samlApp("app-01", 1234), samlApp("app-02", nil)},
			want:    map[string]string{},
		},
		{
			name:    "example app list without github",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apps:    []okta.ListApplications200ResponseInner{samlApp("app-01", nil), samlApp("app-02", nil)},
			want:    map[string]string{},
		},
		{
			name:    "list error",
			groupID: "873121ec-646f-4e70-84ad-fd56db401631",
			apiErr:  true,
			wantErr: true,
		},
		{
			name:    "empty groupid error",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler

			switch {
			case tt.groupID == "":
				h = noCallHandler(t)
			case tt.apiErr:
				h = errorHandler()
			default:
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, tt.apps)
				})
			}

			c := newTestClient(t, h)

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
