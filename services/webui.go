package services

import (
	"github.com/elos/x/models/cal"
	"github.com/elos/x/records"
)

type WebUIClient interface {
	records.WebUIClient
}

type CalWebUIClient interface {
	cal.WebUIClient
}
