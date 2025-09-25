package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ptrciafae/hotels-merge/internal/hotels"
)

type Server struct {
	httpServer *http.Server
	handlers   *Handlers
}

func New(store *hotels.HotelStore) *Server {
	handlers := NewHandlers(store)
	mux := http.NewServeMux()

	// home route
	mux.HandleFunc("/", handlers.handleGetAllHotels)

	// config routes
	mux.HandleFunc("GET /hotels", handlers.handleQueryHotels)

	srv := &http.Server{
		Addr:         "127.0.0.1:8085",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		handlers:   handlers,
	}
}

func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
