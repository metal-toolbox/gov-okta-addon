package cmd

import (
	"testing"

	okt "github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/stretchr/testify/assert"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.uber.org/zap"
)

func Test_uniqueEmails(t *testing.T) {
	setupLogging()

	tests := []struct {
		name  string
		users []*okt.User
		want  map[string]string
	}{
		{
			name: "example external ids",
			users: []*okt.User{
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
						"email":       "oktaid1@boomsauce.com",
					},
				},
				{
					Id: "oktaid2",
					Profile: &okt.UserProfile{
						"pingSubject": "test2",
						"email":       "oktaid2@boomsauce.com",
					},
				},
				{
					Id: "oktaid3",
					Profile: &okt.UserProfile{
						"pingSubject": "test3",
						"email":       "oktaid3@boomsauce.com",
					},
				},
			},
			want: map[string]string{"oktaid1@boomsauce.com": "oktaid1", "oktaid2@boomsauce.com": "oktaid2", "oktaid3@boomsauce.com": "oktaid3"},
		},
		{
			name: "example non unique values",
			users: []*okt.User{
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
						"email":       "oktaid1@boomsauce.com",
					},
				},
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
						"email":       "oktaid1@boomsauce.com",
					},
				},
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
						"email":       "oktaid1@boomsauce.com",
					},
				},
			},
			want: map[string]string{"oktaid1@boomsauce.com": "oktaid1"},
		},
		{
			name: "example empty values",
			users: []*okt.User{
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
						"email":       "oktaid1@boomsauce.com",
					},
				},
				{
					Id: "",
					Profile: &okt.UserProfile{
						"pingSubject": "",
					},
				},
				{
					Id: "",
					Profile: &okt.UserProfile{
						"pingSubject": "",
					},
				},
			},
			want: map[string]string{"oktaid1@boomsauce.com": "oktaid1"},
		},
	}
	for _, tt := range tests {
		oc, _ := okta.NewClient(
			okta.WithLogger(zap.NewNop()),
		)

		t.Run(tt.name, func(t *testing.T) {
			got := uniqueEmails(oc, tt.users)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_userType(t *testing.T) {
	setupLogging()

	tests := []struct {
		name    string
		user    *okt.User
		want    string
		wantErr bool
	}{
		{
			name: "example userType",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"userType": "testUserType",
				},
			},
			want: "testUserType",
		},
		{
			name: "not found",
			user: &okt.User{
				Profile: &okt.UserProfile{},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"userType": 12345,
				},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := userType(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
