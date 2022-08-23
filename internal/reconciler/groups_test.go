package reconciler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.equinixmetal.net/governor/pkg/api/v1alpha"
)

var testOrganizationsList = []byte(`
[
	{
		"id": "7b1e8b5a-17ad-454f-ba4f-841191b70d44",
		"name": "Pajama Party",
		"created_at": "2001-01-01T01:01:01.668476Z",
		"updated_at": "2001-01-01T01:01:01.668476Z",
		"slug": "pajama-party"
	},
	{
		"id": "3c2738c4-75ce-4df6-9f58-bac6d4372634",
		"name": "Costume Party",
		"created_at": "2001-01-01T01:01:01.668476Z",
		"updated_at": "2001-01-01T01:01:01.668476Z",
		"slug": "costume-party"
	},
	{
		"id": "dd934f26-fc25-4e39-984d-1c7bff566bac",
		"name": "Pizza Party",
		"created_at": "2001-01-01T01:01:01.668476Z",
		"updated_at": "2001-01-01T01:01:01.668476Z",
		"slug": "pizza-party"
	},
	{
		"id": "530d5e91-52ab-462c-8ef6-77b2e737713d",
		"name": "Pool Party",
		"created_at": "2001-01-01T01:01:01.668476Z",
		"updated_at": "2001-01-01T01:01:01.668476Z",
		"slug": "pool-party"
	},
	{
		"id": "18367178-22b5-41de-8828-06d9b84c8b0a",
		"name": "Dance Party",
		"created_at": "2001-01-01T01:01:01.668476Z",
		"updated_at": "2001-01-01T01:01:01.668476Z",
		"slug": "dance-party"
	}
]
`)

func testOrganizationSlice(t *testing.T) []*v1alpha.Organization {
	out := []*v1alpha.Organization{}

	if err := json.Unmarshal(testOrganizationsList, &out); err != nil {
		assert.NoError(t, err)
	}

	return out
}

func Test_getGroupOrgSlugs(t *testing.T) {
	tests := []struct {
		name  string
		group *v1alpha.Group
		orgs  []*v1alpha.Organization
		want  []string
	}{
		{
			name: "example org slugs",
			group: &v1alpha.Group{
				Organizations: []string{
					"7b1e8b5a-17ad-454f-ba4f-841191b70d44",
					"dd934f26-fc25-4e39-984d-1c7bff566bac",
				},
			},
			orgs: testOrganizationSlice(t),
			want: []string{
				"pajama-party",
				"pizza-party",
			},
		},
		{
			name:  "empty group orgs list",
			group: &v1alpha.Group{},
			orgs:  testOrganizationSlice(t),
			want:  []string{},
		},
		{
			name: "empty governor orgs list",
			group: &v1alpha.Group{
				Organizations: []string{
					"7b1e8b5a-17ad-454f-ba4f-841191b70d44",
					"dd934f26-fc25-4e39-984d-1c7bff566bac",
				},
			},
			orgs: []*v1alpha.Organization{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getGroupOrgSlugs(tt.group, tt.orgs)
			assert.Equal(t, tt.want, got)
		})
	}
}
