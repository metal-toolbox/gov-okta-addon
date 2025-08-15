package okta

import (
	"context"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

// GetLogsBounded returns the okta log events bounded by since and until with the passed query parameters.  Note if we don't
// pass both since and until to okta, the API assumes this is a polling request and always returns a "NextPage".
func (c *Client) GetLogsBounded(ctx context.Context, since, until time.Time, qp *query.Params) ([]*okta.LogEvent, error) {
	if qp == nil {
		qp = &query.Params{}
	}

	c.logger.Debug("getting okta log events", zap.Time("events.since", since))

	qp.Since = since.Format("2006-01-02T15:04:05Z")
	qp.Until = until.Format("2006-01-02T15:04:05Z")
	qp.Limit = defaultPageLimit

	events, resp, err := c.logEventIface.GetLogs(ctx, qp)
	if err != nil {
		return nil, err
	}

	evtsResp := events

	for resp.HasNextPage() {
		nextPage := []*okta.LogEvent{}

		resp, err = resp.Next(ctx, &nextPage)
		if err != nil {
			return nil, err
		}

		evtsResp = append(evtsResp, nextPage...)
	}

	return evtsResp, nil
}

// LogEventHandlerFn is a handler functions for a log event entry
type LogEventHandlerFn func(context.Context, *okta.LogEvent)

// PollLogs starts a goroutine that queries the okta event log api in "polling mode".
// https://developer.okta.com/docs/reference/api/system-log/#polling-requests
func (c *Client) PollLogs(ctx context.Context, interval time.Duration, start time.Time, qp *query.Params, handler LogEventHandlerFn) {
	go c.pollLogs(ctx, interval, start, qp, handler)
}

func (c *Client) pollLogs(ctx context.Context, interval time.Duration, start time.Time, qp *query.Params, handler LogEventHandlerFn) {
	if qp == nil {
		qp = &query.Params{}
	}

	qp.Since = start.Format("2006-01-02T15:04:05Z")

	tick := time.NewTicker(interval)

	var resp *okta.Response

	for {
		select {
		case <-tick.C:
			c.logger.Debug("running poller loop")

			var err error

			events := []*okta.LogEvent{}

			if resp == nil {
				events, resp, err = c.logEventIface.GetLogs(ctx, qp)
				if err != nil {
					c.logger.Error("error getting log events from okta", zap.Error(err))
					continue
				}
			} else {
				resp, err = resp.Next(ctx, &events)
				if err != nil {
					c.logger.Error("error calling next log events from okta", zap.Error(err))
					continue
				}
			}

			for _, evt := range events {
				handler(ctx, evt)
			}
		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}
