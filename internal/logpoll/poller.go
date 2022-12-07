// Package logpoll runs a long poll against the okta eventlog
package logpoll

import (
	"context"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"github.com/okta/okta-sdk-golang/v2/okta/query"
	"go.uber.org/zap"
)

var (
	// defaultPollerInterval is the default for how often to poll for new events
	defaultPollerInterval = 30 * time.Second
	// defaultColdStartLookback is the default for how far back to go for events on a cold start
	defaultColdStartLookback = 6 * time.Hour
)

// LogEventHandlerFn is a handler functions for a log event entry
type LogEventHandlerFn func(*okta.LogEvent)

type oktaIface interface {
	GetLogsBounded(context.Context, time.Time, time.Time, *query.Params) ([]*okta.LogEvent, error)
}

// LogPoller is the control structure for long polling okta event logs
type LogPoller struct {
	interval    time.Duration
	last        time.Time
	logger      *zap.Logger
	oktaClient  oktaIface
	queryParams query.Params
}

// Option is a functional configuration option
type Option func(r *LogPoller)

// WithInterval sets the reconciler interval
func WithInterval(i time.Duration) Option {
	return func(r *LogPoller) {
		r.interval = i
	}
}

// WithLastRun sets the "since" time for the first call to okta
func WithLastRun(t time.Time) Option {
	return func(r *LogPoller) {
		r.last = t
	}
}

// WithLogger sets logger
func WithLogger(l *zap.Logger) Option {
	return func(r *LogPoller) {
		r.logger = l
	}
}

// WithOktaClient sets okta client
func WithOktaClient(o oktaIface) Option {
	return func(r *LogPoller) {
		r.oktaClient = o
	}
}

// WithQueryParams allows passing optional okta query params. Some
// query params will get overwritten in the calls to okta.
func WithQueryParams(p query.Params) Option {
	return func(r *LogPoller) {
		r.queryParams = p
	}
}

// New returns a new log poller
func New(opts ...Option) *LogPoller {
	lp := LogPoller{
		interval:    defaultPollerInterval,
		last:        time.Now().UTC().Add(-defaultColdStartLookback),
		logger:      zap.NewNop(),
		queryParams: query.Params{},
	}

	for _, opt := range opts {
		opt(&lp)
	}

	lp.logger.Debug("returning new log poller", zap.Any("poller", lp))

	return &lp
}

// Poll starts the long poll of okta's event log, calling the handler function for each entry
func (p *LogPoller) Poll(ctx context.Context, handler LogEventHandlerFn) {
	go p.poll(ctx, handler)
}

func (p *LogPoller) poll(ctx context.Context, handler LogEventHandlerFn) {
	tick := time.NewTicker(p.interval)

	for {
		select {
		case <-tick.C:
			p.logger.Debug("running poller loop")

			qTime := time.Now().UTC()

			events, err := p.oktaClient.GetLogsBounded(ctx, p.last, qTime, &p.queryParams)
			if err != nil {
				p.logger.Error("error getting log events from okta", zap.Error(err))
				continue
			}

			p.last = qTime

			for _, evt := range events {
				handler(evt)
			}
		case <-ctx.Done():
			tick.Stop()
			return
		}
	}
}
