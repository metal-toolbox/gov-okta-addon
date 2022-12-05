package reconciler

import (
	"context"

	lp "go.equinixmetal.net/gov-okta-addon/internal/logpoll"
	"go.uber.org/zap"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
)

func (r *Reconciler) startEventLogPollerSubscriptions(ctx context.Context) {
	poller := lp.New(
		lp.WithLogger(r.logger),
		lp.WithOktaClient(r.oktaClient),
		lp.WithQueryParams(query.Params{
			Filter: "eventType eq \"user.lifecycle.create\"",
		}),
	)

	r.logger.Debug("starting event log polling", zap.Any("poller", poller))

	poller.Poll(ctx, r.oktaLogEventHandler)
}

func (r *Reconciler) oktaLogEventHandler(evt *okta.LogEvent) {
	r.logger.Debug("handling event from okta log", zap.String("okta.event.type", evt.EventType))
}
