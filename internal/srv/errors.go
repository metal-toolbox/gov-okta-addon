package srv

import "errors"

var (
	// ErrEventMissingGroupID is returned when a group event is missing the group ID
	ErrEventMissingGroupID = errors.New("event missing group ID")
	// ErrEventMissingUserID is returned when a user event is missing the user ID
	ErrEventMissingUserID = errors.New("event missing user ID")
)
