package okta

import (
	"context"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

// GetLogSince returns the okta log events since the given time and with the passed query parameters
func (c *Client) GetLogSince(ctx context.Context, since time.Time, qp *query.Params) ([]*okta.LogEvent, error) {
	if qp == nil {
		qp = &query.Params{}
	}

	c.logger.Debug("getting okta log events", zap.Time("events.since", since))

	qp.Since = since.Format("2006-01-02T15:04:05Z")

	if c.logEventIface == nil {
		c.logger.Warn("yo its nil")
	}

	events, resp, err := c.logEventIface.GetLogs(ctx, qp)
	if err != nil {
		return nil, err
	}

	for {
		if !resp.HasNextPage() {
			break
		}

		nextPage := []*okta.LogEvent{}

		resp, err = resp.Next(ctx, &nextPage)
		if err != nil {
			return nil, err
		}

		events = append(events, nextPage...)
	}

	return events, nil
}
