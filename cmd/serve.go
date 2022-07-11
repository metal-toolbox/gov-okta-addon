package cmd

import (
	"context"
	"os"
	"os/signal"

	audithelpers "github.com/metal-toolbox/auditevent/helpers"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	server := &srv.Server{
		Debug:           viper.GetBool("logging.debug"),
		Listen:          viper.GetString("listen"),
		Logger:          logger.Desugar(),
		AuditFileWriter: auf,
	}

	logger.Infow("starting server", "address", viper.GetString("listen"))

	if err := server.Run(ctx); err != nil {
		logger.Fatalw("failed starting server", "error", err)
	}

	return nil
}
