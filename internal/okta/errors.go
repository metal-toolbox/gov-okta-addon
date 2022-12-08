package okta

import "errors"

var (
	// ErrBadOktaGroupParameter is returned when a bad or unexpected okta group is passed to a function
	ErrBadOktaGroupParameter = errors.New("bad group parameter")
	// ErrGroupGovernorIDNotFound is returned when the governor id is not found on a group profile
	ErrGroupGovernorIDNotFound = errors.New("governor id not found on group profile")
	// ErrGroupGovernorIDNotString is returned if the Governor ID on a group is not a string
	ErrGroupGovernorIDNotString = errors.New("governor id on group profile is not a string")
	// ErrNilGroupProfile is returned when the profile on an okta group is nil
	ErrNilGroupProfile = errors.New("okta group profile is nil")
	// ErrGroupsNotFound is returned when a group is not found in Okta
	ErrGroupsNotFound = errors.New("group(s) not found")
	// ErrUnexpectedGroupsCount is returned when we get an unexpected number of groups, usually != 1
	ErrUnexpectedGroupsCount = errors.New("unexpected number of groups returned")
	// ErrUnexpectedUsersCount is returned when we get an unexpected number of users, usually != 1
	ErrUnexpectedUsersCount = errors.New("unexpected number of users returned")
	// ErrApplicationBadParameters is returned when bad parameters are not passed to an app request
	ErrApplicationBadParameters = errors.New("application request bad parameters")

	// ErrOktaUserExternalIDNotString is returned when the okta user profile contains an external id that's not a string
	ErrOktaUserExternalIDNotString = errors.New("okta user external id in profile is not a string")
	// ErrOktaUserEmailNotString is returned when the okta user profile contains an email that's not a string
	ErrOktaUserEmailNotString = errors.New("okta user email in profile is not a string")
	// ErrOktaUserFirstNameNotString is returned when the okta user profile contains a first name that's not a string
	ErrOktaUserFirstNameNotString = errors.New("okta user first name in profile is not a string")
	// ErrOktaUserIDEmpty is returned when the okta user has an empty id
	ErrOktaUserIDEmpty = errors.New("okta user id empty")
	// ErrOktaUserLastNameNotString is returned when the okta user profile contains a last name that's not a string
	ErrOktaUserLastNameNotString = errors.New("okta user last name in profile is not a string")
	// ErrOktaUserTypeNotString is returned when the okta user profile contains a user type that's not a string
	ErrOktaUserTypeNotString = errors.New("okta user type in profile is not a string")
)
