package routes

import (
	"log"
	"net/http"

	"github.com/elos/gaia/services"
	"github.com/elos/x/records"
	"golang.org/x/net/context"
)

func RegisterGET(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	resp, err := webui.RegisterGET(ctx, new(records.RegisterGETRequest))
	if err != nil {
		log.Print("webui.RegisterGET error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}

func RegisterPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, webui services.WebUIClient) {
	if err := r.ParseForm(); err != nil {
		log.Printf("r.ParseForm() error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp, err := webui.RegisterPOST(ctx, &records.RegisterPOSTRequest{
		Public:  r.FormValue("username"),
		Private: r.FormValue("password"),
	})
	if err != nil {
		log.Printf("webui.RegisterPOST error: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp.ServeHTTP(w, r)
}
