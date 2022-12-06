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

	for {
		if !resp.HasNextPage() {
			break
		}

		nextPage := []*okta.LogEvent{}

		resp, err = resp.Next(ctx, &nextPage)
		if err != nil {
			return nil, err
		}

		evtsResp = append(evtsResp, nextPage...)
	}

	return evtsResp, nil
}
