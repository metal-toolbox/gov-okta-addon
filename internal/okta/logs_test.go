package okta

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func testActor() *okta.LogActor {
	return &okta.LogActor{
		AlternateId: okta.PtrString("system@okta.com"),
		DisplayName: okta.PtrString("Okta System"),
		Id:          okta.PtrString("zzzzzzzzz"),
		Type:        okta.PtrString("SystemPrincipal"),
	}
}

func logEvent(published time.Time) okta.LogEvent {
	return okta.LogEvent{
		Actor:          testActor(),
		EventType:      okta.PtrString("user.lifecycle.create"),
		DisplayMessage: okta.PtrString("Create okta user"),
		Published:      okta.PtrTime(published),
	}
}

var testEvents = []okta.LogEvent{
	logEvent(time.Date(2013, time.June, 19, 0o7, 14, 0o0, 0o0, time.UTC)),
	logEvent(time.Date(2015, time.November, 20, 0o4, 40, 0o0, 0o0, time.UTC)),
	logEvent(time.Date(2019, time.March, 28, 21, 21, 0o0, 0o0, time.UTC)),
}

type mockLogEventsClient struct {
	t   *testing.T
	err error

	logEvents []okta.LogEvent

	maxIter int
	iter    int
}

func (m *mockLogEventsClient) GetLogs(_ context.Context, since, _, _, _ string, _ int32) ([]okta.LogEvent, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}

	s, err := time.Parse(oktaTimeFormat, since)
	if err != nil {
		return nil, "", err
	}

	events := []okta.LogEvent{}

	if m.iter < m.maxIter {
		for _, e := range m.logEvents {
			if e.GetPublished().Before(s) {
				continue
			}

			events = append(events, e)
		}
	}

	m.iter++

	return events, "", nil
}

func TestClient_GetLogsBounded(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		logEvents []okta.LogEvent
		since     time.Time
		want      []*okta.LogEvent
		wantErr   bool
	}{
		{
			name:      "example",
			logEvents: testEvents,
			since:     time.Date(2018, time.January, 0o1, 0o0, 0o0, 0o0, 0o0, time.UTC),
			want: []*okta.LogEvent{
				{
					Actor:          testActor(),
					EventType:      okta.PtrString("user.lifecycle.create"),
					DisplayMessage: okta.PtrString("Create okta user"),
					Published:      okta.PtrTime(time.Date(2019, time.March, 28, 21, 21, 0o0, 0o0, time.UTC)),
				},
			},
		},
		{
			name:      "error",
			logEvents: []okta.LogEvent{},
			since:     time.Date(2018, time.January, 0o1, 0o0, 0o0, 0o0, 0o0, time.UTC),
			err:       errors.New("boomsauce"), //nolint:err113
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
					maxIter:   10,
				},
			}

			got, err := c.GetLogsBounded(context.TODO(), tt.since, time.Now().UTC(), "")
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
	testTime := time.Date(2011, time.September, 20, 15, 15, 0o0, 0o0, time.UTC)

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
		"",
		func(_ context.Context, le *okta.LogEvent) {
			events = append(events, le)
		},
	)

	<-ctx.Done()

	want := make([]*okta.LogEvent, len(testEvents))
	for i := range testEvents {
		want[i] = &testEvents[i]
	}

	assert.Equal(t, want, events)

	errCtx, errCancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer errCancel()

	errClient := &Client{
		logger: zap.NewNop(),
		logEventIface: &mockLogEventsClient{
			t:         t,
			logEvents: testEvents,
			err:       errors.New("boomsauce"), //nolint:err113
			maxIter:   1,
		},
	}

	errEvents := []*okta.LogEvent{}

	errClient.pollLogs(
		ctx,
		1*time.Microsecond,
		testTime,
		"",
		func(_ context.Context, le *okta.LogEvent) {
			errEvents = append(errEvents, le)
		},
	)

	<-errCtx.Done()

	assert.Equal(t, []*okta.LogEvent{}, errEvents)
}
