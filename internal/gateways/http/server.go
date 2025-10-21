package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	cfg "subs_tracker/internal/config"
	"subs_tracker/internal/gateways/http/mw"
	"subs_tracker/internal/usecase"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

// Server holds HTTP server address, router, logger, and graceful shutdown settings.
type Server struct {
	host            string
	port            uint16
	shutdownTimeout time.Duration
	router          *gin.Engine
	log             *slog.Logger
	srv             *http.Server
}

// UseCases bundles application use cases injected into HTTP handlers.
type UseCases struct {
	Sub *usecase.Subscription
}

// New constructs a Server with defaults, applies options, and wires the Gin router.
func New(useCases UseCases, cfg cfg.Config, log *slog.Logger, options ...func(server *Server)) *Server {
	r := SetupGin(cfg, useCases, log)

	s := &Server{
		host:            "localhost",
		port:            8080,
		router:          r,
		log:             slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		shutdownTimeout: 5 * time.Second,
	}

	for _, o := range options {
		o(s)
	}

	if s.shutdownTimeout <= 0 {
		s.shutdownTimeout = 5 * time.Second
	}

	return s
}

// WithHost returns an option that sets the server host.
func WithHost(host string) func(*Server) {
	return func(s *Server) {
		if host != "" {
			s.host = host
		}
	}
}

// WithPort returns an option that sets the server port.
func WithPort(port uint16) func(*Server) {
	return func(s *Server) {
		if port != 0 {
			s.port = port
		}
	}
}

// WithLogger returns an option that sets the server logger.
func WithLogger(log *slog.Logger) func(*Server) {
	return func(s *Server) {
		if log != nil {
			s.log = log
		}
	}
}

// WithTimeout returns an option that sets the graceful shutdown timeout.
func WithTimeout(timeout time.Duration) func(server *Server) {
	return func(s *Server) {
		if timeout > 0 {
			s.shutdownTimeout = timeout
		}
	}
}

// SetupGin configures Gin mode, middleware, CORS, and routes from the provided config.
func SetupGin(cfg cfg.Config, useCases UseCases, log *slog.Logger) *gin.Engine {
	switch cfg.Env {
	case envLocal:
		gin.SetMode(gin.DebugMode)
	case envDev:
		gin.SetMode(gin.DebugMode)
	case envProd:
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()

	r.Use(mw.RecoveryWithSlog(log))
	r.Use(mw.GinSlog(log))

	origins := cfg.Server.CORSOrigins
	if len(origins) == 0 {
		origins = buildAllowedOrigins(cfg)
	}
	origins = append(origins, []string{"http://localhost:8082", "http://127.0.0.1:8082"}...)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	setupRouter(r, useCases)
	return r
}

// buildAllowedOrigins derives default allowed CORS origins from the server host and swagger port.
func buildAllowedOrigins(c cfg.Config) []string {
	host := c.Server.Host
	if host == "" {
		host = "127.0.0.1"
	}
	swPort := os.Getenv("SWAGGER_PORT_HOST")
	if swPort == "" {
		swPort = "8082"
	}

	return []string{
		fmt.Sprintf("http://%s:%s", host, swPort),
		fmt.Sprintf("https://%s:%s", host, swPort),
	}
}

// Run starts the HTTP server, listens for context cancellation, and shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	s.srv = srv

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server started", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		<-errCh
		s.log.Info("server shutdown complete")
		return nil
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}

// Close gracefully shuts down the underlying HTTP server if it is running.
func (s *Server) Close() error {
	if s.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}
