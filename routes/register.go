package routes

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/elos/data"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const usernameParam = "username"
const passwordParam = "password"

func RegisterPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB) {
	if err := r.ParseForm(); err != nil {
		log.Printf("RegisterPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	username := r.Form[usernameParam]
	if len(username) == 0 {
		http.Error(w, "You must specifiy a username", http.StatusBadRequest)
		return
	}

	password := r.Form[passwordParam]
	if len(password) == 0 {
		http.Error(w, "You must specifiy a password", http.StatusBadRequest)
		return
	}

	u, _, err := user.Create(db, username[0], password[0])
	if err != nil {
		log.Printf("RegisterPOST Error: creating user: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bytes, err := json.Marshal(u)
	if err != nil {
		log.Print("RegisterPOST Error: marshalling user json")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bytes)
}
