package main

import (
	"log"

	"github.com/elos/models"
)

func main() {
	db, err := models.MongoDB("localhost")
	if err != nil {
		log.Fatal(err)
	}

	db.Query(models.UserKind)
}
