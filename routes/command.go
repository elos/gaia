package routes

import (
	"net/http"

	"github.com/elos/gaia/services"
	"github.com/elos/gaia/services/sms"
	"golang.org/x/net/context"
)

// Expects: From, To, Body params
func CommandSMSPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, logger services.Logger, sessions services.SMSCommandSessions) {
	m, err := sms.ExtractMessageFromRequest(r)

	if err != nil {
		logger.Fatal(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	sessions.Inbound(m)

	w.WriteHeader(http.StatusNoContent)
}
