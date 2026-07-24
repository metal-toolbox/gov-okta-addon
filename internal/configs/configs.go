// Package configs contains configuration structures and utilities for the application.
package configs

import (
	"context"
	"net/url"
	"time"

	"github.com/metal-toolbox/gov-okta-addon/internal/okta"
	govclient "github.com/metal-toolbox/governor-api/pkg/client"
	govcfg "github.com/metal-toolbox/governor-api/pkg/configs"
	sdkcfg "github.com/metal-toolbox/governor-extension-sdk/pkg/configs"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// DefaultReconcilerInterval is the default interval for the reconciler loop
	DefaultReconcilerInterval = 1 * time.Hour
	// DefaultEventlogInterval is the default interval for the okta eventlog poller
	DefaultEventlogInterval = 30 * time.Second
	// DefaultEventlogLookback is the coldstart lookback period for the okta eventlog poller
	DefaultEventlogLookback = 8 * time.Hour
)

// AppConfig holds the application configuration
var AppConfig struct {
	govcfg.Configs `mapstructure:",squash"` // brings WorkloadIdentity, NATSConn, Validate, ...

	Audit    sdkcfg.Audit
	Tracing  sdkcfg.Tracing
	Logging  sdkcfg.Logging
	Governor sdkcfg.Governor
	Server   sdkcfg.Server
	NATS     sdkcfg.NATSConfig

	DryRun     bool `mapstructure:"dryrun"`
	SkipDelete bool `mapstructure:"skip-delete"`
	Okta       Okta
	Reconciler Reconciler
	Eventlog   Eventlog
	Sync       Sync
}

// Okta holds Okta API configuration
type Okta struct {
	URL     string `mapstructure:"url"`
	Token   string `mapstructure:"token"`
	NoCache bool   `mapstructure:"nocache"`
}

// Reconciler holds reconciler configuration
type Reconciler struct {
	Interval time.Duration `mapstructure:"interval"`
	Locking  bool          `mapstructure:"locking"`
}

// Eventlog holds okta eventlog poller configuration
type Eventlog struct {
	Interval time.Duration `mapstructure:"interval"`
	Lookback time.Duration `mapstructure:"lookback"`
}

// Sync holds configuration for the one-shot okta->governor sync commands. The
// dry-run toggle is shared with serve via the top-level "dryrun" key.
type Sync struct {
	SelectorPrefix string   `mapstructure:"selector-prefix"`
	SkipGroups     []string `mapstructure:"skip-groups"`
	SkipOktaUpdate bool     `mapstructure:"skip-okta-update"`
}

// MustOktaFlags registers Okta related flags and binds them to viper
// Panics on error
func MustOktaFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("okta-url", "https://example.okta.com", "url for Okta client calls")
	viperBindFlag(v, "okta.url", flags.Lookup("okta-url"))
	flags.String("okta-token", "", "token for access to the Okta API")
	viperBindFlag(v, "okta.token", flags.Lookup("okta-token"))
	flags.Bool("okta-nocache", false, "disable the okta client cache, useful for development")
	viperBindFlag(v, "okta.nocache", flags.Lookup("okta-nocache"))
}

// MustSyncGroupsFlags registers flags specific to the group sync command and
// binds them to viper. Panics on error.
func MustSyncGroupsFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.Bool("skip-okta-update", false, "do not make changes to okta groups (ie. setting the governor_id)")
	viperBindFlag(v, "sync.skip-okta-update", flags.Lookup("skip-okta-update"))
	flags.String("selector-prefix", "", "if set, only group names that start with this string will be processed")
	viperBindFlag(v, "sync.selector-prefix", flags.Lookup("selector-prefix"))
	flags.StringSlice("skip-groups", []string{"Everyone", "catchall"}, "groups to skip during the sync")
	viperBindFlag(v, "sync.skip-groups", flags.Lookup("skip-groups"))
}

// MustReconcilerFlags registers Reconciler related flags and binds them to viper
// Panics on error
func MustReconcilerFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.Duration("reconciler-interval", DefaultReconcilerInterval, "interval for the reconciler loop")
	viperBindFlag(v, "reconciler.interval", flags.Lookup("reconciler-interval"))
	flags.Bool("reconciler-locking", false, "enable reconciler locking and leader election")
	viperBindFlag(v, "reconciler.locking", flags.Lookup("reconciler-locking"))
	flags.Bool("skip-delete", true, "do not delete anything in okta during reconcile loop")
	viperBindFlag(v, "skip-delete", flags.Lookup("skip-delete"))
	flags.Duration("eventlog-interval", DefaultEventlogInterval, "run interval for the okta eventlog poller")
	viperBindFlag(v, "eventlog.interval", flags.Lookup("eventlog-interval"))
	flags.Duration("eventlog-lookback", DefaultEventlogLookback, "coldstart lookback time period for the okta eventlog poller")
	viperBindFlag(v, "eventlog.lookback", flags.Lookup("eventlog-lookback"))
}

// viperBindFlag provides a wrapper around the viper bindings that handles error checks
func viperBindFlag(v *viper.Viper, name string, flag *pflag.Flag) {
	if err := v.BindPFlag(name, flag); err != nil {
		panic(err)
	}
}

// NewOktaClient returns an okta API client configured from the app config.
func NewOktaClient(logger *zap.Logger) (*okta.Client, error) {
	return okta.NewClient(
		okta.WithLogger(logger),
		okta.WithURL(AppConfig.Okta.URL),
		okta.WithToken(AppConfig.Okta.Token),
		okta.WithCache(!AppConfig.Okta.NoCache),
	)
}

// NewGovernorClient returns a governor API client based on auth configs. When
// workload identity federation is enabled it uses a WIF token source, otherwise
// it falls back to the client credentials flow.
func NewGovernorClient(ctx context.Context, opts ...govclient.Option) (*govclient.Client, error) {
	var (
		ts  oauth2.TokenSource
		err error
	)

	if AppConfig.Governor.WorkloadIdentity {
		ts, err = AppConfig.WorkloadIdentity.ToTokenSource(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		cc := &clientcredentials.Config{
			ClientID:       AppConfig.Governor.ClientID,
			ClientSecret:   AppConfig.Governor.ClientSecret,
			TokenURL:       AppConfig.Governor.TokenURL,
			EndpointParams: url.Values{"audience": {AppConfig.Governor.Audience}},
			Scopes:         AppConfig.Governor.Scopes,
		}

		ts = cc.TokenSource(ctx)
	}

	opts = append(
		opts,
		govclient.WithURL(AppConfig.Governor.URL),
		govclient.WithTokenSource(ts),
	)

	return govclient.NewClient(opts...)
}
