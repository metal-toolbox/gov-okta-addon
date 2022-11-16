// Package auctx helps to store and retrieve audit events in the context
package auctx

import (
	"context"
	"fmt"

	"github.com/metal-toolbox/auditevent"
)

type auditEventKeyType string

const auditEventKey auditEventKeyType = "auditevent"

// ErrAuditEventKeyNotFound is returned when auditEventKey is not found in the context
var ErrAuditEventKeyNotFound = fmt.Errorf("%s key not found in context", auditEventKey)

// WithAuditEvent adds an audit event to the context
func WithAuditEvent(ctx context.Context, auevent *auditevent.AuditEvent) context.Context {
	return context.WithValue(ctx, auditEventKey, auevent)
}

// GetAuditEvent gets an audit event from the context
func GetAuditEvent(ctx context.Context) *auditevent.AuditEvent {
	auEvent, ok := ctx.Value(auditEventKey).(*auditevent.AuditEvent)
	if !ok {
		// audit event not found in context
		return nil
	}

	return auEvent
}

// WriteAuditEvent assembles a complete audit event and writes it to the event writer
func WriteAuditEvent(ctx context.Context, evWriter *auditevent.EventWriter, evType string, evTarget map[string]string) error {
	ae := GetAuditEvent(ctx)
	if ae == nil {
		return ErrAuditEventKeyNotFound
	}

	ae.Type = evType

	return evWriter.Write(ae.WithTarget(evTarget))
}
