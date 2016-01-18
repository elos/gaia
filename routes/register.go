package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
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

	u := models.NewUser()
	u.SetID(db.NewID())
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()

	c := models.NewCredential()
	c.SetID(db.NewID())
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	c.Public = username[0]
	c.Private = password[0]
	c.SetOwner(u)

	if err := db.Save(u); err != nil {
		log.Print("RegisterPOST Error: saving user")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := db.Save(c); err != nil {
		log.Print("RegisterPOST Error: saving credential")
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
