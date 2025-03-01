package cmd

import (
	"context"
	"errors"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/metal-toolbox/addonx/natslock"
	"github.com/metal-toolbox/auditevent"
	audithelpers "github.com/metal-toolbox/auditevent/helpers"
	"github.com/metal-toolbox/gov-okta-addon/internal/okta"
	"github.com/metal-toolbox/gov-okta-addon/internal/reconciler"
	"github.com/metal-toolbox/gov-okta-addon/internal/srv"
	"github.com/metal-toolbox/iam-runtime-contrib/iamruntime"
	"github.com/metal-toolbox/iam-runtime/pkg/iam/runtime/identity"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2/clientcredentials"

	governor "github.com/metal-toolbox/governor-api/pkg/client"
)

const (
	// defaultNATSQueueSize is the default for the number of subscribers per subject and queue group
	defaultNATSQueueSize = 10
)

// serveCmd starts the gov-okta-addon service
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the gov-okta-addon service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return serve(cmd.Context(), viper.GetViper())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("listen", "0.0.0.0:8000", "address to listen on")
	viperBindFlag("listen", serveCmd.Flags().Lookup("listen"))

	serveCmd.PersistentFlags().Bool("dry-run", false, "do not make any changes, just log what would be done")
	viperBindFlag("dryrun", serveCmd.PersistentFlags().Lookup("dry-run"))
	serveCmd.PersistentFlags().Bool("skip-delete", true, "do not delete anything in okta during reconcile loop")
	viperBindFlag("skip-delete", serveCmd.PersistentFlags().Lookup("skip-delete"))

	serveCmd.Flags().String("nats-url", "nats://127.0.0.1:4222", "NATS server connection url")
	viperBindFlag("nats.url", serveCmd.Flags().Lookup("nats-url"))
	serveCmd.PersistentFlags().String("nats-creds-file", "", "Path to the file containing the NATS credentials file")
	viperBindFlag("nats.creds-file", serveCmd.PersistentFlags().Lookup("nats-creds-file"))
	serveCmd.Flags().String("nats-subject-prefix", "governor.events", "prefix for NATS subjects")
	viperBindFlag("nats.subject-prefix", serveCmd.Flags().Lookup("nats-subject-prefix"))
	serveCmd.Flags().String("nats-queue-group", "governor.addons.gov-okta-addon", "queue group for load balancing messages across NATS consumers")
	viperBindFlag("nats.queue-group", serveCmd.Flags().Lookup("nats-queue-group"))
	serveCmd.Flags().Int("nats-queue-size", defaultNATSQueueSize, "queue size for load balancing messages across NATS consumers")
	viperBindFlag("nats.queue-size", serveCmd.Flags().Lookup("nats-queue-size"))
	serveCmd.PersistentFlags().Bool("nats-use-runtime-access-token", false, "use IAM runtime to authenticate to NATS")
	viperBindFlag("nats.use-runtime-access-token", serveCmd.PersistentFlags().Lookup("nats-use-runtime-access-token"))

	// IAM runtime
	serveCmd.PersistentFlags().String("iam-runtime-socket", "unix:///tmp/runtime.sock", "IAM runtime socket path")
	viperBindFlag("iam-runtime.socket", serveCmd.PersistentFlags().Lookup("iam-runtime-socket"))
	serveCmd.PersistentFlags().Duration("iam-runtime-timeout", defaultIAMRuntimeTimeoutSeconds*time.Second, "IAM runtime timeout")
	viperBindFlag("iam-runtime.timeout", serveCmd.PersistentFlags().Lookup("iam-runtime-timeout"))

	// Tracing Flags
	serveCmd.Flags().Bool("tracing", false, "enable tracing support")
	viperBindFlag("tracing.enabled", serveCmd.Flags().Lookup("tracing"))
	serveCmd.Flags().String("tracing-provider", "jaeger", "tracing provider to use")
	viperBindFlag("tracing.provider", serveCmd.Flags().Lookup("tracing-provider"))
	serveCmd.Flags().String("tracing-endpoint", "", "endpoint where traces are sent")
	viperBindFlag("tracing.endpoint", serveCmd.Flags().Lookup("tracing-endpoint"))
	serveCmd.Flags().String("tracing-environment", "production", "environment value in traces")
	viperBindFlag("tracing.environment", serveCmd.Flags().Lookup("tracing-environment"))

	serveCmd.Flags().String("audit-log-path", "/app-audit/audit.log", "file path to write audit logs to.")
	viperBindFlag("audit.log-path", serveCmd.Flags().Lookup("audit-log-path"))

	// Okta related flags
	serveCmd.Flags().String("okta-url", "https://example.okta.com", "url for Okta client calls")
	viperBindFlag("okta.url", serveCmd.Flags().Lookup("okta-url"))
	serveCmd.Flags().String("okta-token", "", "token for access to the Okta API")
	viperBindFlag("okta.token", serveCmd.Flags().Lookup("okta-token"))
	serveCmd.Flags().Bool("okta-nocache", false, "disable the okta client cache, useful for development")
	viperBindFlag("okta.nocache", serveCmd.Flags().Lookup("okta-nocache"))

	// Governor related flags
	serveCmd.Flags().String("governor-url", "https://api.governor.metalkube.net", "url of the governor api")
	viperBindFlag("governor.url", serveCmd.Flags().Lookup("governor-url"))
	serveCmd.Flags().String("governor-client-id", "gov-okta-addon-governor", "oauth client ID for client credentials flow")
	viperBindFlag("governor.client-id", serveCmd.Flags().Lookup("governor-client-id"))
	serveCmd.Flags().String("governor-client-secret", "", "oauth client secret for client credentials flow")
	viperBindFlag("governor.client-secret", serveCmd.Flags().Lookup("governor-client-secret"))
	serveCmd.Flags().String("governor-token-url", "http://hydra:4444/oauth2/token", "url used for client credential flow")
	viperBindFlag("governor.token-url", serveCmd.Flags().Lookup("governor-token-url"))
	serveCmd.Flags().String("governor-audience", "https://api.governor.metalkube.net", "oauth audience for client credential flow")
	viperBindFlag("governor.audience", serveCmd.Flags().Lookup("governor-audience"))

	// Reconciler flags
	serveCmd.Flags().Duration("reconciler-interval", reconciler.DefaultReconcileInterval, "interval for the reconciler loop")
	viperBindFlag("reconciler.interval", serveCmd.Flags().Lookup("reconciler-interval"))
	serveCmd.Flags().Duration("eventlog-interval", reconciler.DefaultEventlogPollerInterval, "run interval for the okta eventlog poller")
	viperBindFlag("eventlog.interval", serveCmd.Flags().Lookup("eventlog-interval"))
	serveCmd.Flags().Duration("eventlog-lookback", reconciler.DefaultEventlogColdStartLookback, "coldstart lookback time period for the okta eventlog poller")
	viperBindFlag("eventlog.lookback", serveCmd.Flags().Lookup("eventlog-lookback"))
	serveCmd.Flags().Bool("reconciler-locking", false, "enable reconciler locking and leader election")
	viperBindFlag("reconciler.locking", serveCmd.Flags().Lookup("reconciler-locking"))
}

