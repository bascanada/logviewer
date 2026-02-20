package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/log/factory"
)

// Server represents the API server instance.
type Server struct {
	config        *config.ContextConfig
	configPath    string
	configMutex   sync.RWMutex
	router        *http.ServeMux
	httpServer    *http.Server
	logger        *slog.Logger
	port          string
	host          string
	searchFactory factory.SearchFactory
	openapiSpec   []byte
	eventBroker   *EventBroker
	configWatcher *ConfigWatcher
}

// NewServer creates a new API server instance.
func NewServer(host, port string, cfg *config.ContextConfig, configPath string, logger *slog.Logger, openapiSpec []byte) (*Server, error) {
	clientFactory, err := factory.GetLogBackendFactory(cfg.Clients)
	if err != nil {
		return nil, err
	}
	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *cfg)
	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()
	eventBroker := NewEventBroker(logger)

	s := &Server{
		config:        cfg,
		configPath:    configPath,
		router:        router,
		logger:        logger,
		port:          port,
		host:          host,
		searchFactory: searchFactory,
		openapiSpec:   openapiSpec,
		eventBroker:   eventBroker,
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.router.HandleFunc("/health", s.healthHandler)
	s.router.HandleFunc("/query/logs", s.queryLogsRouter)
	s.router.HandleFunc("/query/fields", s.queryFieldsRouter)
	s.router.HandleFunc("/contexts", s.contextsHandler)
	s.router.HandleFunc("/contexts/", s.contextsHandler)
	s.router.HandleFunc("/openapi.yaml", s.openapiHandler)
	s.router.HandleFunc("/events", s.eventsHandler)
}

// Start runs the HTTP server and blocks until a signal is received.
func (s *Server) Start() error {
	// Start config watcher if config path is set
	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()

	if s.configPath != "" {
		watcher, err := NewConfigWatcher(s, s.configPath, s.logger)
		if err != nil {
			s.logger.Warn("failed to create config watcher", "err", err)
		} else {
			s.configWatcher = watcher
			if err := s.configWatcher.Start(watchCtx); err != nil {
				s.logger.Warn("failed to start config watcher", "err", err)
			}
		}
	}

	handler := s.chainMiddleware(s.router, s.recoveryMiddleware, s.corsMiddleware, s.requestIDMiddleware, s.loggingMiddleware)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// Create listener first to get the actual assigned port (important when port=0)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Get the actual port (useful when port=0 was requested)
	actualAddr := listener.Addr().(*net.TCPAddr)
	actualPort := actualAddr.Port

	s.httpServer = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Channel to listen for errors starting the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		s.logger.Info("starting server", "addr", listener.Addr().String())
		// Print in a format the VS Code extension can parse
		fmt.Printf("Server listening on port %d\n", actualPort)
		serverErrors <- s.httpServer.Serve(listener)
	}()

	// Channel to listen for shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a shutdown signal or a server error
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}

	case sig := <-shutdown:
		s.logger.Info("shutdown signal received", "signal", sig)

		// Stop config watcher
		watchCancel()
		if s.configWatcher != nil {
			s.configWatcher.Stop()
		}

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

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping server")
	if s.configWatcher != nil {
		s.configWatcher.Stop()
	}
	return s.httpServer.Shutdown(ctx)
}

// ReloadConfig reloads the configuration file and recreates factories
func (s *Server) ReloadConfig(ctx context.Context) error {
	s.configMutex.Lock()
	defer s.configMutex.Unlock()

	s.logger.Info("reloading configuration", "path", s.configPath)

	// Load new config
	newConfig, err := config.LoadContextConfig(s.configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create new factories with new config
	clientFactory, err := factory.GetLogBackendFactory(newConfig.Clients)
	if err != nil {
		return fmt.Errorf("failed to create client factory: %w", err)
	}

	searchFactory, err := factory.GetLogSearchFactory(clientFactory, *newConfig)
	if err != nil {
		return fmt.Errorf("failed to create search factory: %w", err)
	}

	// Atomically update config and factories
	s.config = newConfig
	s.searchFactory = searchFactory

	s.logger.Info("configuration reloaded successfully", "contexts", len(newConfig.Contexts), "clients", len(newConfig.Clients))

	return nil
}

// GetConfig returns a copy of the current configuration (thread-safe)
func (s *Server) GetConfig() *config.ContextConfig {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()
	return s.config
}
