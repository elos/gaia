package routes

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/elos/data"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/location"
	"github.com/elos/models/tag"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

const (
	altitudeParam  = "altitude"
	latitudeParam  = "latitude"
	longitudeParam = "longitude"
)

func MobileLocationPOST(ctx context.Context, w http.ResponseWriter, r *http.Request, l services.Logger, db data.DB) {
	// Parse the form value
	if err := r.ParseForm(); err != nil {
		l.Printf("MobileLocationPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Altitude
	alt := r.FormValue(altitudeParam)
	if alt == "" {
		http.Error(w, "You must specify an altitude", http.StatusBadRequest)
		return
	}
	altitude, err := strconv.ParseFloat(alt, 64)
	if err != nil {
		http.Error(w, "Parsing altitude", http.StatusBadRequest)
		return
	}

	// Latitude
	lat := r.FormValue(latitudeParam)
	if lat == "" {
		http.Error(w, "You must specify a latitude", http.StatusBadRequest)
		return
	}
	latitude, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		http.Error(w, "Parsing latitude", http.StatusBadRequest)
		return
	}

	// Longitude
	lon := r.FormValue(longitudeParam)
	if lon == "" {
		http.Error(w, "You must specify an longitude", http.StatusBadRequest)
		return
	}
	longitude, err := strconv.ParseFloat(alt, 64)
	if err != nil {
		http.Error(w, "Parsing longitude", http.StatusBadRequest)
		return
	}

	// Retrieve the user this request was authenticated as
	u, ok := user.FromContext(ctx)
	if !ok { // This is certainly an issue, and should _never_ happen
		l.Print("MobileLocationPOST Error: failed to retrieve user from context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Create location
	loc := location.NewCoords(altitude, latitude, longitude)
	loc.SetID(db.NewID())
	loc.SetOwner(u)

	now := loc.CreatedAt

	e := models.NewEvent()
	e.CreatedAt = now
	e.SetID(db.NewID())
	e.SetOwner(u)
	e.Name = "Location Update"
	e.SetLocation(loc)
	e.Time = now
	e.UpdatedAt = now

	locationTag, err1 := tag.ForName(db, u, tag.Location)
	updateTag, err2 := tag.ForName(db, u, tag.Update)
	mobileTag, err3 := tag.ForName(db, u, tag.Mobile)
	if err1 != nil || err2 != nil || err3 != nil {
		l.Printf("MobileLocationPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	e.IncludeTag(locationTag)
	e.IncludeTag(updateTag)
	e.IncludeTag(mobileTag)

	if err = db.Save(loc); err != nil {
		l.Printf("MobileLocationPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err = db.Save(e); err != nil {
		l.Printf("MobileLocationPOST Error: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	bytes, err := json.MarshalIndent(e, "", "	")
	if err != nil {
		l.Printf("MobileLocationPOST Error: while marshalling json %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
