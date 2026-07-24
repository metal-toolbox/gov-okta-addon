package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/metal-toolbox/addonx/natslock"
	"github.com/metal-toolbox/auditevent"
	audithelpers "github.com/metal-toolbox/auditevent/helpers"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	governor "github.com/metal-toolbox/governor-api/pkg/client"
	govcfg "github.com/metal-toolbox/governor-api/pkg/configs"
	sdkcfg "github.com/metal-toolbox/governor-extension-sdk/pkg/configs"
	extserver "github.com/metal-toolbox/governor-extension-sdk/pkg/server"

	"github.com/metal-toolbox/gov-okta-addon/internal/configs"
	"github.com/metal-toolbox/gov-okta-addon/internal/reconciler"
	"github.com/metal-toolbox/gov-okta-addon/internal/srv"
)

const govClientTimeout = 10 * time.Second

// serveCmd starts the gov-okta-addon service
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the gov-okta-addon service",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	v := viper.GetViper()
	flags := serveCmd.Flags()

	sdkcfg.MustServerFlags(v, flags)
	sdkcfg.MustNATSFlags(v, flags)
	configs.MustReconcilerFlags(v, flags)
}

func serve(cmdCtx context.Context) error {
	if err := validateMandatoryFlags(); err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(cmdCtx)

	go func() {
		<-c
		logger.Debug("received shutdown signal, cancel called")
		cancel()
	}()

	tp, traceShutdown, err := configs.AppConfig.Tracing.InitTracing(ctx, appName)
	if err != nil {
		logger.Fatalw("failed to initialize tracing", "error", err)
	}

	tracer := tp.Tracer(appName)

	defer func() {
		logger.Info("shutting down otel tracer")

		if err := traceShutdown(context.Background()); err != nil {
			logger.Desugar().Error("failed to shutdown tracer provider", zap.Error(err))
		}
	}()

	auditpath := configs.AppConfig.Audit.LogPath

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

	// NATS connection
	nc, err := configs.AppConfig.NATSConn(ctx, appName, govcfg.WithLogger(logger.Desugar()))
	if err != nil {
		logger.Fatalw("failed to create NATS client connection", "error", err)
	}

	defer nc.Close()

	natsClient, err := extserver.NewNATSClient(
		extserver.WithNATSLogger(logger.Desugar()),
		extserver.WithNATSConn(nc),
		extserver.WithNATSPrefix(configs.AppConfig.NATS.SubjectPrefix),
		extserver.WithNATSQueueGroup(configs.AppConfig.NATS.QueueGroup, configs.AppConfig.NATS.QueueSize),
		extserver.WithNATSTracer(tracer),
	)
	if err != nil {
		logger.Fatalw("failed creating new NATS client", "error", err)
	}

	oc, err := configs.NewOktaClient(logger.Desugar())
	if err != nil {
		return err
	}

	gc, err := configs.NewGovernorClient(
		ctx,
		governor.WithLogger(logger.Desugar()),
		governor.WithHTTPClient(&http.Client{
			Timeout:   govClientTimeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}),
	)
	if err != nil {
		logger.Fatalw("failed creating governor client", "error", err)
	}

	var locker *natslock.Locker

	if configs.AppConfig.Reconciler.Locking {
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
		reconciler.WithIntervals(
			configs.AppConfig.Reconciler.Interval,
			configs.AppConfig.Eventlog.Interval,
			configs.AppConfig.Eventlog.Lookback,
		),
		reconciler.WithGovernorClient(gc),
		reconciler.WithOktaClient(oc),
		reconciler.WithLocker(locker),
		reconciler.WithDryRun(configs.AppConfig.DryRun),
		reconciler.WithSkipDelete(configs.AppConfig.SkipDelete),
	)

	proc := srv.NewProcessor(
		rec,
		srv.WithLogger(logger.Desugar().With(zap.String("component", "events-processor"))),
		srv.WithTracer(tracer),
	)

	// gov-okta-addon is a conventional (non-interactive) governor addon that only
	// processes events, so no extension ID or ERDs are registered.
	server := extserver.NewServer(
		configs.AppConfig.Server.Listen,
		"", // extensionID
		"", // erdDir
		extserver.WithEventProcessor(proc),
		extserver.WithAuditFileWriter(auf),
		extserver.WithLogger(logger.Desugar()),
		extserver.WithDebug(configs.AppConfig.Logging.Debug),
		extserver.WithGovernorClient(gc),
		extserver.WithNATSClient(natsClient),
		extserver.WithTracer(tracer),
	)

	logger.Infow("starting server",
		"address", configs.AppConfig.Server.Listen,
		"dryrun", configs.AppConfig.DryRun,
		"skip-delete", configs.AppConfig.SkipDelete,
		"governor-url", configs.AppConfig.Governor.URL,
		"okta-url", configs.AppConfig.Okta.URL,
	)

	// the SDK event router only handles event-driven reconciliation, so run the
	// periodic reconciler loop alongside it
	go rec.Run(ctx)

	if err := server.Run(ctx); err != nil {
		logger.Fatalw("failed starting server", "error", err)
	}

	rec.Stop()

	return nil
}

// newNATSLocker creates a new NATS jetstream locker from a NATS connection
func newNATSLocker(nc *nats.Conn) (*natslock.Locker, error) {
	jets, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	const timePastInterval = 10 * time.Second

	bucketName := appName + "-lock"
	ttl := configs.AppConfig.Reconciler.Interval + timePastInterval

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
	if err := configs.AppConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	errs := []string{}

	if configs.AppConfig.Okta.URL == "" {
		errs = append(errs, ErrOktaURLRequired.Error())
	}

	if configs.AppConfig.Okta.Token == "" {
		errs = append(errs, ErrOktaTokenRequired.Error())
	}

	if configs.AppConfig.Governor.URL == "" {
		errs = append(errs, ErrGovernorURLRequired.Error())
	}

	// under workload identity federation the client-credentials settings are
	// not used, so only require them when WIF is disabled
	if !configs.AppConfig.Governor.WorkloadIdentity {
		if configs.AppConfig.Governor.ClientID == "" {
			errs = append(errs, ErrGovernorClientIDRequired.Error())
		}

		if configs.AppConfig.Governor.ClientSecret == "" {
			errs = append(errs, ErrGovernorClientSecretRequired.Error())
		}

		if configs.AppConfig.Governor.TokenURL == "" {
			errs = append(errs, ErrGovernorClientTokenURLRequired.Error())
		}

		if configs.AppConfig.Governor.Audience == "" {
			errs = append(errs, ErrGovernorClientAudienceRequired.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf("%s", strings.Join(errs, "\n")) //nolint:govet,err113,staticcheck
}
