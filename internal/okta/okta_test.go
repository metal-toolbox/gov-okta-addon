package okta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newTestClient returns a Client wired to an httptest server running the given handler.
// The okta v6 SDK derives its request host/scheme from the configured org URL, so we point
// those exported config fields at the test server (the SDK never enforces the https check).
func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	require.NoError(t, err)

	config, err := okta.NewConfiguration(
		okta.WithOrgUrl("https://test.okta.local"),
		okta.WithToken("test-token"),
		okta.WithCache(false),
	)
	require.NoError(t, err)

	config.Host = u.Host
	config.Scheme = u.Scheme
	config.HTTPClient = srv.Client()

	return &Client{
		logger: zap.NewNop(),
		client: okta.NewAPIClient(config),
	}
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(t *testing.T, w http.ResponseWriter, status int, v interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if v == nil {
		return
	}

	require.NoError(t, json.NewEncoder(w).Encode(v))
}

// errorHandler responds with a 500 to every request, exercising the SDK error path.
func errorHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorSummary":"boom"}`))
	})
}

// noCallHandler fails the test if the okta API is called; used to assert that client-side
// validation short-circuits before any request is made.
func noCallHandler(t *testing.T) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected okta api call: %s %s", r.Method, r.URL.Path)
	})
}

// samlApp builds an okta application (as the oneOf list response wrapper) with the given id
// and, optionally, a githubOrg app setting.  signOnMode is set so the SDK's oneOf decoder
// round-trips the JSON back into a SamlApplication.
func samlApp(id string, githubOrg interface{}) okta.ListApplications200ResponseInner {
	app := &okta.SamlApplication{
		Application: okta.Application{
			Id:         okta.PtrString(id),
			SignOnMode: "SAML_2_0",
			Label:      "test app " + id,
		},
	}

	if githubOrg != nil {
		app.Settings = &okta.SamlApplicationSettings{
			AdditionalProperties: map[string]interface{}{
				"app": map[string]interface{}{"githubOrg": githubOrg},
			},
		}
	}

	return okta.ListApplications200ResponseInner{SamlApplication: app}
}

// appIDs extracts the okta ids from a list of application response wrappers.
func appIDs(apps []okta.ListApplications200ResponseInner) []string {
	ids := make([]string, 0, len(apps))
	for _, a := range apps {
		id, _, _ := appGithubOrg(a)
		ids = append(ids, id)
	}

	return ids
}

// userIDs extracts the okta ids from a list of users.
func userIDs(users []*okta.User) []string {
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = u.GetId()
	}

	return ids
}

// groupIDs extracts the okta ids from a list of groups.
func groupIDs(groups []*okta.Group) []string {
	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.GetId()
	}

	return ids
}
