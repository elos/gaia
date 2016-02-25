package agents_test

import (
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/agents"
	"github.com/elos/models"
	"github.com/elos/models/event"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func TestLocationAgent(t *testing.T) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatal(err)
	}

	changes := data.FilterKind(db.Changes(), models.ProfileKind)

	ctx, stop := context.WithCancel(context.Background())
	go agents.LocationAgent(ctx, db, u)
	defer stop()

	// give control to agent thread
	time.Sleep(1 * time.Millisecond)

	_, loc, err := event.LocationUpdate(db, u, 50, 50, 50)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case profileChange := <-*changes:
		p := profileChange.Record.(*models.Profile)

		if loc.Id != p.LocationId {
			t.Fatal("Expected profile's location id to now match be the new location ")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for profile update")
	}
}
