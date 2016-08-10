package routes

import (
	"log"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

const (
	sessionCookie = "elos-session-token"
)

func cookie(s *models.Session) *http.Cookie {
	return &http.Cookie{
		Name:    sessionCookie,
		Value:   s.Token,
		Expires: s.Expires(),
		Path:    "/",
	}
}

func session(r *http.Request, db data.DB) (*models.Session, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, err
	}
	return models.SessionForToken(db, c.Value)
}

func LoginPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	if err := r.ParseForm(); err != nil {
		log.Printf("r.ParseForm error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp, err := webui.LoginPOST(ctx, &records.LoginPOSTRequest{
		Public:  r.FormValue("public"),
		Private: r.FormValue("private"),
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}

func LoginGET(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	resp, err := webui.LoginGET(ctx, new(records.LoginGETRequest))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
