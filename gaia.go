package gaia

import (
	"log"
	"net/http"

	"github.com/elos/gaia/services"
	"github.com/gorilla/context"
)

type Middleware struct {
}

type Services struct {
	services.DB
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

	return &Gaia{
		mux:        mux,
		Middleware: m,
		Services:   s,
	}
}

func (gaia *Gaia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context.ClearHandler(http.HandlerFunc(gaia.mux.ServeHTTP)).ServeHTTP(w, r)
}
