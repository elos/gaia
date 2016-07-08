package main

import (
	"log"
	"os"

	"github.com/elos/gaia/routes/records/form"
	"github.com/elos/models"
)

func main() {
	e := new(models.Event)
	bytes, err := form.Marshal(e, "event")
	if err != nil {
		log.Fatalf("form.Marshal error: %v", err)
	}
	if _, err := os.Stdout.Write(bytes); err != nil {
		log.Fatalf("w.Write error: %v", err)
	}
}