func serve(cmdCtx context.Context, _ *viper.Viper) error {
	initTracing()

	if err := validateMandatoryFlags(); err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(cmdCtx)

	go func() {
		<-c
		cancel()
	}()

	auditpath := viper.GetString("audit.log-path")

	if auditpath == "" {
		logger.Fatal("failed starting server. Audit log file path can't be empty")
	}

	// WARNING: This will block until the file is available;
	// make sure an initContainer creates the file
	auf, auerr := audithelpers.OpenAuditLogFileUntilSuccessWithContext(ctx, auditpath)
	if auerr != nil {
		logger.Fatalw("couldn't open audit file.", "error", auerr)
	}
	defer auf.Close()

	nc, natsClose, err := newNATSConnection(
		viper.GetString("nats.creds-file"),
		viper.GetString("nats.url"))
	if err != nil {
		logger.Fatalw("failed to create NATS client connection", "error", err)
	}

	defer natsClose()

	natsClient, err := srv.NewNATSClient(
		srv.WithNATSLogger(logger.Desugar()),
		srv.WithNATSConn(nc),
		srv.WithNATSPrefix(viper.GetString("nats.subject-prefix")),
		srv.WithNATSQueueGroup(viper.GetString(("nats.queue-group")), viper.GetInt(("nats.queue-size"))),
	)
	if err != nil {
		logger.Fatalw("failed creating new NATS client", "error", err)
	}

	oc, err := okta.NewClient(
		okta.WithLogger(logger.Desugar()),
		okta.WithURL(viper.GetString("okta.url")),
		okta.WithToken(viper.GetString("okta.token")),
		okta.WithCache((!viper.GetBool("okta.nocache"))),
	)
	if err != nil {
		return err
	}

	gc, err := governor.NewClient(
		governor.WithLogger(logger.Desugar()),
		governor.WithURL(viper.GetString("governor.url")),
		governor.WithClientCredentialConfig(&clientcredentials.Config{
			ClientID:       viper.GetString("governor.client-id"),
			ClientSecret:   viper.GetString("governor.client-secret"),
			TokenURL:       viper.GetString("governor.token-url"),
			EndpointParams: url.Values{"audience": {viper.GetString("governor.audience")}},
			Scopes: []string{
				"read:governor:users",
				"create:governor:users",
				"update:governor:users",
				"read:governor:groups",
				"read:governor:organizations",
			},
		}),
	)
	if err != nil {
		return err
	}

	var locker *natslock.Locker

	if viper.GetBool("reconciler.locking") {
		l, err := newNATSLocker(nc)
		if err != nil {
			logger.Warnw("failed to initialize NATS locker", "error", err)
		}

		if l != nil {
			locker = l
		}
	}

	rec := reconciler.New(
		reconciler.WithAuditEventWriter(auditevent.NewDefaultAuditEventWriter(auf)),
		reconciler.WithLogger(logger.Desugar()),
		reconciler.WithIntervals(viper.GetDuration("reconciler.interval"), viper.GetDuration("eventlog.interval"), viper.GetDuration("eventlog.lookback")),
		reconciler.WithGovernorClient(gc),
		reconciler.WithOktaClient(oc),
		reconciler.WithLocker(locker),
		reconciler.WithDryRun(viper.GetBool("dryrun")),
		reconciler.WithSkipDelete(viper.GetBool("skip-delete")),
	)

	server := &srv.Server{
		Debug:           viper.GetBool("logging.debug"),
		DryRun:          viper.GetBool("dryrun"),
		Listen:          viper.GetString("listen"),
		Logger:          logger.Desugar(),
		AuditFileWriter: auf,
		NATSClient:      natsClient,
		Reconciler:      rec,
	}

	logger.Infow("starting server",
		"address", viper.GetString("listen"),
		"dryrun", server.DryRun,
		"skip-delete", viper.GetBool("skip-delete"),
		"governor-url", viper.GetString("governor.url"),
		"okta-url", viper.GetString("okta.url"),
	)

	if err := server.Run(ctx); err != nil {
		logger.Fatalw("failed starting server", "error", err)
	}

	return nil
}

