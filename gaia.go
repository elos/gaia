package gaia

import (
	"log"
	"net/http"

	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
)

type Middleware struct {
}

type Services struct {
	services.DB
	services.Logger
	services.SMSCommandSessions
	services.WebCommandSessions
	services.AppFileSystem
}

type Gaia struct {
	mux http.Handler
	*Middleware
	*Services
	cancelAll func()
}

func New(ctx context.Context, m *Middleware, s *Services) *Gaia {
	mux, cancelAll := router(ctx, m, s)

	if s.DB == nil {
		log.Fatal("Service DB is nil")
	}

	if s.Logger == nil {
		log.Fatal("Service Logger is nil")
	}

	if s.SMSCommandSessions == nil {
		log.Fatal("Service SMSCommandSessions is nil")
	}

	return &Gaia{
		mux:        mux,
		Middleware: m,
		Services:   s,
		cancelAll:  cancelAll,
	}
}

func (gaia *Gaia) Close() {
	gaia.cancelAll()
}

func (gaia *Gaia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gaia.mux.ServeHTTP(w, r)
}
