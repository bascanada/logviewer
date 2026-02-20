package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventType represents the type of SSE event
type EventType string

const (
	EventConfigReloaded EventType = "config-reloaded"
	EventServerError    EventType = "server-error"
)

// Event represents a server-sent event
type Event struct {
	Type EventType              `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// EventBroker manages SSE connections and broadcasts events
type EventBroker struct {
	clients      map[chan Event]struct{}
	clientsMutex sync.RWMutex
	logger       *slog.Logger
}

// NewEventBroker creates a new event broker
func NewEventBroker(logger *slog.Logger) *EventBroker {
	return &EventBroker{
		clients: make(map[chan Event]struct{}),
		logger:  logger,
	}
}

// Subscribe adds a new client to receive events
func (b *EventBroker) Subscribe() chan Event {
	b.clientsMutex.Lock()
	defer b.clientsMutex.Unlock()

	client := make(chan Event, 10) // Buffer to avoid blocking
	b.clients[client] = struct{}{}
	b.logger.Debug("client subscribed to events", "total_clients", len(b.clients))
	return client
}

// Unsubscribe removes a client from receiving events
func (b *EventBroker) Unsubscribe(client chan Event) {
	b.clientsMutex.Lock()
	defer b.clientsMutex.Unlock()

	delete(b.clients, client)
	close(client)
	b.logger.Debug("client unsubscribed from events", "total_clients", len(b.clients))
}

// Broadcast sends an event to all subscribed clients
func (b *EventBroker) Broadcast(event Event) {
	b.clientsMutex.RLock()
	defer b.clientsMutex.RUnlock()

	b.logger.Debug("broadcasting event", "type", event.Type, "clients", len(b.clients))

	for client := range b.clients {
		select {
		case client <- event:
			// Event sent successfully
		case <-time.After(100 * time.Millisecond):
			// Client not reading, skip to avoid blocking
			b.logger.Warn("client not reading events, skipping")
		}
	}
}

// ClientCount returns the number of active clients
func (b *EventBroker) ClientCount() int {
	b.clientsMutex.RLock()
	defer b.clientsMutex.RUnlock()
	return len(b.clients)
}

// eventsHandler handles SSE connections
func (s *Server) eventsHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Make sure we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.logger.Error("streaming not supported")
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Subscribe to events
	eventChan := s.eventBroker.Subscribe()
	defer s.eventBroker.Unsubscribe(eventChan)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"message\":\"connected\"}\n\n")
	flusher.Flush()

	// Use request context for cancellation
	ctx := r.Context()

	// Send heartbeat every 30 seconds to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("client disconnected")
			return

		case event := <-eventChan:
			// Marshal event data
			data, err := json.Marshal(event)
			if err != nil {
				s.logger.Error("failed to marshal event", "err", err)
				continue
			}

			// Send event
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
			flusher.Flush()

		case <-ticker.C:
			// Send heartbeat
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// ConfigWatcher watches config files and triggers reloads
type ConfigWatcher struct {
	watcher      *fsnotify.Watcher
	server       *Server
	configPath   string
	cancelFunc   context.CancelFunc
	logger       *slog.Logger
	isReloading  bool
	reloadMutex  sync.Mutex
	lastReload   time.Time
	debounceTime time.Duration
}

// NewConfigWatcher creates a new config file watcher
func NewConfigWatcher(server *Server, configPath string, logger *slog.Logger) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	cw := &ConfigWatcher{
		watcher:      watcher,
		server:       server,
		configPath:   configPath,
		logger:       logger,
		debounceTime: 1 * time.Second, // Debounce rapid successive changes
	}

	return cw, nil
}

// Start begins watching the config file
func (cw *ConfigWatcher) Start(ctx context.Context) error {
	// Add config file to watcher
	if err := cw.watcher.Add(cw.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	cw.logger.Info("started watching config file", "path", cw.configPath)

	// Start watching in goroutine
	go cw.watch(ctx)

	return nil
}

// watch handles file system events
func (cw *ConfigWatcher) watch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			cw.logger.Info("config watcher stopped")
			return

		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Only handle Write and Create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				cw.logger.Info("config file changed", "op", event.Op.String(), "path", event.Name)
				cw.handleConfigChange(ctx)
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			cw.logger.Error("config watcher error", "err", err)
		}
	}
}

// handleConfigChange reloads config and broadcasts event
func (cw *ConfigWatcher) handleConfigChange(ctx context.Context) {
	cw.reloadMutex.Lock()
	defer cw.reloadMutex.Unlock()

	// Debounce: ignore if reload happened recently
	if time.Since(cw.lastReload) < cw.debounceTime {
		cw.logger.Debug("config change ignored (debounced)")
		return
	}

	if cw.isReloading {
		cw.logger.Debug("config reload already in progress")
		return
	}

	cw.isReloading = true
	defer func() {
		cw.isReloading = false
		cw.lastReload = time.Now()
	}()

	cw.logger.Info("reloading configuration")

	// Reload config via server's reload method
	if err := cw.server.ReloadConfig(ctx); err != nil {
		cw.logger.Error("failed to reload config", "err", err)

		// Broadcast error event
		cw.server.eventBroker.Broadcast(Event{
			Type: EventServerError,
			Data: map[string]interface{}{
				"message": fmt.Sprintf("Failed to reload config: %v", err),
			},
		})
		return
	}

	cw.logger.Info("configuration reloaded successfully")

	// Broadcast success event
	cw.server.eventBroker.Broadcast(Event{
		Type: EventConfigReloaded,
		Data: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"message":   "Configuration reloaded",
		},
	})
}

// Stop stops watching the config file
func (cw *ConfigWatcher) Stop() error {
	cw.logger.Info("stopping config watcher")
	return cw.watcher.Close()
}
