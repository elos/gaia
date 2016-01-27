package gaia

import (
	"net/http"

	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// basic logging
func logRequest(handle http.HandlerFunc, logger services.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s", r.Method, r.URL)
		handle(w, r)
	}
}

func router(m *Middleware, s *Services) http.Handler {
	mux := http.NewServeMux()

	// /register/
	mux.HandleFunc(routes.Register, logRequest(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			routes.RegisterPOST(context.Background(), w, r, s.DB)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /record/
	mux.HandleFunc(routes.Record, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(context.Background(), w, r, s.Logger, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "GET":
			routes.RecordGET(ctx, w, r, s.Logger, s.DB)
		case "POST":
			routes.RecordPOST(ctx, w, r, s.Logger, s.DB)
		case "DELETE":
			routes.RecordDELETE(ctx, w, r, s.Logger, s.DB)
		case "OPTIONS":
			routes.RecordOPTIONS(ctx, w, r)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /record/query/
	mux.HandleFunc(routes.RecordQuery, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(context.Background(), w, r, s.Logger, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "POST":
			routes.RecordQueryPOST(ctx, w, r, s.Logger, s.DB)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /command/sms/
	mux.HandleFunc(routes.CommandSMS, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		switch r.Method {
		case "POST":
			routes.CommandSMSPOST(ctx, w, r, s.Logger, s.SMSCommandSessions)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /command/web/
	mux.HandleFunc(routes.CommandWeb, logRequest(websocket.Handler(
		routes.ContextualizeCommandWebGET(s.DB, s.Logger),
	).ServeHTTP, s.Logger))

	return mux
}
