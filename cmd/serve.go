package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	audithelpers "github.com/metal-toolbox/auditevent/helpers"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.equinixmetal.net/gov-okta-addon/internal/governor"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/gov-okta-addon/internal/reconciler"
	"go.equinixmetal.net/gov-okta-addon/internal/srv"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// defaultNATSQueueSize is the default for the number of subscribers per subject and queue group
	defaultNATSQueueSize = 10
)

// serveCmd starts the gov-okta-addon service
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "starts the gov-okta-addon service",
	RunE: func(cmd *cobra.Command, args []string) error {
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
	serveCmd.Flags().String("nats-token", "", "NATS auth token")
	viperBindFlag("nats.token", serveCmd.Flags().Lookup("nats-token"))
	serveCmd.Flags().String("nats-nkey", "", "Path to the file containing the NATS nkey keypair")
	viperBindFlag("nats.nkey", serveCmd.Flags().Lookup("nats-nkey"))
	serveCmd.Flags().String("nats-subject-prefix", "equinixmetal.governor.events", "prefix for NATS subjects")
	viperBindFlag("nats.subject-prefix", serveCmd.Flags().Lookup("nats-subject-prefix"))
	serveCmd.Flags().String("nats-queue-group", "equinixmetal.governor.addons.gov-okta-addon", "queue group for load balancing messages across NATS consumers")
	viperBindFlag("nats.queue-group", serveCmd.Flags().Lookup("nats-queue-group"))
	serveCmd.Flags().Int("nats-queue-size", defaultNATSQueueSize, "queue size for load balancing messages across NATS consumers")
	viperBindFlag("nats.queue-size", serveCmd.Flags().Lookup("nats-queue-size"))

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
	serveCmd.Flags().String("okta-url", "https://equinixmetal.okta.com", "url for Okta client calls")
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
	serveCmd.Flags().Duration("reconciler-interval", 1*time.Hour, "interval for the reconciler loop")
	viperBindFlag("reconciler.interval", serveCmd.Flags().Lookup("reconciler-interval"))
}

func serve(cmdCtx context.Context, v *viper.Viper) error {
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
	auf, auerr := audithelpers.OpenAuditLogFileUntilSuccess(auditpath)
	if auerr != nil {
		logger.Fatalw("couldn't open audit file.", "error", auerr)
	}
	defer auf.Close()

	nc, err := newNATSConnection()
	if err != nil {
		logger.Fatalw("failed to create NATS client connection", "error", err)
	}

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
				"read:governor:groups",
				"read:governor:organizations",
			},
		}),
	)
	if err != nil {
		return err
	}

	rec := reconciler.New(
		reconciler.WithAuditEventWriter(auditevent.NewDefaultAuditEventWriter(auf)),
		reconciler.WithLogger(logger.Desugar()),
		reconciler.WithInterval(viper.GetDuration("reconciler.interval")),
		reconciler.WithGovernorClient(gc),
		reconciler.WithOktaClient(oc),
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
func newNATSConnection() (*nats.Conn, error) {
	opts := []nats.Option{}

	if viper.GetBool("development") {
		logger.Debug("enabling development settings")

		opts = append(opts, nats.Token(viper.GetString("nats.token")))
	} else {
		opt, err := nats.NkeyOptionFromSeed(viper.GetString("nats-nkey"))
		if err != nil {
			return nil, err
		}

		opts = append(opts, opt)
	}

	return nats.Connect(
		viper.GetString("nats-url"),
		opts...,
	)
}

// validateMandatoryFlags collects the mandatory flag validation
func validateMandatoryFlags() error {
	errs := []string{}

	if viper.GetString("nats.url") == "" {
		errs = append(errs, ErrNATSURLRequired.Error())
	}

	if viper.GetString("nats.token") == "" && viper.GetString("nats.nkey") == "" {
		errs = append(errs, ErrNATSAuthRequired.Error())
	}

	if viper.GetString("okta.url") == "" {
		errs = append(errs, ErrOktaURLRequired.Error())
	}

	if viper.GetString("okta.token") == "" {
		errs = append(errs, ErrOktaTokenRequired.Error())
	}

	if viper.GetString("governor.url") == "" {
		errs = append(errs, ErrGovernorURLRequired.Error())
	}

	if viper.GetString("governor.client-id") == "" {
		errs = append(errs, ErrGovernorClientIDRequired.Error())
	}

	if viper.GetString("governor.client-secret") == "" {
		errs = append(errs, ErrGovernorClientSecretRequired.Error())
	}

	if viper.GetString("governor.token-url") == "" {
		errs = append(errs, ErrGovernorClientTokenURLRequired.Error())
	}

	if viper.GetString("governor.audience") == "" {
		errs = append(errs, ErrGovernorClientAudienceRequired.Error())
	}

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errs, "\n")) //nolint:goerr113
}
