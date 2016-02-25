package agents

import (
	"log"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/event"
	"github.com/elos/models/tag"
	"golang.org/x/net/context"
)

const (
	WEB_SENSOR_LOCATION = "WEB_SENSOR_LOCATION"
)

func WebSensorsAgent(ctx context.Context, db data.DB, u *models.User) {
	// Get the db's changes, then filter by updates, then
	// filter by whether this user can read the record
	changes := data.Filter(data.FilterKind(db.Changes(), models.EventKind), func(c *data.Change) bool {
		ok, _ := access.CanRead(db, u, c.Record)
		return ok
	})

Run:
	for {
		select {
		case c, ok := <-*changes:
			if !ok {
				break Run
			}

			switch c.Record.(*models.Event).Name {
			case WEB_SENSOR_LOCATION:
				webSensorLocation(db, u, c.Record.(*models.Event).Data)
			}
		case <-ctx.Done():
			break Run

		}
	}
}

func webSensorLocation(db data.DB, u *models.User, eventData map[string]interface{}) {
	log.Printf("%++v", eventData)
	webTag, err := tag.ForName(db, u, "WEB")
	if err != nil {
		log.Fatal(err)
	}

	lat, ok := eventData["latitude"].(float64)
	if !ok {
		return // bail
	}

	lon, ok := eventData["longitude"].(float64)
	if !ok {
		return // bail
	}

	_, _, err = event.LocationUpdate(
		db,
		u,
		0,
		lat,
		lon,
		webTag,
	)

	if err != nil {
		log.Fatal(err)
	}
}
