package reconciler

import "errors"

var (
	// ErrGroupMembershipNotFound is returned when a group membership action
	// is requested and the user is not found in the group
	ErrGroupMembershipNotFound = errors.New("user not found in group")
	// ErrGroupMembershipFound is returned when a group membership delete request finds the
	// user in the governor group
	ErrGroupMembershipFound = errors.New("delete request user found in group")
	// ErrGovernorUserPendingStatus is returned when an event it received for a user with pending status
	ErrGovernorUserPendingStatus = errors.New("governor user has pending status")
	// ErrUserStillExists is returned when a user delete request finds the user still exists in governor
	ErrUserStillExists = errors.New("delete request user still exists")
	// ErrUserExternalIDMissing is returned when an action is requested that requires the external id, but its missing
	ErrUserExternalIDMissing = errors.New("user external id is missing")
	// ErrUserListEmpty is returned when a user reconcile gets an empty user list from governor or okta
	ErrUserListEmpty = errors.New("reconcile got an empty user list")
)
