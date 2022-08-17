package okta

import "errors"

var (
	// ErrGroupsNotFound is returned when a group is not found in Okta
	ErrGroupsNotFound = errors.New("group(s) not found")
	// ErrUnexpectedGroupsCount is returned when we get an unexpected number of groups, usually != 1
	ErrUnexpectedGroupsCount = errors.New("unexpected number of groups returned")
	// ErrUnexpectedUsersCount is returned when we get an unexpected number of users, usually != 1
	ErrUnexpectedUsersCount = errors.New("unexpected number of users returned")
)
