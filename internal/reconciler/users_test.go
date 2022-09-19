package reconciler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/volatiletech/null/v8"
	"go.equinixmetal.net/governor/pkg/api/v1alpha1"
)

func Test_userDeleted(t *testing.T) {
	testResp := func(r []byte) *v1alpha1.User {
		resp := v1alpha1.User{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		return &resp
	}

	testRespWithTime := func(r []byte, dt time.Time) *v1alpha1.User {
		resp := v1alpha1.User{}
		if err := json.Unmarshal(r, &resp); err != nil {
			t.Error(err)
		}

		resp.DeletedAt = null.TimeFrom(dt)

		return &resp
	}

	type args struct {
		user *v1alpha1.User
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "user deleted just now",
			args: args{
				user: testRespWithTime([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       "bob@example.com"
				}`), time.Now()),
			},
			want: true,
		},
		{
			name: "user deleted 23h ago",
			args: args{
				user: testRespWithTime([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       "bob@example.com"
				}`), time.Now().Add(-23*time.Hour)),
			},
			want: true,
		},
		{
			name: "user deleted 25h ago",
			args: args{
				user: testRespWithTime([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       "bob@example.com"
				}`), time.Now().Add(-25*time.Hour)),
			},
			want: false,
		},
		{
			name: "user deleted long time ago",
			args: args{
				user: testResp([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       "bob@example.com",
					"deleted_at":  "2022-08-11T14:38:33.027346Z"
				}`)),
			},
			want: false,
		},
		{
			name: "user not deleted",
			args: args{
				user: testResp([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       "bob@example.com"
				}`)),
			},
			want: false,
		},
		{
			name: "user nil",
			args: args{
				user: nil,
			},
			want: false,
		},
		{
			name: "user missing id",
			args: args{
				user: testResp([]byte(`{
					"id":          "",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       "bob@example.com"
				}`)),
			},
			want: false,
		},
		{
			name: "user missing external_id",
			args: args{
				user: testResp([]byte(`{
					"id":          "012345",
					"external_id": "",
					"name":        "bob",
					"email":       "bob@example.com"
				}`)),
			},
			want: false,
		},
		{
			name: "user missing name",
			args: args{
				user: testResp([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "",
					"email":       "bob@example.com"
				}`)),
			},
			want: false,
		},
		{
			name: "user missing email",
			args: args{
				user: testResp([]byte(`{
					"id":          "012345",
					"external_id": "ext012345",
					"name":        "bob",
					"email":       ""
				}`)),
			},
			want: false,
		},
		{
			name: "user missing multiple fields",
			args: args{
				user: testResp([]byte(`{
					"id":          "012345"
				}`)),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := userDeleted(tt.args.user); got != tt.want {
				t.Errorf("userDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}
