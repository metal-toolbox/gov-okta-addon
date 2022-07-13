package cmd

import (
	"context"
	"os"
	"os/signal"

	audithelpers "github.com/metal-toolbox/auditevent/helpers"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.equinixmetal.net/gov-okta-addon/internal/okta"
	"go.equinixmetal.net/gov-okta-addon/internal/srv"
)

// serveCmd startes the gov-okta-addon service
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

	serveCmd.Flags().String("nats-url", "nats://127.0.0.1:4222", "NATS server connection url")
	viperBindFlag("nats.url", serveCmd.Flags().Lookup("nats-url"))

	serveCmd.Flags().String("nats-token", "", "NATS auth token")
	viperBindFlag("nats.token", serveCmd.Flags().Lookup("nats-token"))

	serveCmd.Flags().String("nats-subject-prefix", "equinixmetal.governor.addons", "prefix for NATS subjects")
	viperBindFlag("nats.subject-prefix", serveCmd.Flags().Lookup("nats-subject-prefix"))

	serveCmd.Flags().String("nats-queue-group", "equinixmetal.governor.addons.gov-okta-addon", "queue group for load balancing messages across NATS consumers")
	viperBindFlag("nats.queue-group", serveCmd.Flags().Lookup("nats-queue-group"))

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
}

func serve(cmdCtx context.Context, v *viper.Viper) error {
	initTracing()

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
		srv.WithNATSQueueGroup(viper.GetString(("nats.queue-group"))),
	)
	if err != nil {
		logger.Fatalw("failed creating new NATS client", "error", err)
	}

	oc, err := okta.NewClient(
		okta.WithLogger(logger.Desugar()),
		okta.WithURL(viper.GetString("okta.url")),
		okta.WithToken(viper.GetString("okta.token")),
	)
	if err != nil {
		return err
	}

	server := &srv.Server{
		Debug:           viper.GetBool("logging.debug"),
		Listen:          viper.GetString("listen"),
		Logger:          logger.Desugar(),
		AuditFileWriter: auf,
		NATSClient:      natsClient,
		OktaClient:      oc,
	}

	logger.Infow("starting server", "address", viper.GetString("listen"))

	if err := server.Run(ctx); err != nil {
		logger.Fatalw("failed starting server", "error", err)
	}

	return nil
}

// newNATSConnection creates a new NATS connection
// TODO: modify for auth/tls settings once we finalize NATS deployment details
func newNATSConnection() (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Token(viper.GetString("nats.token")),
	}

	if viper.GetBool("development") {
		logger.Debug("enabling development settings")
	}

	return nats.Connect(
		viper.GetString("nats-url"),
		opts...,
	)
}
