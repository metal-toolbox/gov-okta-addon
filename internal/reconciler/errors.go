package reconciler

import "errors"

var (
	// ErrGroupMembershipNotFound is returned when a group membership action
	// is requested and the user is not found in the group
	ErrGroupMembershipNotFound = errors.New("user not found in group")
	// ErrGroupMembershipFound is returned when a group membership delete request finds the
	// user in the governor group
	ErrGroupMembershipFound = errors.New("delete request user found in group")
	// ErrUserStillExists is returned when a user delete request finds the user still exists in governor
	ErrUserStillExists = errors.New("delete request user still exists")
	// ErrUserListEmpty is returned when a user reconcile gets an empty user list from governor or okta
	ErrUserListEmpty = errors.New("reconcile got an empty user list")
)
