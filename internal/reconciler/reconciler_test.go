package reconciler

import (
	"testing"
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
