package okta

import (
	"context"
	"net/http"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
)

func TestClient_AssignGroupToApplication(t *testing.T) {
	tests := []struct {
		name    string
		appID   string
		groupID string
		apiErr  bool
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
			apiErr:  true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler

			switch {
			case tt.appID == "" || tt.groupID == "":
				h = noCallHandler(t)
			case tt.apiErr:
				h = errorHandler()
			default:
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, okta.ApplicationGroupAssignment{Id: okta.PtrString(tt.groupID)})
				})
			}

			c := newTestClient(t, h)

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
		apiErr  bool
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
			apiErr:  true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler

			switch {
			case tt.appID == "" || tt.groupID == "":
				h = noCallHandler(t)
			case tt.apiErr:
				h = errorHandler()
			default:
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusNoContent, nil)
				})
			}

			c := newTestClient(t, h)

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
		apiErr      bool
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
			apiErr:  true,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h http.Handler

			switch {
			case tt.appID == "":
				h = noCallHandler(t)
			case tt.apiErr:
				h = errorHandler()
			default:
				h = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					writeJSON(t, w, http.StatusOK, tt.assignments)
				})
			}

			c := newTestClient(t, h)

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
	t.Run("single page", func(t *testing.T) {
		apps := []okta.ListApplications200ResponseInner{
			samlApp("app-01", nil),
			samlApp("app-02", nil),
			samlApp("app-03", nil),
		}

		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, apps)
		}))

		got, err := c.listApplications(context.TODO(), "")
		assert.NoError(t, err)
		assert.Equal(t, []string{"app-01", "app-02", "app-03"}, appIDs(got))
	})

	t.Run("paginated", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("after") == "" {
				w.Header().Set("Link", `<https://test.okta.local/api/v1/apps?after=page2>; rel="next"`)
				writeJSON(t, w, http.StatusOK, []okta.ListApplications200ResponseInner{samlApp("app-01", nil), samlApp("app-02", nil)})

				return
			}

			writeJSON(t, w, http.StatusOK, []okta.ListApplications200ResponseInner{samlApp("app-03", nil)})
		}))

		got, err := c.listApplications(context.TODO(), "")
		assert.NoError(t, err)
		assert.Equal(t, []string{"app-01", "app-02", "app-03"}, appIDs(got))
	})

	t.Run("api error", func(t *testing.T) {
		c := newTestClient(t, errorHandler())

		_, err := c.listApplications(context.TODO(), "")
		assert.Error(t, err)
	})
}

func TestClient_GithubCloudApplications(t *testing.T) {
	tests := []struct {
		name    string
		apiErr  bool
		apps    []okta.ListApplications200ResponseInner
		want    map[string]string
		wantErr bool
	}{
		{
			name: "example apps",
			apps: []okta.ListApplications200ResponseInner{
				samlApp("app-01", "testorg01"),
				samlApp("app-02", "testorg02"),
				samlApp("app-03", nil),
				samlApp("app-05", []string{"some", "not", "string"}),
				samlApp("app-06", nil),
			},
			want: map[string]string{
				"testorg01": "app-01",
				"testorg02": "app-02",
			},
		},
		{
			name: "nil settings",
			apps: []okta.ListApplications200ResponseInner{
				samlApp("app-01", nil),
				samlApp("app-02", nil),
				samlApp("app-03", nil),
			},
			want: map[string]string{},
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
					writeJSON(t, w, http.StatusOK, tt.apps)
				})
			}

			c := newTestClient(t, h)

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
