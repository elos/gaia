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

func TestWebSensorsAgent(t *testing.T) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		t.Fatal(err)
	}

	changes := data.FilterKind(db.Changes(), models.EventKind)

	ctx, stop := context.WithCancel(context.Background())
	go agents.WebSensorsAgent(ctx, db, u)
	defer stop()

	// give control to agent thread
	time.Sleep(1 * time.Millisecond)

	_, err = event.WebSensorLocation(db, u, 50, 50)
	if err != nil {
		t.Fatal(err)
	}

	// read off the event we just created
	<-*changes

	select {
	case eventChange := <-*changes:
		e := eventChange.Record.(*models.Event)

		if e.Name != "Location Update" {
			t.Fatal("Expected a location update to be produced")
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timed out waiting for event creation")
	}
}
