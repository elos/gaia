package routes

import (
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
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

func LoginPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
	l := logger.WithPrefix("LoginPOST: ")
	u, ok := user.FromContext(ctx)
	if !ok {
		l.Print("failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	s := models.NewSessionForUser(u)
	s.SetID(db.NewID())

	if err := db.Save(s); err != nil {
		l.Printf("failed to save session: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, cookie(s))
	http.Redirect(w, r, "/", http.StatusFound)
}

func LoginGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger, webui services.WebUIClient) {
	resp, err := webui.LoginGET(ctx, new(records.LoginGETRequest))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(int(resp.Code))
	w.Write(resp.Body)
}
