package routes

import (
	"net/http"

	"github.com/elos/gaia/routes/records"
	"github.com/elos/gaia/services"
	"golang.org/x/net/context"
)

// Use the Records var to dispatch http requests to the various /records/... subroutes.
// e.g.,
//		routes.Records.CreateGET(ctx, w, r, db, logger)
var Records = struct {
	QueryGET   func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	NewGET     func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	CreateGET  func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	CreatePOST func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	EditGET    func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	EditPOST   func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	ViewGET    func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
	DeletePOST func(context.Context, http.ResponseWriter, *http.Request, services.WebUIClient)
}{
	QueryGET:   records.QueryGET,
	NewGET:     records.NewGET,
	CreateGET:  records.CreateGET,
	CreatePOST: records.CreatePOST,
	EditGET:    records.EditGET,
	EditPOST:   records.EditPOST,
	ViewGET:    records.ViewGET,
	DeletePOST: records.DeletePOST,
}
