package srv

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"go.equinixmetal.net/governor/pkg/api/v1alpha"
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

	switch payload.Action {
	case v1alpha.GovernorEventCreate:
		s.Logger.Info("creating group", zap.String("governor.group.id", payload.GroupID))

		out, err := s.Reconciler.ReconcileGroupExists(ctx, payload.GroupID)
		if err != nil {
			s.Logger.Error("error reconciling group creation", zap.Error(err))
			return
		}

		if err := s.Reconciler.ReconcileGroupsApplicationAssignments(ctx, payload.GroupID); err != nil {
			s.Logger.Error("error reconciling group creation application assignment", zap.Error(err))
			return
		}

		s.Logger.Info("successfully created group", zap.String("governor.group.id", payload.GroupID), zap.String("okta.group.id", out))
	case v1alpha.GovernorEventUpdate:
		s.Logger.Info("updating group", zap.String("governor.group.id", payload.GroupID))

		out, err := s.Reconciler.ReconcileGroupUpdate(context.Background(), payload.GroupID)
		if err != nil {
			s.Logger.Error("error reconciling group update", zap.Error(err))
			return
		}

		if err := s.Reconciler.ReconcileGroupsApplicationAssignments(ctx, payload.GroupID); err != nil {
			s.Logger.Error("error reconciling group creation application assignment", zap.Error(err))
			return
		}

		s.Logger.Info("successfully updated group", zap.String("governor.group.id", payload.GroupID), zap.String("okta.group.id", out))
	case v1alpha.GovernorEventDelete:
		s.Logger.Info("deleting group", zap.String("governor.group.id", payload.GroupID))

		out, err := s.Reconciler.ReconcileGroupDelete(context.TODO(), payload.GroupID)
		if err != nil {
			s.Logger.Error("error deleting group", zap.Error(err))
			return
		}

		s.Logger.Info("successfully deleted group", zap.String("governor.group.id", payload.GroupID), zap.String("okta.group.id", out))
	default:
		s.Logger.Warn("unexpected action in governor event", zap.String("governor.action", payload.Action))
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

	switch payload.Action {
	case v1alpha.GovernorEventCreate:
		s.Logger.Info("creating group membership", zap.String("governor.id", payload.GroupID), zap.String("governor.id", payload.UserID))

		// TODO validate the user is a member of the group from governor API (requires https://packet.atlassian.net/browse/DEL-1236)?
		gid, err := s.OktaClient.GetGroupByGovernorID(context.Background(), payload.GroupID)
		if err != nil {
			s.Logger.Error("error getting group by governor id", zap.String("governor.id", payload.GroupID), zap.Error(err))
			return
		}

		// TODO get the email address of the user from governor API (requires https://packet.atlassian.net/browse/DEL-1236)
		uid, err := s.OktaClient.GetUserIDByEmail(context.Background(), "test@example.com")
		if err != nil {
			s.Logger.Error("error getting user by email address", zap.String("user.email", "test@example.com"), zap.Error(err))
			return
		}

		if err := s.OktaClient.AddGroupUser(context.Background(), gid, uid); err != nil {
			s.Logger.Error("failed to add user to group", zap.String("user.email", "test@example.com"), zap.String("okta.user.id", uid), zap.String("okta.group.id", gid), zap.Error(err))
			return
		}

	case v1alpha.GovernorEventDelete:
		s.Logger.Info("deleting group", zap.String("governor.id", payload.GroupID), zap.String("governor.id", payload.UserID))

		// TODO validate the user is not a member of the group from governor API (requires https://packet.atlassian.net/browse/DEL-1236)?
		gid, err := s.OktaClient.GetGroupByGovernorID(context.Background(), payload.GroupID)
		if err != nil {
			s.Logger.Error("error getting group by governor id", zap.String("governor.id", payload.GroupID), zap.Error(err))
			return
		}

		// TODO get the email address of the user from governor API (requires https://packet.atlassian.net/browse/DEL-1236)
		uid, err := s.OktaClient.GetUserIDByEmail(context.Background(), "test@example.com")
		if err != nil {
			s.Logger.Error("error getting user by email address", zap.String("user.email", "test@example.com"), zap.Error(err))
			return
		}

		if err := s.OktaClient.RemoveGroupUser(context.Background(), gid, uid); err != nil {
			s.Logger.Error("failed to remove user from group", zap.String("user.email", "test@example.com"), zap.String("okta.user.id", uid), zap.String("okta.group.id", gid), zap.Error(err))
			return
		}
	default:
		s.Logger.Warn("unexpected action in governor event", zap.String("governor.action", payload.Action))
		return
	}
}

// usersMessageHandler handles messages for governor user events
func (s *Server) usersMessageHandler(m *nats.Msg) {
	_, err := s.unmarshalPayload(m)
	if err != nil {
		s.Logger.Warn("unable to unmarshal governor payload", zap.Error(err))
		return
	}
}

func (s *Server) unmarshalPayload(m *nats.Msg) (*v1alpha.Event, error) {
	s.Logger.Debug("received a message:", zap.String("nats.data", string(m.Data)), zap.String("nats.subject", m.Subject))

	payload := v1alpha.Event{}
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}
