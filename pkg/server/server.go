package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/berlingoqc/logviewer/pkg/log/client/config"
)

type Server struct {
	config     *config.ContextConfig
	router     *http.ServeMux
	httpServer *http.Server
	logger     *slog.Logger
	port       string
	host       string
}

func NewServer(host, port string, cfg *config.ContextConfig, logger *slog.Logger) *Server {
	router := http.NewServeMux()
	s := &Server{
		config: cfg,
		router: router,
		logger: logger,
		port:   port,
		host:   host,
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/health", s.healthHandler)
	s.router.HandleFunc("/query/logs", s.queryLogsHandler)
	s.router.HandleFunc("/query/fields", s.queryFieldsHandler)
	s.router.HandleFunc("/contexts", s.contextsHandler)
	s.router.HandleFunc("/contexts/", s.contextsHandler)
}

func (s *Server) Start() error {
	handler := s.chainMiddleware(s.router, s.corsMiddleware, s.requestIDMiddleware, s.loggingMiddleware, s.recoveryMiddleware)

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", s.host, s.port),
		Handler: handler,
	}

	// Channel to listen for errors starting the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		s.logger.Info("starting server", "addr", s.httpServer.Addr)
		serverErrors <- s.httpServer.ListenAndServe()
	}()

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a shutdown signal or a server error
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-shutdown:
		s.logger.Info("shutdown signal received", "signal", sig)

		// Create a context with a timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Attempt to gracefully shutdown the server
		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.logger.Error("graceful shutdown failed", "err", err)
			return s.httpServer.Close()
		}
		s.logger.Info("server shutdown gracefully")
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping server")
	return s.httpServer.Shutdown(ctx)
}
