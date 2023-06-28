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

//nolint:gofumpt,govet
var (
	testEvents = []*okta.LogEvent{
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
	}
)

type mockLogEventsClient struct {
	t   *testing.T
	err error

	logEvents []*okta.LogEvent

	maxIter int
	iter    int

	resp *okta.Response
}

func (m *mockLogEventsClient) GetLogs(_ context.Context, qp *query.Params) ([]*okta.LogEvent, *okta.Response, error) {
	if m.err != nil {
		return nil, nil, m.err
	}

	s, err := time.Parse("2006-01-02T15:04:05Z", qp.Since)
	if err != nil {
		return nil, nil, err
	}

	events := []*okta.LogEvent{}

	if m.iter < m.maxIter {
		for _, e := range m.logEvents {
			if e.Published.Before(s) {
				continue
			}

			events = append(events, e)
		}
	}

	m.iter++

	return events, &okta.Response{}, nil
}

func published(d time.Time) *time.Time {
	return &d
}

func TestClient_GetLogsBounded(t *testing.T) {
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
			name:      "example",
			logEvents: testEvents,
			since:     time.Date(2018, time.January, 01, 00, 00, 00, 00, time.UTC),
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
					maxIter:   10,
				},
			}
			got, err := c.GetLogsBounded(context.TODO(), tt.since, time.Now().UTC(), tt.qp)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_pollLogs(t *testing.T) {
	testTime := time.Date(2011, time.September, 20, 15, 15, 00, 00, time.UTC) //nolint:gofumpt

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	client := &Client{
		logger: zap.NewNop(),
		logEventIface: &mockLogEventsClient{
			t:         t,
			logEvents: testEvents,
			maxIter:   1,
		},
	}

	events := []*okta.LogEvent{}

	client.pollLogs(
		ctx,
		1*time.Microsecond,
		testTime,
		nil,
		func(_ context.Context, le *okta.LogEvent) {
			events = append(events, le)
		},
	)

	<-ctx.Done()

	assert.Equal(t, testEvents, events)

	errCtx, errCancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer errCancel()

	errClient := &Client{
		logger: zap.NewNop(),
		logEventIface: &mockLogEventsClient{
			t:         t,
			logEvents: testEvents,
			err:       errors.New("boomsauce"), //nolint:goerr113
			maxIter:   1,
		},
	}

	errEvents := []*okta.LogEvent{}

	errClient.pollLogs(
		ctx,
		1*time.Microsecond,
		testTime,
		nil,
		func(_ context.Context, le *okta.LogEvent) {
			events = append(events, le)
		},
	)

	<-errCtx.Done()

	assert.Equal(t, []*okta.LogEvent{}, errEvents)
}
