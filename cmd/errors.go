package cmd

import "errors"

var (
	// ErrNATSURLRequired is returned when a NATS url is missing
	ErrNATSURLRequired = errors.New("nats url is required and cannot be empty")
	// ErrNATSAuthRequired is returned when a NATS auth method is missing
	ErrNATSAuthRequired = errors.New("nats token or nkey auth is required and cannot be empty")
	// ErrOktaURLRequired is returned when an Okta URL is missing
	ErrOktaURLRequired = errors.New("okta url is required and cannot be empty")
	// ErrOktaTokenRequired is returned when an Okta token is missing
	ErrOktaTokenRequired = errors.New("okta token is required and cannot be empty")
	// ErrGovernorURLRequired is returned when a governor URL is missing
	ErrGovernorURLRequired = errors.New("governor url is required and cannot be empty")
	// ErrGovernorClientIDRequired is returned when a governor client id is missing
	ErrGovernorClientIDRequired = errors.New("governor oauth client id is required and cannot be empty")
	// ErrGovernorClientSecretRequired is returned when a governor client secret is missing
	ErrGovernorClientSecretRequired = errors.New("governor oauth client secret is required and cannot be empty")
	// ErrGovernorClientTokenURLRequired is returned when a governor token url is missing
	ErrGovernorClientTokenURLRequired = errors.New("governor oauth client token url is required and cannot be empty")
	// ErrGovernorClientAudienceRequired is returned when a governor client audience is missing
	ErrGovernorClientAudienceRequired = errors.New("governor oauth client audience is required and cannot be empty")
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
