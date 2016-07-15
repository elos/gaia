package routes_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/elos/data/builtin/mem"
	"github.com/elos/gaia/routes"
	"github.com/elos/gaia/services"
	"github.com/elos/models"
	"github.com/elos/models/user"
	"golang.org/x/net/context"
)

func BenchmarkRecordPostEvent(b *testing.B) {
	db := mem.NewDB()
	u, _, err := user.Create(db, "username", "password")
	if err != nil {
		b.Fatalf("user.Create error: %v", err)
	}

	ctx := user.NewContext(context.Background(), u)

	logger := services.NewTestLogger(b)

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		req, err := http.NewRequest(
			"POST",
			"http://www.elos.pw/record/?"+url.Values{
				"kind": []string{models.EventKind.String()},
			}.Encode(),
			bytes.NewBuffer(
				[]byte(`{
				"name": "event name",
				"data": {
					"sensor1": 34,
					"sensor2": 4.3
				},
				"owner_id": "`+u.ID().String()+`"
			}`),
			))
		if err != nil {
			b.Fatalf("http.NewRequest error: %v", err)
		}
		routes.RecordPOST(ctx, rec, req, logger, db)
	}
}
