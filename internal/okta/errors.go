package okta

import "errors"

var (
	// ErrUnexpectedGroupsCount is returned when we get an unexpected number of groups, usually != 1
	ErrUnexpectedGroupsCount = errors.New("unexpected number of groups returned")
	// ErrUnexpectedUsersCount is returned when we get an unexpected number of users, usually != 1
	ErrUnexpectedUsersCount = errors.New("unexpected number of users returned")
)
