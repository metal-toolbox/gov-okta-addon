package okta

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockLogEventsClient struct {
	t   *testing.T
	err error

	logEvents []*okta.LogEvent

	resp *okta.Response
}

func (m *mockLogEventsClient) GetLogs(ctx context.Context, qp *query.Params) ([]*okta.LogEvent, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	s, err := time.Parse("2006-01-02T15:04:05Z", qp.Since)
	if err != nil {
		return nil, nil, err
	}

	resp := []*okta.LogEvent{}

	for _, e := range m.logEvents {
		if e.Published.Before(s) {
			continue
		}

		resp = append(resp, e)
	}

	return resp, m.resp, nil
}

func TestClient_GetLogSince(t *testing.T) {
	published := func(d time.Time) *time.Time {
		return &d
	}

	tests := []struct {
		name      string
		err       error
		logEvents []*okta.LogEvent
		since     time.Time
		qp        *query.Params
		want      []*okta.LogEvent
		wantErr   bool
	}{
		//nolint:gofumpt,govet
		{
			name: "example",
			logEvents: []*okta.LogEvent{
				{
					Actor:          &okta.LogActor{"system@okta.com", nil, "Okta System", "zzzzzzzzz", "SystemPrincipal"},
					EventType:      "user.lifecycle.create",
					DisplayMessage: "Create okta user",
					Published:      published(time.Date(2013, time.June, 19, 07, 14, 00, 00, time.UTC)),
				},
				{
					Actor:          &okta.LogActor{"system@okta.com", nil, "Okta System", "zzzzzzzzz", "SystemPrincipal"},
					EventType:      "user.lifecycle.create",
					DisplayMessage: "Create okta user",
					Published:      published(time.Date(2015, time.November, 20, 04, 40, 00, 00, time.UTC)),
				},
				{
					Actor:          &okta.LogActor{"system@okta.com", nil, "Okta System", "zzzzzzzzz", "SystemPrincipal"},
					EventType:      "user.lifecycle.create",
					DisplayMessage: "Create okta user",
					Published:      published(time.Date(2019, time.March, 28, 21, 21, 00, 00, time.UTC)),
				},
			},
			since: time.Date(2018, time.January, 01, 00, 00, 00, 00, time.UTC),
			want: []*okta.LogEvent{
				{
					Actor:          &okta.LogActor{"system@okta.com", nil, "Okta System", "zzzzzzzzz", "SystemPrincipal"},
					EventType:      "user.lifecycle.create",
					DisplayMessage: "Create okta user",
					Published:      published(time.Date(2019, time.March, 28, 21, 21, 00, 00, time.UTC)),
				},
			},
		},
		//nolint:gofumpt,govet,goerr113
		{
			name:      "error",
			logEvents: []*okta.LogEvent{},
			since:     time.Date(2018, time.January, 01, 00, 00, 00, 00, time.UTC),
			err:       errors.New("boomsauce"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger: zap.NewNop(),
				logEventIface: &mockLogEventsClient{
					t:         t,
					err:       tt.err,
					logEvents: tt.logEvents,
					resp:      &okta.Response{},
				},
			}
			got, err := c.GetLogSince(context.TODO(), tt.since, tt.qp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
