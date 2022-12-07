// Package logpoll runs a long poll against the okta eventlog
package logpoll

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"github.com/stretchr/testify/assert"

	okt "go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.uber.org/zap"
)

type moktaclient struct {
	t   *testing.T
	err error
}

var (
	testEvents = []*okta.LogEvent{
		{DisplayMessage: "test1"},
		{DisplayMessage: "test2"},
		{DisplayMessage: "test3"},
		{DisplayMessage: "test4"},
		{DisplayMessage: "test5"},
	}

	testTime = time.Now().UTC()
)

func (m *moktaclient) GetLogsBounded(ctx context.Context, since, until time.Time, _ *query.Params) ([]*okta.LogEvent, error) {
	if m.err != nil {
		return nil, m.err
	}

	// this logic is backwards from real okta events because
	// we only want to return the test events the first time
	if since.After(testTime) {
		return []*okta.LogEvent{}, nil
	}

	return testEvents, nil
}

func TestNew(t *testing.T) {
	testTime := time.Now().UTC().Add(-1 * time.Hour)

	tests := []struct {
		name string
		opts []Option
		want *LogPoller
	}{
		{
			name: "new log poller min opts",
			opts: []Option{
				WithLastRun(testTime),
			},
			want: &LogPoller{
				logger:   zap.NewNop(),
				interval: 30 * time.Second,
				last:     testTime,
			},
		},
		{
			name: "new log poller override query params",
			opts: []Option{
				WithInterval(10 * time.Second),
				WithLastRun(testTime),
				// this is the default, explicit for test clarity
				WithLogger(zap.NewNop()),
				WithOktaClient(&okt.Client{}),
				WithQueryParams(query.Params{
					Filter: "eventType eq \"user.lifecycle.create\"",
				}),
			},
			want: &LogPoller{
				logger:     zap.NewNop(),
				interval:   10 * time.Second,
				last:       testTime,
				oktaClient: &okt.Client{},
				queryParams: query.Params{
					Filter: "eventType eq \"user.lifecycle.create\"",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.opts...)

			assert.IsType(t, &LogPoller{}, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLogPoller_poll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	lp := New(
		WithLastRun(testTime),
		WithInterval(1*time.Microsecond),
		WithOktaClient(&moktaclient{t, nil}),
	)

	events := []*okta.LogEvent{}

	lp.poll(ctx, func(_ context.Context, le *okta.LogEvent) {
		events = append(events, le)
	})

	<-ctx.Done()

	assert.Equal(t, testEvents, events)

	errCtx, errCancel := context.WithTimeout(context.TODO(), 1*time.Second)
	defer errCancel()

	lpErr := New(
		WithLastRun(testTime),
		WithInterval(1*time.Microsecond),
		WithOktaClient(&moktaclient{t, errors.New("boomsauce")}), //nolint:goerr113
	)

	errEvents := []*okta.LogEvent{}

	lpErr.poll(errCtx, func(_ context.Context, le *okta.LogEvent) {
		errEvents = append(errEvents, le)
	})

	<-errCtx.Done()

	assert.Equal(t, []*okta.LogEvent{}, errEvents)
}
