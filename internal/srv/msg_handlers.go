package srv

import (
	"context"

	"github.com/metal-toolbox/auditevent"
	"github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"

	"github.com/metal-toolbox/gov-okta-addon/internal/auctx"
)

// GroupCreate handles a governor group create event: it ensures the group
// exists in okta, reconciles its application assignments and its membership.
func (p *Processor) GroupCreate(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-group-create")
	defer span.End()

	logger := p.logger.With(zap.String("governor.group.id", payload.GroupID))

	if payload.GroupID == "" {
		logger.Error("bad event payload", zap.Error(ErrEventMissingGroupID))
		return ErrEventMissingGroupID
	}

	logger.Info("creating group")

	gid, err := p.reconciler.GroupCreate(ctx, payload.GroupID)
	if err != nil {
		logger.Error("error reconciling group creation", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	if err := p.reconciler.GroupsApplicationAssignments(ctx, payload.GroupID); err != nil {
		logger.Error("error reconciling group creation application assignment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	if err := p.reconciler.GroupMembership(ctx, payload.GroupID, gid); err != nil {
		logger.Error("error reconciling group creation membership", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully created group", zap.String("okta.group.id", gid))

	return nil
}

// GroupUpdate handles a governor group update event: it updates the group in
// okta and reconciles its application assignments.
func (p *Processor) GroupUpdate(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-group-update")
	defer span.End()

	logger := p.logger.With(zap.String("governor.group.id", payload.GroupID))

	if payload.GroupID == "" {
		logger.Error("bad event payload", zap.Error(ErrEventMissingGroupID))
		return ErrEventMissingGroupID
	}

	logger.Info("updating group")

	gid, err := p.reconciler.GroupUpdate(ctx, payload.GroupID)
	if err != nil {
		logger.Error("error reconciling group update", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	if err := p.reconciler.GroupsApplicationAssignments(ctx, payload.GroupID); err != nil {
		logger.Error("error reconciling group update application assignment", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully updated group", zap.String("okta.group.id", gid))

	return nil
}

// GroupDelete handles a governor group delete event: it deletes the group in okta.
func (p *Processor) GroupDelete(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-group-delete")
	defer span.End()

	logger := p.logger.With(zap.String("governor.group.id", payload.GroupID))

	if payload.GroupID == "" {
		logger.Error("bad event payload", zap.Error(ErrEventMissingGroupID))
		return ErrEventMissingGroupID
	}

	logger.Info("deleting group")

	gid, err := p.reconciler.GroupDelete(ctx, payload.GroupID)
	if err != nil {
		logger.Error("error deleting group", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully deleted group", zap.String("okta.group.id", gid))

	return nil
}

// MemberCreate handles a member being added to a governor group.
func (p *Processor) MemberCreate(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-member-create")
	defer span.End()

	logger := p.logger.With(zap.String("governor.group.id", payload.GroupID), zap.String("governor.user.id", payload.UserID))

	logger.Info("creating group membership")

	gid, uid, err := p.reconciler.GroupMembershipCreate(ctx, payload.GroupID, payload.UserID)
	if err != nil {
		logger.Error("error creating group membership", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully created group membership", zap.String("okta.group.id", gid), zap.String("okta.user.id", uid))

	return nil
}

// MemberDelete handles a member being removed from a governor group.
func (p *Processor) MemberDelete(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-member-delete")
	defer span.End()

	logger := p.logger.With(zap.String("governor.group.id", payload.GroupID), zap.String("governor.user.id", payload.UserID))

	logger.Info("deleting group membership")

	gid, uid, err := p.reconciler.GroupMembershipDelete(ctx, payload.GroupID, payload.UserID)
	if err != nil {
		logger.Error("error deleting group membership", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully deleted group membership", zap.String("okta.group.id", gid), zap.String("okta.user.id", uid))

	return nil
}

// UserUpdate handles a governor user update event.
func (p *Processor) UserUpdate(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-user-update")
	defer span.End()

	logger := p.logger.With(zap.String("governor.user.id", payload.UserID))

	if payload.UserID == "" {
		logger.Error("bad event payload", zap.Error(ErrEventMissingUserID))
		return ErrEventMissingUserID
	}

	logger.Info("updating user")

	uid, err := p.reconciler.UserUpdate(ctx, payload.UserID)
	if err != nil {
		logger.Error("error updating user", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully updated user", zap.String("okta.user.id", uid))

	return nil
}

// UserDelete handles a governor user delete event.
func (p *Processor) UserDelete(ctx context.Context, payload *v1alpha1.Event) error {
	ctx, span := p.tracer.Start(ctx, "process-user-delete")
	defer span.End()

	logger := p.logger.With(zap.String("governor.user.id", payload.UserID))

	if payload.UserID == "" {
		logger.Error("bad event payload", zap.Error(ErrEventMissingUserID))
		return ErrEventMissingUserID
	}

	logger.Info("deleting user")

	uid, err := p.reconciler.UserDelete(ctx, payload.UserID)
	if err != nil {
		logger.Error("error deleting user", zap.Error(err))
		span.SetStatus(codes.Error, err.Error())

		return err
	}

	logger.Info("successfully deleted user", zap.String("okta.user.id", uid))

	return nil
}

// auditMiddleware stashes a stub audit event in the context keyed off the
// event's AuditID and subject. The reconciler finalizes it (setting type and
// target) via auctx.WriteAuditEvent on the success path.
func (p *Processor) auditMiddleware(next eventrouter.Handler) eventrouter.Handler {
	return func(ctx context.Context, e *v1alpha1.Event) error {
		subject := eventrouter.GetSubjectFromContext(ctx)

		ctx = auctx.WithAuditEvent(
			ctx,
			auditevent.NewAuditEventWithID(
				e.AuditID,
				"", // eventType to be populated later
				auditevent.EventSource{
					Type:  "NATS",
					Value: subject,
					Extra: map[string]interface{}{
						"nats.subject": subject,
					},
				},
				auditevent.OutcomeSucceeded,
				map[string]string{
					"event": "governor",
				},
				"gov-okta-addon",
			),
		)

		return next(ctx, e)
	}
}
