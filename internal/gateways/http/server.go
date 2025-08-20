package http

import (
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log/slog"
	"os"
	cfg "subs_tracker/internal/config"
	"subs_tracker/internal/gateways/http/mw"
	"subs_tracker/internal/usecase"
	"sync"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

type Server struct {
	host    string
	port    uint16
	wg      sync.WaitGroup
	timeout time.Duration
	router  *gin.Engine
	log     *slog.Logger
}

type UseCases struct {
	Sub *usecase.Subscription
}

func NewServer(useCases UseCases, cfg cfg.Config, log *slog.Logger, options ...func(server *Server)) *Server {
	r := SetupGin(cfg, useCases, log)

	s := &Server{
		host:   "localhost",
		port:   8080,
		router: r,
		log: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}

	for _, o := range options {
		o(s)
	}

	return s
}

func WithHost(host string) func(*Server) {
	return func(s *Server) {
		s.host = host
	}
}

func WithPort(port uint16) func(*Server) {
	return func(s *Server) {
		s.port = port
	}
}

func WithLogger(log *slog.Logger) func(*Server) {
	return func(s *Server) {
		s.log = log
	}
}

func WithTimeout(timeout time.Duration) func(server *Server) {
	return func(s *Server) {
		s.timeout = timeout
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
func (s *Server) Run() error {
	return s.router.Run(fmt.Sprintf("%s:%d", s.host, s.port))
}

func (s *Server) Close() error {
	s.wg.Wait()
	s.log.Info("Server closed successfully")
	return nil
}
