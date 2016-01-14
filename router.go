package gaia

import (
	"net/http"

	"github.com/elos/gaia/routes"
	"golang.org/x/net/context"
)

func router(m *Middleware, s *Services) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc(routes.Record, func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(context.Background(), w, r, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "GET":
			routes.RecordGET(ctx, w, r, s.DB)
		case "POST":
			routes.RecordPOST(ctx, w, r, s.DB)
		case "DELETE":
			routes.RecordDELETE(ctx, w, r, s.DB)
		case "OPTIONS":
			routes.RecordOPTIONS(ctx, w, r)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	})

	mux.HandleFunc(routes.RecordQuery, func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(context.Background(), w, r, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "POST":
			routes.RecordQueryPOST(ctx, w, r, s.DB)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	})

	return mux
}
