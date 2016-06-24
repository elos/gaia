package routes

import (
	"net/http"

	"github.com/elos/data"
	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
)

// Use the Records var to dispatch http requests to the various /records/... subroutes.
// e.g.,
//		routes.Records.CreateGET(ctx, w, r, db, logger)
var Records = struct {
	QueryGET   func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	NewGET     func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	NewPOST    func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	CreateGET  func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	CreatePOST func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	EditGET    func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	EditPOST   func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	ViewGET    func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
	DeletePOST func(context.Context, http.ResponseWriter, *http.Request, data.DB, services.Logger)
}{
	QueryGET:   records.QueryGET,
	NewGET:     records.NewGET,
	NewPOST:    records.NewPOST,
	CreateGET:  records.CreateGET,
	CreatePOST: records.CreatePOST,
	EditGET:    records.EditGET,
	EditPOST:   records.EditPOST,
	ViewGET:    records.ViewGET,
	DeletePOST: records.DeletePOST,
}
