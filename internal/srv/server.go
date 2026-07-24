package srv

import (
	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventprocessor"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"

	"github.com/metal-toolbox/gov-okta-addon/internal/reconciler"
)

// Processor handles governor events for the okta addon
type Processor struct {
	logger     *zap.Logger
	tracer     trace.Tracer
	reconciler *reconciler.Reconciler
}

// Processor implements the [eventprocessor.EventProcessor] interface
var _ eventprocessor.EventProcessor = (*Processor)(nil)

// Option is a function that configures a Processor
type Option func(*Processor)

// NewProcessor creates a new events processor
func NewProcessor(rec *reconciler.Reconciler, opts ...Option) *Processor {
	p := &Processor{
		reconciler: rec,
		logger:     zap.NewNop(),
		tracer:     noop.NewTracerProvider().Tracer("events-processor"),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithLogger configures the logger for the Processor
func WithLogger(logger *zap.Logger) Option {
	return func(p *Processor) {
		p.logger = logger
	}
}

// WithTracer configures the tracer for the Processor
func WithTracer(tracer trace.Tracer) Option {
	return func(p *Processor) {
		p.tracer = tracer
	}
}

// Register wires up the governor event handlers on the event router. The
// gov-okta-addon does not use governor's interactive extension capability, so
// the extension object is ignored.
func (p *Processor) Register(er eventrouter.EventRouter, _ *v1alpha1.Extension) {
	p.logger.Info("registering governor event handlers")

	// group events: a governor group created/updated/deleted
	er.Create(govevents.GovernorGroupsEventSubject, p.GroupCreate, p.auditMiddleware)
	er.Update(govevents.GovernorGroupsEventSubject, p.GroupUpdate, p.auditMiddleware)
	er.Delete(govevents.GovernorGroupsEventSubject, p.GroupDelete, p.auditMiddleware)

	// group membership events: a member added/removed from a group
	er.Create(govevents.GovernorMembersEventSubject, p.MemberCreate, p.auditMiddleware)
	er.Delete(govevents.GovernorMembersEventSubject, p.MemberDelete, p.auditMiddleware)

	// user events: a governor user updated/deleted
	er.Update(govevents.GovernorUsersEventSubject, p.UserUpdate, p.auditMiddleware)
	er.Delete(govevents.GovernorUsersEventSubject, p.UserDelete, p.auditMiddleware)
}
