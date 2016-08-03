package gaia

import (
	"log"
	"net/http"

	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// TODO: re-add CORS to all of these
const (
	AllowOriginHeader      = "Access-Control-Allow-Origin"
	AllowCredentialsHeader = "Access-Control-Allow-Credentials"
	AllowMethodsHeader     = "Access-Control-Allow-Methods"
	AllowHeadersHeader     = "Access-Control-Allow-Headers"
)

// basic logging
func logRequest(handle http.HandlerFunc, logger services.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s", r.Method, r.URL)
		handle(w, r)
	}
}

func cors(handle http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(AllowOriginHeader, r.Header.Get("Origin"))
		w.Header().Add(AllowCredentialsHeader, "true")
		w.Header().Add(AllowMethodsHeader, "POST")
		w.Header().Add(AllowMethodsHeader, "DELETE")
		w.Header().Add(AllowMethodsHeader, "GET")
		w.Header().Add(AllowMethodsHeader, "OPTIONS")
		w.Header().Add(AllowHeadersHeader, "Authorization")
		handle(w, r)
	}
}

func router(ctx context.Context, m *Middleware, s *Services) (http.Handler, context.CancelFunc) {
	mux := http.NewServeMux()
	requestBackground, cancelAll := context.WithCancel(ctx)

	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(s.AppFileSystem)))

	mux.HandleFunc(routes.Index, logRequest(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			log.Print("index")
			ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
			if !ok {
				return
			}
			routes.RegisterGET(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /records/query/
	mux.HandleFunc(routes.RecordsQuery, logRequest(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
			if !ok {
				return
			}
			routes.Records.QueryGET(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /records/new/
	mux.HandleFunc(routes.RecordsNew, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "GET":
			routes.Records.NewGET(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /records/create/
	mux.HandleFunc(routes.RecordsCreate, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "GET":
			routes.Records.CreateGET(ctx, w, r, s.DB, s.Logger)
		case "POST":
			routes.Records.CreatePOST(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /records/edit/
	mux.HandleFunc(routes.RecordsEdit, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
		if !ok {
			return
		}
		switch r.Method {
		case "GET":
			routes.Records.EditGET(ctx, w, r, s.DB, s.Logger)
		case "POST":
			routes.Records.EditPOST(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /records/view/
	mux.HandleFunc(routes.RecordsView, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
		if !ok {
			return
		}
		switch r.Method {
		case "GET":
			routes.Records.ViewGET(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /register/
	mux.HandleFunc(routes.Register, logRequest(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			routes.RegisterPOST(requestBackground, w, r, s.DB, s.Logger)
		case "GET":
			routes.RegisterGET(requestBackground, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /login/
	mux.HandleFunc(routes.Login, logRequest(func(w http.ResponseWriter, r *http.Request) {

		switch r.Method {
		case "POST":
			ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
			if !ok {
				return
			}

			routes.LoginPOST(ctx, w, r, s.DB, s.Logger)
		case "GET":
			routes.LoginGET(requestBackground, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// /record/
	mux.HandleFunc(routes.Record, logRequest(cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			routes.RecordOPTIONS(requestBackground, w, r)
			return
		}

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
	}), s.Logger))

	// /record/query/
	mux.HandleFunc(routes.RecordQuery, logRequest(cors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			routes.RecordOPTIONS(requestBackground, w, r)
			return
		}

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
	}), s.Logger))

	// /event/
	mux.HandleFunc(routes.Event, logRequest(cors(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "POST":
			routes.EventPOST(ctx, w, r, s.DB, s.Logger)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}), s.Logger))

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

	// /mobile/location/
	mux.HandleFunc(routes.MobileLocation, logRequest(func(w http.ResponseWriter, r *http.Request) {
		ctx, ok := routes.Authenticate(requestBackground, w, r, s.Logger, s.DB)
		if !ok {
			return
		}

		switch r.Method {
		case "POST":
			routes.MobileLocationPOST(ctx, w, r, s.Logger, s.DB)
		default:
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
	}, s.Logger))

	// Handle letsencrypt
	fs := http.FileServer(http.Dir("/var/www/elos/"))
	mux.Handle("/.well-known/", logRequest(func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	}, s.Logger))

	return mux, cancelAll
}
