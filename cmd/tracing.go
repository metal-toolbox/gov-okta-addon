package cmd

import (
	"context"
	"net/url"
	"time"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const timeout = 10 * time.Second

func initTracing() {
	if viper.GetBool("tracing.enabled") {
		initTracer(
			viper.GetString("tracing.endpoint"),
			viper.GetBool("tracing.insecure"),
		)
	}
}

// initTracer returns an OpenTelemetry TracerProvider configured to use
// the OTLP GRPC exporter that will send spans to the provided endpoint. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func initTracer(endpoint string, insecure bool) *tracesdk.TracerProvider {
	_, err := url.Parse(endpoint)
	if err != nil {
		logger.Fatalw("invalid tracing endpoint", "error", err)
	}

	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithTimeout(timeout),
	}

	if insecure {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	}

	exp, err := otlptracegrpc.New(context.Background(), exporterOpts...)
	if err != nil {
		logger.Fatalf("failed to initialize tracing", "error", err)
	}

	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in an Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("gov-okta-addon"),
			attribute.String("environment", viper.GetString("tracing.environment")),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp
}
