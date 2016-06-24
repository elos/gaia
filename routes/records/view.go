package records

import (
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
)

func ViewGET(ctx context.Context, w http.ResponseWriter, r *http.Request, db data.DB, logger services.Logger) {
}
