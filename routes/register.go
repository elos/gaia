package routes

import (
	"fmt"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

// These are the expected parameters to the RegisterPOST.
const (
	usernameParam = "username"
	passwordParam = "password"
)

// RegisterPOST handles a request to create a new elos user.
func RegisterPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger, webui services.WebUIClient) {
	l := logger.WithPrefix("RegisterPOST: ")

	if err := r.ParseForm(); err != nil {
		l.Printf("error: parsing form: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	username := r.FormValue(usernameParam)
	if len(username) == 0 {
		l.Print("no username found")
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", usernameParam), http.StatusBadRequest)
		return
	}

	password := r.FormValue(passwordParam)
	if len(password) == 0 {
		l.Print("no password found")
		http.Error(w, fmt.Sprintf("You must specify a %q parameter", passwordParam), http.StatusBadRequest)
		return
	}

	resp, err := webui.RegisterPOST(ctx, &records.RegisterPOSTRequest{
		Public:  username,
		Private: password,
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(int(resp.Code))
	w.Write(resp.Body)
}

func RegisterGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger, webui services.WebUIClient) {
	resp, err := webui.RegisterGET(ctx, new(records.RegisterGETRequest))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(int(resp.Code))
	w.Write(resp.Body)
}
