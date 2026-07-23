package okta

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"github.com/stretchr/testify/assert"
)

// testLogEvents returns a fixed set of user.lifecycle.create log events.
func testLogEvents() []okta.LogEvent {
	event := func(published time.Time) okta.LogEvent {
		return okta.LogEvent{
			EventType:      okta.PtrString("user.lifecycle.create"),
			DisplayMessage: okta.PtrString("Create okta user"),
			Published:      okta.PtrTime(published),
		}
	}

	return []okta.LogEvent{
		event(time.Date(2013, time.June, 19, 7, 14, 0, 0, time.UTC)),
		event(time.Date(2015, time.November, 20, 4, 40, 0, 0, time.UTC)),
		event(time.Date(2019, time.March, 28, 21, 21, 0, 0, time.UTC)),
	}
}

func TestClient_GetLogsBounded(t *testing.T) {
	since := time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("single page", func(t *testing.T) {
		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(t, w, http.StatusOK, testLogEvents())
		}))

		got, err := c.GetLogsBounded(context.TODO(), since, time.Now().UTC(), "")
		assert.NoError(t, err)
		assert.Len(t, got, 3)

		for _, e := range got {
			assert.Equal(t, "user.lifecycle.create", e.GetEventType())
		}
	})

	t.Run("paginated", func(t *testing.T) {
		events := testLogEvents()

		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("after") == "" {
				w.Header().Set("Link", `<https://test.okta.local/api/v1/logs?after=page2>; rel="next"`)
				writeJSON(t, w, http.StatusOK, events[:2])

				return
			}

			writeJSON(t, w, http.StatusOK, events[2:])
		}))

		got, err := c.GetLogsBounded(context.TODO(), since, time.Now().UTC(), "")
		assert.NoError(t, err)
		assert.Len(t, got, 3)
	})

	t.Run("api error", func(t *testing.T) {
		c := newTestClient(t, errorHandler())

		_, err := c.GetLogsBounded(context.TODO(), since, time.Now().UTC(), "")
		assert.Error(t, err)
	})
}

func TestClient_pollLogs(t *testing.T) {
	start := time.Date(2011, time.September, 20, 15, 15, 0, 0, time.UTC)

	t.Run("collects events", func(t *testing.T) {
		var calls atomic.Int64

		c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Link", `<https://test.okta.local/api/v1/logs?after=cursor>; rel="next"`)

			// only the first poll returns events; subsequent polls are empty
			if calls.Add(1) == 1 {
				writeJSON(t, w, http.StatusOK, testLogEvents())
				return
			}

			writeJSON(t, w, http.StatusOK, []okta.LogEvent{})
		}))

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()

		events := []*okta.LogEvent{}

		// pollLogs blocks until ctx is done, appending synchronously via the handler
		c.pollLogs(ctx, time.Millisecond, start, "", func(_ context.Context, le *okta.LogEvent) {
			events = append(events, le)
		})

		assert.Len(t, events, 3)
	})

	t.Run("api error collects nothing", func(t *testing.T) {
		c := newTestClient(t, errorHandler())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		events := []*okta.LogEvent{}

		c.pollLogs(ctx, time.Millisecond, start, "", func(_ context.Context, le *okta.LogEvent) {
			events = append(events, le)
		})

		assert.Empty(t, events)
	})
}
