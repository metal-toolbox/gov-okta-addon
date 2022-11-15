package srv

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"

	"go.equinixmetal.net/gov-okta-addon/internal/reconciler"
)

// Server implements the HTTP Server
type Server struct {
	Logger          *zap.Logger
	Listen          string
	Debug           bool
	DryRun          bool
	AuditFileWriter io.Writer
	NATSClient      *NATSClient
	Reconciler      *reconciler.Reconciler
}

var (
	readTimeout     = 10 * time.Second
	writeTimeout    = 20 * time.Second
	corsMaxAge      = 12 * time.Hour
	shutdownTimeout = 5 * time.Second
)

func (s *Server) setup() *gin.Engine {
	// Setup default gin router
	r := gin.New()

	r.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowAllOrigins:  true,
		AllowCredentials: true,
		MaxAge:           corsMaxAge,
	}))

	p := ginprometheus.NewPrometheus("gin")

	// Remove any params from the URL string to keep the number of labels down
	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		return c.FullPath()
	}

	p.Use(r)

	customLogger := s.Logger.With(zap.String("component", "httpsrv"))
	r.Use(
		ginzap.GinzapWithConfig(customLogger, &ginzap.Config{
			TimeFormat: time.RFC3339,
			SkipPaths:  []string{"/healthz", "/healthz/readiness", "/healthz/liveness"},
			UTC:        true,
		}),
	)

	r.Use(ginzap.RecoveryWithZap(s.Logger.With(zap.String("component", "httpsrv")), true))

	tp := otel.GetTracerProvider()
	if tp != nil {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}

		r.Use(otelgin.Middleware(hostname, otelgin.WithTracerProvider(tp)))
	}

	// Health endpoints
	r.GET("/healthz", s.livenessCheck)
	r.GET("/healthz/liveness", s.livenessCheck)
	r.GET("/healthz/readiness", s.readinessCheck)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"message": "invalid request - route not found"})
	})

	return r
}

// NewServer returns a configured server
func (s *Server) NewServer() *http.Server {
	if !s.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return &http.Server{
		Handler:      s.setup(),
		Addr:         s.Listen,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
}

// Run will start the server listening on the specified address
func (s *Server) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	httpsrv := s.NewServer()

	go func() {
		if err := httpsrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()

	go s.Reconciler.Run(ctx)

	if err := s.registerSubscriptionHandlers(); err != nil {
		panic(err)
	}

	<-ctx.Done()

	ctxShutDown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer func() {
		cancel()
	}()

	wg.Add(1)

	go func() {
		defer wg.Done()

		if err := s.shutdownSubscriptions(); err != nil {
			s.Logger.Warn("error shutting down subscription", zap.Error(err))
		}
	}()

	if err := httpsrv.Shutdown(ctxShutDown); err != nil {
		return err
	}

	// wait for clean shutdown
	wg.Wait()

	s.Logger.Info("server shutdown cleanly", zap.String("time", time.Now().UTC().Format(time.RFC3339)))

	return nil
}

// livenessCheck ensures that the server is up and responding
func (s *Server) livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}

// readinessCheck ensures that the server is up and that we are able to process requests.
func (s *Server) readinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "UP",
	})
}
