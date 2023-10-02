package srv

import (
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
)

func TestServer_unmarshalPayload(t *testing.T) {
	tests := []struct {
		name    string
		message *nats.Msg
		want    *v1alpha1.Event
		wantErr bool
	}{
		{
			name: "example message",
			message: &nats.Msg{
				Subject: "foobar",
				Data:    []byte(`{"version": "v1", "action": "CREATE", "group_id": "12345"}`),
			},
			want: &v1alpha1.Event{
				Version: "v1",
				Action:  "CREATE",
				GroupID: "12345",
			},
		},
		{
			name: "bad payload",
			message: &nats.Msg{
				Subject: "foobar",
				Data:    []byte(`{`),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				Logger: zap.NewNop(),
			}
			got, err := s.unmarshalPayload(tt.message)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
