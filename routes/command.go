package routes

import (
	"net/http"

	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
)

func CommandSMSPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, sessions services.SMSCommandSessions) {
	m, err := services.Extract(r)

	if err != nil {
		logger.Fatal(err)
	}

	sessions.Inbound(m)
}
