package agents

import (
	"log"
	"time"

	"github.com/elos/data"
	"github.com/elos/models"
	"github.com/elos/models/access"
	"github.com/elos/models/event"
	"github.com/elos/models/tag"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func LocationAgent(ctx context.Context, db data.DB, u *models.User) {
	locTag, err := tag.ForName(db, u, tag.Location)
	if err != nil {
		log.Fatal(err)
	}
	updTag, err := tag.ForName(db, u, tag.Update)
	if err != nil {
		log.Fatal(err)
	}

	// Get the db's changes, then filter by updates, then
	// filter by whether this user can read the record
	changes := data.Filter(data.FilterKind(db.Changes(), models.EventKind), func(c *data.Change) bool {
		ok, _ := access.CanRead(db, u, c.Record)
		if !ok {
			return false
		}

		return event.ContainsTags(c.Record.(*models.Event), locTag, updTag)
	})

Run:
	for {
		select {
		case c, ok := <-*changes:
			if !ok {
				break Run
			}

			locationUpdate(db, u, c.Record.(*models.Event))
		case <-ctx.Done():
			break Run

		}
	}
}

func locationUpdate(db data.DB, u *models.User, e *models.Event) {
	loc, _ := e.Location(db)
	p, err := user.Profile(db, u)
	if err == data.ErrNotFound {
		p = models.NewProfile()
		p.CreatedAt = time.Now()
		p.UpdatedAt = p.CreatedAt
		p.SetID(db.NewID())
		p.SetOwner(u)
	}
	p.SetLocation(loc)
	db.Save(p)
}
