package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

// These are the expected parameters to the RegisterPOST.
const (
	usernameParam = "username"
	passwordParam = "password"
)

// RegisterPOST handles a request to create a new elos user.
func RegisterPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
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

	u, _, err := user.Create(db, username, password)
	if err != nil {
		l.Printf("error: creating user: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(u)
	if err != nil {
		l.Print("error: marshalling user json")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bytes)
}
