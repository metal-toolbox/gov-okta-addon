package natslock

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// Locker is a distributed lock backed by a JetStream key-value store
type Locker struct {
	KVStore nats.KeyValue
	KVKey   string
	Logger  *zap.Logger
}

// Option is a functional configuration option
type Option func(l *Locker)

// WithKeyValueStore sets the nats key value store
func WithKeyValueStore(kv nats.KeyValue) Option {
	return func(l *Locker) {
		l.KVStore = kv
	}
}

// WithLogger sets logger
func WithLogger(log *zap.Logger) Option {
	return func(l *Locker) {
		l.Logger = log
	}
}

// New returns a new locker
func New(opts ...Option) *Locker {
	lock := Locker{
		KVKey:  "leader",
		Logger: zap.NewNop(),
	}

	for _, opt := range opts {
		opt(&lock)
	}

	return &lock
}

// NewKeyValue returns a JetStream key-value store with the given name. If the
// bucket does not exist, it will be created with the given TTL.
func NewKeyValue(jets nats.JetStreamContext, name string, ttl time.Duration) (nats.KeyValue, error) {
	if name == "" || ttl == 0 {
		return nil, ErrBadParameter
	}

	jkv, err := jets.KeyValue(name)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}

		// create jetstream key-value bucket
		jkv, err = jets.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: name,
			TTL:    ttl,
		})
		if err != nil {
			return nil, err
		}
	}

	return jkv, nil
}

// AcquireLead attempts to acquire the leader lock for the given id and returns true if successful.
// If the lock is already held by another id, it will return false.
func (l *Locker) AcquireLead(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, ErrBadParameter
	}

	entry, err := l.KVStore.Get(l.KVKey)

	switch {
	case err == nil:
		l.Logger.Debug("got key value", zap.String("key", l.KVKey), zap.ByteString("value", entry.Value()))

		uuidVal, err := uuid.FromString(string(entry.Value()))
		if err != nil {
			// we expect to find a uuid value in the lock but it's something else
			l.Logger.Warn("unable to parse uuid lock value, will try to update the lock", zap.Error(err))

			// this isn't supposed to happen, so let's try to update the lock and take the lead
			_, err := l.KVStore.PutString(l.KVKey, id.String())
			if err != nil {
				l.Logger.Error("error updating lock", zap.Error(err))
				return false, err
			}

			return true, nil
		}

		if uuidVal != id {
			l.Logger.Info("existing lock found (someone else is the leader)", zap.String("id", id.String()), zap.String("value", uuidVal.String()))
			return false, nil
		}

		l.Logger.Info("existing lock found (i am the leader)", zap.String("id", id.String()), zap.String("value", uuidVal.String()))

		// update the lock so the ttl doesn't expire
		_, err = l.KVStore.PutString(l.KVKey, id.String())
		if err != nil {
			l.Logger.Warn("unable to update lock", zap.String("id", id.String()), zap.Error(err))
		}

		return true, nil

	case errors.Is(err, nats.ErrKeyNotFound):
		// create the lock and make this id the leader
		_, err := l.KVStore.PutString(l.KVKey, id.String())
		if err != nil {
			// log warning and proceed (should be safe as there's no existing lock)
			l.Logger.Warn("unable to create leader lock, still proceeding as lead", zap.String("id", id.String()), zap.Error(err))
			return true, nil
		}

		l.Logger.Info("obtained leader lock", zap.String("id", id.String()))

		return true, nil

	default:
		l.Logger.Error("error getting lock key from kv store", zap.Error(err))
		return false, err
	}
}

// ReleaseLead releases the leader lock if the given id belongs to the current leader
func (l *Locker) ReleaseLead(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrBadParameter
	}

	entry, err := l.KVStore.Get(l.KVKey)
	if err != nil {
		return nil
	}

	l.Logger.Debug("got key value", zap.String("key", l.KVKey), zap.ByteString("value", entry.Value()))

	uuidVal, err := uuid.FromString(string(entry.Value()))
	if err != nil {
		return nil
	}

	if uuidVal != id {
		return nil
	}

	return l.KVStore.Purge(l.KVKey)
}

// Name returns the name of the locker kv store
func (l *Locker) Name() string {
	return l.KVStore.Bucket()
}

// TTL returns the ttl of the locker kv store
func (l *Locker) TTL() time.Duration {
	kvStatus, err := l.KVStore.Status()
	if err != nil {
		l.Logger.Error(err.Error())
		return 0
	}

	return kvStatus.TTL()
}
