package server

import (
	"context"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Service is a http.Handler with Startup and Shutdown lifecycle
type Service interface {
	http.Handler

	// Startup controls app startup
	Startup(ctx context.Context) error

	// Shutdown controls app shutdown
	Shutdown(ctx context.Context) error
}

// Server wraps the Service with additional http and app lifecycle handling
type Server struct {
	http.Server
	App             Service
	Address         string
	Port            int
	CertFile        string
	KeyFile         string
	PathPrefix      string
	StartupTimeout  time.Duration
	ShutdownTimeout time.Duration
	Logger          *zap.Logger
	Debug           bool
}

// New create new Server
func New(app Service, options ...Option) *Server {
	s := &Server{}
	s.App = app
	s.Port = 8000
	s.MaxHeaderBytes = 1 << 20
	s.StartupTimeout = time.Second * 10
	s.ShutdownTimeout = time.Second * 10
	s.Logger = zap.NewNop()

	s.Handler = pathHandler(http.MethodGet, map[string]http.HandlerFunc{
		"/favicon.ico": handleOk,
		"/healthcheck": handleOk,
	})(s.App)

	for _, option := range options {
		option(s)
	}
	if s.PathPrefix != "" {
		s.Handler = http.StripPrefix(s.PathPrefix, s.Handler)
	}
	s.Handler = s.panicHandler(s.Handler)
	s.Addr = s.Address + ":" + strconv.Itoa(s.Port)

	return s
}

func (s *Server) Run() {
	s.startup()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.listenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("listen", zap.Error(err))
		}
	}()
	s.Logger.Info("listen", zap.String("addr", s.Addr))
	<-done

	s.shutdown()
}

func (s *Server) startup() {
	ctx, cancel := context.WithTimeout(context.Background(), s.StartupTimeout)
	defer cancel()
	if err := s.App.Startup(ctx); err != nil {
		s.Logger.Fatal("app-startup", zap.Error(err))
	}
}

func (s *Server) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
	defer cancel()
	s.Logger.Info("shutdown")
	if err := s.Shutdown(ctx); err != nil {
		s.Logger.Error("server-shutdown", zap.Error(err))
	}
	if err := s.App.Shutdown(ctx); err != nil {
		s.Logger.Error("app-shutdown", zap.Error(err))
	}
}

func (s *Server) listenAndServe() error {
	if s.CertFile != "" && s.KeyFile != "" {
		return s.ListenAndServeTLS(s.CertFile, s.KeyFile)
	}
	return s.ListenAndServe()
}
