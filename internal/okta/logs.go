package okta

import (
	"context"
	"time"

	"github.com/okta/okta-sdk-golang/v6/okta"
	"go.uber.org/zap"
)

const oktaTimeFormat = "2006-01-02T15:04:05Z"

// GetLogsBounded returns the okta log events bounded by since and until with the passed filter.  Note if we don't
// pass both since and until to okta, the API assumes this is a polling request and always returns a "NextPage".
func (c *Client) GetLogsBounded(ctx context.Context, since, until time.Time, filter string) ([]*okta.LogEvent, error) {
	c.logger.Debug("getting okta log events", zap.Time("events.since", since))

	sinceStr := since.Format(oktaTimeFormat)
	untilStr := until.Format(oktaTimeFormat)

	evtsResp := []*okta.LogEvent{}
	after := ""

	for {
		events, next, err := c.logEventIface.GetLogs(ctx, sinceStr, untilStr, after, filter, defaultPageLimit)
		if err != nil {
			return nil, err
		}

		for i := range events {
			evtsResp = append(evtsResp, &events[i])
		}

		if next == "" {
			break
		}

		after = next
	}

	return evtsResp, nil
}

// LogEventHandlerFn is a handler functions for a log event entry
type LogEventHandlerFn func(context.Context, *okta.LogEvent)

// PollLogs starts a goroutine that queries the okta event log api in "polling mode".
// https://developer.okta.com/docs/reference/api/system-log/#polling-requests
func (c *Client) PollLogs(ctx context.Context, interval time.Duration, start time.Time, filter string, handler LogEventHandlerFn) {
	go c.pollLogs(ctx, interval, start, filter, handler)
}

func (c *Client) pollLogs(ctx context.Context, interval time.Duration, start time.Time, filter string, handler LogEventHandlerFn) {
	sinceStr := start.Format(oktaTimeFormat)

	tick := time.NewTicker(interval)

	// after is the pagination cursor returned by okta; once we have one we follow it
	// instead of re-requesting from "since".
	after := ""

	for {
		select {
		case <-tick.C:
			c.logger.Debug("running poller loop")

			since := sinceStr
			if after != "" {
				since = ""
			}

			events, next, err := c.logEventIface.GetLogs(ctx, since, "", after, filter, defaultPageLimit)
			if err != nil {
				c.logger.Error("error getting log events from okta", zap.Error(err))
				continue
			}

			for i := range events {
				handler(ctx, &events[i])
			}

			after = next
		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}
