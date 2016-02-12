package test

import (
	"log"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/elos/data"
	"github.com/elos/data/transfer"
	"github.com/elos/gaia/routes"
	"github.com/elos/models"
	"github.com/elos/models/event"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

// currently blocks forever?
func TestLocationAgent(t *testing.T) {
	t.Skip()
	db, _, s := testInstance(t, context.Background())
	defer s.Close()
	u, cred := testUser(t, db)

	t.Log("Opening websocket")
	serverURL := s.URL
	origin := serverURL
	wsURL := strings.Replace(serverURL, "http", "ws", 1)
	params := url.Values{}
	params.Set("public", cred.Public)
	params.Set("private", cred.Private)
	params.Set("kind", models.ProfileKind.String())
	wsURL += routes.RecordChanges + "?" + params.Encode()
	ws, err := websocket.Dial(wsURL, "", origin)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close()
	t.Log("Websocket openened")

	t.Log("Creating new location update")
	var loc *models.Location
	if _, loc, err = event.LocationUpdate(db, u, 50, 50, 50); err != nil {
		t.Fatal(err)
	}
	t.Log("Created location update")

	time.Sleep(500 * time.Millisecond)

	log.Print("h")

	var ct transfer.ChangeTransport
	if err := websocket.JSON.Receive(ws, &ct); err != nil {
		t.Fatal(err)
	}

	log.Print("h")

	tc := transfer.ChangeFrom(&ct, models.ModelFor(ct.RecordKind))

	if tc.ChangeKind != data.Update {
		t.Fatal("Expected ChangeKind to be Update")
	}

	if tc.Record.(*models.Profile).LocationId != loc.Id {
		t.Fatal("Expected profile to have new location's id")
	}
}
