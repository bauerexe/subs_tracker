package http

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"os"
	cfg "subs_tracker/internal/config"
	"subs_tracker/internal/gateways/http/mw"
	"subs_tracker/internal/usecase"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

type Server struct {
	host            string
	port            uint16
	shutdownTimeout time.Duration
	router          *gin.Engine
	log             *slog.Logger
	srv             *http.Server
}

type UseCases struct {
	Sub *usecase.Subscription
}

func NewServer(useCases UseCases, cfg cfg.Config, log *slog.Logger, options ...func(server *Server)) *Server {
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

func WithHost(host string) func(*Server) {
	return func(s *Server) {
		if host != "" {
			s.host = host
		}
	}
}

func WithPort(port uint16) func(*Server) {
	return func(s *Server) {
		if port != 0 {
			s.port = port
		}
	}
}

func WithLogger(log *slog.Logger) func(*Server) {
	return func(s *Server) {
		if log != nil {
			s.log = log
		}
	}
}

func WithTimeout(timeout time.Duration) func(server *Server) {
	return func(s *Server) {
		if timeout > 0 {
			s.shutdownTimeout = timeout
		}
	}
}

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

func (s *Server) Close() error {
	if s.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}
