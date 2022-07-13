package srv

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

const (
	// GovernorEventCreate is the action passed on create events
	// TODO: this definition should move into governor itself and be imported.
	GovernorEventCreate = "CREATE"
	// GovernorEventUpdate is the action passed on update events
	// TODO: this definition should move into governor itself and be imported.
	GovernorEventUpdate = "UPDATE"
	// GovernorEventDelete is the action passed on delete events
	// TODO: this definition should move into governor itself and be imported.
	GovernorEventDelete = "DELETE"
)

// GovernorEvent is an event notification from Governor.
// TODO: this definition should move into governor itself and be imported.
type GovernorEvent struct {
	Version string `json:"version"`
	Action  string `json:"action"`
	GroupID string `json:"group_id,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

// groupsMessageHandler handles messages for governor group events
func (s *Server) groupsMessageHandler(m *nats.Msg) {
	payload, err := s.unmarshalPayload(m)
	if err != nil {
		s.Logger.Warn("unable to unmarshal governor payload", zap.Error(err))
		return
	}

	switch payload.Action {
	case GovernorEventCreate:
		s.Logger.Info("creating group", zap.String("governor.id", payload.GroupID))

		// TODO get the group from governor API by ID (requires https://packet.atlassian.net/browse/DEL-1236)
		out, err := s.OktaClient.CreateGroup(context.Background(), "testgroup01", "test group", map[string]interface{}{"governor_id": "12345"})
		if err != nil {
			s.Logger.Error("error creating group", zap.Error(err))
			return
		}

		s.Logger.Info("successfully created group", zap.String("governor.id", payload.GroupID), zap.String("okta.group.id", out))
	case GovernorEventUpdate:
		s.Logger.Info("updating group", zap.String("governor.id", payload.GroupID))

		gid, err := s.OktaClient.GetGroupByGovernorID(context.Background(), payload.GroupID)
		if err != nil {
			s.Logger.Error("error getting group by governor id", zap.String("governor.id", payload.GroupID), zap.Error(err))
			return
		}

		// TODO get the group from governor API by ID (requires https://packet.atlassian.net/browse/DEL-1236)
		if err := s.OktaClient.UpdateGroup(context.Background(), gid, "testgroup01", "test group improved", map[string]interface{}{"governor_id": payload.GroupID}); err != nil {
			s.Logger.Error("error updating group", zap.Error(err))
			return
		}

		s.Logger.Info("successfully updated group", zap.String("governor.id", payload.GroupID), zap.String("okta.group.id", gid))
	case GovernorEventDelete:
		s.Logger.Info("deleting group", zap.String("governor.id", payload.GroupID))

		// TODO validate the group is deleted from governor API by ID (requires https://packet.atlassian.net/browse/DEL-1236)?

		gid, err := s.OktaClient.GetGroupByGovernorID(context.Background(), payload.GroupID)
		if err != nil {
			s.Logger.Error("error getting group by governor id", zap.String("governor.id", payload.GroupID), zap.Error(err))
			return
		}

		if err := s.OktaClient.DeleteGroup(context.Background(), gid); err != nil {
			s.Logger.Error("error deleting group", zap.Error(err))
			return
		}

		s.Logger.Info("successfully deleted group", zap.String("governor.id", payload.GroupID), zap.String("okta.group.id", gid))
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
	case GovernorEventCreate:
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

	case GovernorEventDelete:
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

func (s *Server) unmarshalPayload(m *nats.Msg) (*GovernorEvent, error) {
	s.Logger.Debug("received a message:", zap.String("nats.data", string(m.Data)), zap.String("nats.subject", m.Subject))

	payload := GovernorEvent{}
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}
