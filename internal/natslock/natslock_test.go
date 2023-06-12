package natslock

import (
	"reflect"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

var jetstream nats.JetStreamContext

func TestMain(m *testing.M) {
	natsSrv, err := natsserver.NewServer(&natsserver.Options{
		Host:      "127.0.0.1",
		Port:      natsserver.RANDOM_PORT,
		Debug:     false,
		JetStream: true,
	})
	if err != nil {
		panic(err)
	}

	defer natsSrv.Shutdown()

	if err := natsserver.Run(natsSrv); err != nil {
		panic(err)
	}

	nc, err := nats.Connect(natsSrv.ClientURL())
	if err != nil {
		panic(err)
	}

	jetstream, err = nc.JetStream()
	if err != nil {
		panic(err)
	}

	m.Run()
}

func TestNew(t *testing.T) {
	locker := New()

	lockerType := reflect.TypeOf(locker).String()
	if lockerType != "*natslock.Locker" {
		t.Errorf("expected type to be '*natslock.Locker', got %s", lockerType)
	}

	locker = New(WithLogger(zap.NewExample()))
	if locker.Logger.Core().Enabled(zap.DebugLevel) != true {
		t.Error("expected logger debug level to be 'true', got 'false'")
	}

	kvStore, err := NewKeyValue(jetstream, "test-bucket-1", time.Minute)
	if err != nil {
		panic(err)
	}

	locker = New(WithKeyValueStore(kvStore))
	if locker.KVStore == nil {
		t.Error("expected locker KVStore to be initialized")
	}

	if locker.KVStore.Bucket() != "test-bucket-1" {
		t.Errorf("expected locker KVStore bucket to be 'test-bucket-1', got %s", locker.KVStore.Bucket())
	}
}

func TestNewKeyValue(t *testing.T) {
	type args struct {
		name string
		ttl  time.Duration
	}

	tests := []struct {
		name    string
		args    args
		want    nats.KeyValue
		wantErr bool
	}{
		{
			name: "missing name",
			args: args{
				ttl: time.Minute,
			},
			wantErr: true,
		},
		{
			name: "missing ttl",
			args: args{
				name: "test-bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewKeyValue(jetstream, tt.args.name, tt.args.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewKeyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// TODO: test success cases
		})
	}
}
