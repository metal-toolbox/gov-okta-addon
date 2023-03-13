package cmd

import "github.com/nats-io/nats.go"

// NewNATSConnection creates a new NATS connection
func NewNATSConnection(appName, credsFile, url string) (*nats.Conn, func(), error) {
	opts := []nats.Option{
		nats.Name(appName),
	}

	if credsFile != "" {
		opts = append(opts, nats.UserCredentials(credsFile))
	} else {
		return nil, nil, ErrMissingNATSCreds
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, nil, err
	}

	return nc, nc.Close, nil
}