// newNATSConnection creates a new NATS connection
func newNATSConnection(credsFile, url string) (*nats.Conn, func(), error) {
	opts := []nats.Option{
		nats.Name(appName),
	}

	if credsFile != "" {
		opts = append(opts, nats.UserCredentials(credsFile))
	} else {
		return nil, nil, ErrMissingNATSCreds
	}

	if viper.GetBool("nats.use-runtime-access-token") {
		rt, err := iamruntime.NewClient(viper.GetString("iam-runtime.socket"))
		if err != nil {
			return nil, nil, err
		}

		timeout := viper.GetDuration("iam-runtime.timeout")

		opts = append(opts, nats.UserInfoHandler(func() (string, string) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			iamCreds, err := rt.GetAccessToken(ctx, &identity.GetAccessTokenRequest{})
			if err != nil {
				logger.Errorw("failed to get an access token from the iam-runtime", "error", err)
				return appName, ""
			}

			return appName, iamCreds.Token
		}))
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, nil, err
	}

	return nc, nc.Close, nil
}

// newNATSLocker creates a new NATS jetstream locker from a NATS connection
func newNATSLocker(nc *nats.Conn) (*natslock.Locker, error) {
	jets, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	const timePastInterval = 10 * time.Second

	bucketName := appName + "-lock"
	ttl := viper.GetDuration("reconciler.interval") + timePastInterval

	kvStore, err := natslock.NewKeyValue(jets, bucketName, ttl)
	if err != nil {
		return nil, err
	}

	return natslock.New(
		natslock.WithKeyValueStore(kvStore),
		natslock.WithLogger(logger.Desugar()),
	)
}

// validateMandatoryFlags collects the mandatory flag validation
func validateMandatoryFlags() error {
	errs := []error{}

	if viper.GetString("nats.url") == "" {
		errs = append(errs, ErrNATSURLRequired)
	}

	if viper.GetString("okta.url") == "" {
		errs = append(errs, ErrOktaURLRequired)
	}

	if viper.GetString("okta.token") == "" {
		errs = append(errs, ErrOktaTokenRequired)
	}

	if viper.GetString("governor.url") == "" {
		errs = append(errs, ErrGovernorURLRequired)
	}

	if viper.GetString("governor.client-id") == "" {
		errs = append(errs, ErrGovernorClientIDRequired)
	}

	if viper.GetString("governor.client-secret") == "" {
		errs = append(errs, ErrGovernorClientSecretRequired)
	}

	if viper.GetString("governor.token-url") == "" {
		errs = append(errs, ErrGovernorClientTokenURLRequired)
	}

	if viper.GetString("governor.audience") == "" {
		errs = append(errs, ErrGovernorClientAudienceRequired)
	}

	return errors.Join(errs...)
}
