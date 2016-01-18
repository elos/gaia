package routes

import (
	"log"
	"net/http"

	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
)

func CommandSMSPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, sessions services.SMSCommandSessions) {
	m, err := services.Extract(r)

	if err != nil {
		log.Fatal(err)
	}
	sessions.Inbound(m)
}
