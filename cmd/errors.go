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
)
