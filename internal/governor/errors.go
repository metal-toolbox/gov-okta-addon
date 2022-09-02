package governor

import "errors"

var (
	// ErrRequestNonSuccess is returned when a call to the governor API returns a non-success status
	ErrRequestNonSuccess = errors.New("got a non-success response from governor")

	// ErrMissingGroupID is returned when a missing or bad group id is passed to a request
	ErrMissingGroupID = errors.New("missing group id in request")

	// ErrMissingOrganizationID is returned when a missing or bad organization id is passed to a request
	ErrMissingOrganizationID = errors.New("missing organization id in request")

	// ErrMissingUserID is returned when a missing or bad user id is passed to a request
	ErrMissingUserID = errors.New("missing user id in request")

	// ErrNilUserRequest is returned when a nil user body is passed to a request
	ErrNilUserRequest = errors.New("nil user request")
)
