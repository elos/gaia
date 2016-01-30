package gaia

import (
	"net/http"

	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// TODO: re-add CORS to all of these

// basic logging
func logRequest(handle http.HandlerFunc, logger services.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s", r.Method, r.URL)
		handle(w, r)
	}
}

func router(ctx context.Context, m *Middleware, s *Services) (http.Handler, context.CancelFunc) {
	mux := http.NewServeMux()
	requestBackground, cancelAll := context.WithCancel(ctx)

	// /register/
	mux.HandleFunc(routes.Register, logRequest(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			routes.RegisterPOST(requestBackground, w, r, s.DB)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /record/
	mux.HandleFunc(routes.Record, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
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
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
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

	// /record/changes/
	mux.HandleFunc(routes.RecordChanges, logRequest(websocket.Handler(
		routes.ContextualizeRecordChangesGET(requestBackground, s.DB, s.Logger),
	).ServeHTTP, s.Logger))

	// /command/sms/
	mux.HandleFunc(routes.CommandSMS, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx := requestBackground

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
		routes.ContextualizeCommandWebGET(requestBackground, s.DB, s.Logger),
	).ServeHTTP, s.Logger))

	return mux, cancelAll
}
