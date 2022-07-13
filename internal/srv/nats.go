package srv

import (
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// NATSClient is a NATS client with some configuration
type NATSClient struct {
	conn       *nats.Conn
	logger     *zap.Logger
	prefix     string
	queueGroup string
}

// NATSOption is a functional configuration option for NATS
type NATSOption func(c *NATSClient)

// NewNATSClient configures and establishes a new NATS client connection
func NewNATSClient(opts ...NATSOption) (*NATSClient, error) {
	client := NATSClient{
		logger: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(&client)
	}

	return &client, nil
}

// WithNATSConn sets the nats connection
func WithNATSConn(nc *nats.Conn) NATSOption {
	return func(c *NATSClient) {
		c.conn = nc
	}
}

// WithNATSPrefix sets the nats subscription prefix
func WithNATSPrefix(p string) NATSOption {
	return func(c *NATSClient) {
		c.prefix = p
	}
}

// WithNATSQueueGroup sets the nats subscription queue group
func WithNATSQueueGroup(q string) NATSOption {
	return func(c *NATSClient) {
		c.queueGroup = q
	}
}

// WithNATSLogger sets the NATS client logger
func WithNATSLogger(l *zap.Logger) NATSOption {
	return func(c *NATSClient) {
		c.logger = l
	}
}

func (s *Server) registerSubscriptionHandlers() error {
	prefix := s.NATSClient.prefix
	qg := s.NATSClient.queueGroup

	// Receive groups channel events
	if _, err := s.NATSClient.conn.QueueSubscribe(prefix+".groups", qg, s.groupsMessageHandler); err != nil {
		return err
	}

	// Receive group memberships channel events
	if _, err := s.NATSClient.conn.QueueSubscribe(prefix+".members", qg, s.membersMessageHandler); err != nil {
		return err
	}

	// Receive users channel events
	if _, err := s.NATSClient.conn.QueueSubscribe(prefix+".users", qg, s.usersMessageHandler); err != nil {
		return err
	}

	return nil
}

func (s *Server) shutdownSubscriptions() error {
	// Drain and close the NATS connection
	return s.NATSClient.conn.Drain()
}
