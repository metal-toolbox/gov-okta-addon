package reconciler

import (
	"testing"

	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func Test_contains(t *testing.T) {
	type args struct {
		list []string
		item string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "example found",
			args: args{
				list: []string{"foo", "bar", "baz"},
				item: "foo",
			},
			want: true,
		},
		{
			name: "example not found",
			args: args{
				list: []string{"foo", "bar", "baz"},
				item: "boz",
			},
			want: false,
		},
		{
			name: "empty list",
			args: args{
				list: []string{},
				item: "boz",
			},
			want: false,
		},
		{
			name: "nil list",
			args: args{
				item: "boz",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.list, tt.args.item); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_containsOrg(t *testing.T) {
	tests := []struct {
		name string
		org  string
		orgs []*v1alpha1.Organization
		want bool
	}{
		{
			name: "org in orgs list",
			org:  "pajama-party",
			orgs: testOrganizationSlice(t),
			want: true,
		},
		{
			name: "org not in orgs list",
			org:  "no-party",
			orgs: testOrganizationSlice(t),
			want: false,
		},
		{
			name: "blank org",
			org:  "",
			orgs: testOrganizationSlice(t),
			want: false,
		},
		{
			name: "empty orgs list",
			org:  "pajama-party",
			orgs: []*v1alpha1.Organization{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsOrg(tt.org, tt.orgs)
			assert.Equal(t, tt.want, got)
		})
	}
}
