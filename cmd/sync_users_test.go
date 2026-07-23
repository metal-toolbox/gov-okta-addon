package cmd

import (
	"testing"

	okt "github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
)

// userWithEmail builds an okta user with the given id and email.
func userWithEmail(id, email string) *okt.User {
	u := &okt.User{Id: okt.PtrString(id), Profile: &okt.UserProfile{}}
	if email != "" {
		u.Profile.Email = okt.PtrString(email)
	}

	return u
}

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
				userWithEmail("oktaid1", "oktaid1@boomsauce.com"),
				userWithEmail("oktaid2", "oktaid2@boomsauce.com"),
				userWithEmail("oktaid3", "oktaid3@boomsauce.com"),
			},
			want: map[string]string{"oktaid1@boomsauce.com": "oktaid1", "oktaid2@boomsauce.com": "oktaid2", "oktaid3@boomsauce.com": "oktaid3"},
		},
		{
			name: "example non unique values",
			users: []*okt.User{
				userWithEmail("oktaid1", "oktaid1@boomsauce.com"),
				userWithEmail("oktaid1", "oktaid1@boomsauce.com"),
				userWithEmail("oktaid1", "oktaid1@boomsauce.com"),
			},
			want: map[string]string{"oktaid1@boomsauce.com": "oktaid1"},
		},
		{
			name: "example empty values",
			users: []*okt.User{
				userWithEmail("oktaid1", "oktaid1@boomsauce.com"),
				userWithEmail("", ""),
				userWithEmail("", ""),
			},
			want: map[string]string{"oktaid1@boomsauce.com": "oktaid1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueEmails(tt.users)
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
					UserType: *okt.NewNullableString(okt.PtrString("testUserType")),
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
