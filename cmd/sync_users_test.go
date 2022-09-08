package cmd

import (
	"testing"

	okt "github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/stretchr/testify/assert"
)

func Test_uniqueExternalIDs(t *testing.T) {
	setupLogging()

	tests := []struct {
		name  string
		users []*okt.User
		want  map[string]struct{}
	}{
		{
			name: "example external ids",
			users: []*okt.User{
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
					},
				},
				{
					Id: "oktaid2",
					Profile: &okt.UserProfile{
						"pingSubject": "test2",
					},
				},
				{
					Id: "oktaid3",
					Profile: &okt.UserProfile{
						"pingSubject": "test3",
					},
				},
			},
			want: map[string]struct{}{"okta|oktaid1": {}, "okta|oktaid2": {}, "okta|oktaid3": {}},
		},
		{
			name: "example non unique values",
			users: []*okt.User{
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
					},
				},
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
					},
				},
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
					},
				},
			},
			want: map[string]struct{}{"okta|oktaid1": {}},
		},
		{
			name: "example empty values",
			users: []*okt.User{
				{
					Id: "oktaid1",
					Profile: &okt.UserProfile{
						"pingSubject": "test1",
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
			want: map[string]struct{}{"okta|oktaid1": {}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueExternalIDs(tt.users)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_email(t *testing.T) {
	setupLogging()

	tests := []struct {
		name    string
		user    *okt.User
		want    string
		wantErr bool
	}{
		{
			name: "example email",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"email": "test1@test.com",
				},
			},
			want: "test1@test.com",
		},
		{
			name: "not found",
			user: &okt.User{
				Profile: &okt.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"email": 12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := email(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_externalID(t *testing.T) {
	setupLogging()

	tests := []struct {
		name    string
		user    *okt.User
		want    string
		wantErr bool
	}{
		{
			name: "example external id",
			user: &okt.User{
				Id: "oktaid1",
				Profile: &okt.UserProfile{
					"pingSubject": "test1",
				},
			},
			want: "oktaid1",
		},
		{
			name:    "not found",
			user:    &okt.User{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := externalID(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
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

func Test_firstName(t *testing.T) {
	setupLogging()

	tests := []struct {
		name    string
		user    *okt.User
		want    string
		wantErr bool
	}{
		{
			name: "example firstName",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"firstName": "Test",
				},
			},
			want: "Test",
		},
		{
			name: "not found",
			user: &okt.User{
				Profile: &okt.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"firstName": 12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := firstName(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_lastName(t *testing.T) {
	setupLogging()

	tests := []struct {
		name    string
		user    *okt.User
		want    string
		wantErr bool
	}{
		{
			name: "example lastName",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"lastName": "One",
				},
			},
			want: "One",
		},
		{
			name: "not found",
			user: &okt.User{
				Profile: &okt.UserProfile{},
			},
			wantErr: true,
		},
		{
			name: "bad values",
			user: &okt.User{
				Profile: &okt.UserProfile{
					"lastName": 12345,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := lastName(tt.user)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
