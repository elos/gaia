package gaia

import (
	"log"
	"net/http"

	"github.com/elos/gaia/services"
)

type Middleware struct {
}

type Services struct {
	services.DB
	services.Logger
	services.SMSCommandSessions
}

type Gaia struct {
	mux http.Handler
	*Middleware
	*Services
}

func New(m *Middleware, s *Services) *Gaia {
	mux := router(m, s)

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
	}
}

func (gaia *Gaia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gaia.mux.ServeHTTP(w, r)
}
