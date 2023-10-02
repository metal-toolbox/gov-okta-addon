package srv

import (
	"context"
	"encoding/json"

	"github.com/metal-toolbox/auditevent"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/equinixmetal/gov-okta-addon/internal/auctx"
	"github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
)

// groupsMessageHandler handles messages for governor group events
func (s *Server) groupsMessageHandler(m *nats.Msg) {
	payload, err := s.unmarshalPayload(m)
	if err != nil {
		s.Logger.Warn("unable to unmarshal governor payload", zap.Error(err))
		return
	}

	if payload.GroupID == "" {
		s.Logger.Error("bad event payload", zap.Error(ErrEventMissingGroupID))
		return
	}

	ctx := context.Background()

	logger := s.Logger.With(zap.String("governor.group.id", payload.GroupID))

	switch payload.Action {
	case v1alpha1.GovernorEventCreate:
		logger.Info("creating group")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		gid, err := s.Reconciler.GroupCreate(ctx, payload.GroupID)
		if err != nil {
			logger.Error("error reconciling group creation", zap.Error(err))
			return
		}

		if err := s.Reconciler.GroupsApplicationAssignments(ctx, payload.GroupID); err != nil {
			logger.Error("error reconciling group creation application assignment", zap.Error(err))
			return
		}

		if err := s.Reconciler.GroupMembership(ctx, payload.GroupID, gid); err != nil {
			logger.Error("error reconciling group creation membership", zap.Error(err))
			return
		}

		logger.Info("successfully created group", zap.String("okta.group.id", gid))

	case v1alpha1.GovernorEventUpdate:
		logger.Info("updating group")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		gid, err := s.Reconciler.GroupUpdate(ctx, payload.GroupID)
		if err != nil {
			logger.Error("error reconciling group update", zap.Error(err))
			return
		}

		if err := s.Reconciler.GroupsApplicationAssignments(ctx, payload.GroupID); err != nil {
			logger.Error("error reconciling group creation application assignment", zap.Error(err))
			return
		}

		logger.Info("successfully updated group", zap.String("okta.group.id", gid))

	case v1alpha1.GovernorEventDelete:
		logger.Info("deleting group")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		gid, err := s.Reconciler.GroupDelete(ctx, payload.GroupID)
		if err != nil {
			logger.Error("error deleting group", zap.Error(err))
			return
		}

		logger.Info("successfully deleted group", zap.String("okta.group.id", gid))

	default:
		logger.Warn("unexpected action in governor event", zap.String("governor.action", payload.Action))
		return
	}
}

// membersMessageHandler handles messages for governor membership events
func (s *Server) membersMessageHandler(m *nats.Msg) {
	payload, err := s.unmarshalPayload(m)
	if err != nil {
		s.Logger.Warn("unable to unmarshal governor payload", zap.Error(err))
		return
	}

	ctx := context.Background()

	logger := s.Logger.With(zap.String("governor.group.id", payload.GroupID), zap.String("governor.user.id", payload.UserID))

	switch payload.Action {
	case v1alpha1.GovernorEventCreate:
		logger.Info("creating group membership")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		gid, uid, err := s.Reconciler.GroupMembershipCreate(ctx, payload.GroupID, payload.UserID)
		if err != nil {
			logger.Error("error creating group membership", zap.Error(err))
			return
		}

		logger.Info("successfully created group membership", zap.String("okta.group.id", gid), zap.String("okta.user.id", uid))

	case v1alpha1.GovernorEventDelete:
		logger.Info("deleting group membership")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		gid, uid, err := s.Reconciler.GroupMembershipDelete(ctx, payload.GroupID, payload.UserID)
		if err != nil {
			logger.Error("error deleting group membership", zap.Error(err))
			return
		}

		logger.Info("successfully deleted group membership", zap.String("okta.group.id", gid), zap.String("okta.user.id", uid))

	default:
		logger.Warn("unexpected action in governor event", zap.String("governor.action", payload.Action))
		return
	}
}

// usersMessageHandler handles messages for governor user events
func (s *Server) usersMessageHandler(m *nats.Msg) {
	payload, err := s.unmarshalPayload(m)
	if err != nil {
		s.Logger.Warn("unable to unmarshal governor payload", zap.Error(err))
		return
	}

	if payload.UserID == "" {
		s.Logger.Error("bad event payload", zap.Error(ErrEventMissingUserID))
		return
	}

	ctx := context.Background()

	logger := s.Logger.With(zap.String("governor.user.id", payload.UserID))

	switch payload.Action {
	case v1alpha1.GovernorEventDelete:
		logger.Info("deleting user")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		uid, err := s.Reconciler.UserDelete(ctx, payload.UserID)
		if err != nil {
			logger.Error("error deleting user", zap.Error(err))
			return
		}

		logger.Info("successfully deleted user", zap.String("okta.user.id", uid))

	case v1alpha1.GovernorEventUpdate:
		logger.Info("updating user")

		ctx = auctx.WithAuditEvent(ctx, s.auditEventNATS(m.Subject, payload))

		uid, err := s.Reconciler.UserUpdate(ctx, payload.UserID)
		if err != nil {
			logger.Error("error updating user", zap.Error(err))
			return
		}

		logger.Info("successfully updated user", zap.String("okta.user.id", uid))

	default:
		logger.Warn("unexpected action in governor event", zap.String("governor.action", payload.Action))
		return
	}
}

func (s *Server) unmarshalPayload(m *nats.Msg) (*v1alpha1.Event, error) {
	s.Logger.Debug("received a message:", zap.String("nats.data", string(m.Data)), zap.String("nats.subject", m.Subject))

	payload := v1alpha1.Event{}
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

// auditEventNATS returns a stub NATS audit event
func (s *Server) auditEventNATS(natsSubj string, event *v1alpha1.Event) *auditevent.AuditEvent {
	return auditevent.NewAuditEventWithID(
		event.AuditID,
		"", // eventType to be populated later
		auditevent.EventSource{
			Type:  "NATS",
			Value: s.NATSClient.conn.ConnectedUrlRedacted(),
			Extra: map[string]interface{}{
				"nats.subject":    natsSubj,
				"nats.queuegroup": s.NATSClient.queueGroup,
			},
		},
		auditevent.OutcomeSucceeded,
		map[string]string{
			"event": "governor",
		},
		"gov-okta-addon",
	)
}
