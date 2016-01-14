package views

import (
	"fmt"

	"github.com/elos/ehttp/templates"
)

type RoutesContext struct {
}

func (r *RoutesContext) Record(kind, id string) string {
	return fmt.Sprintf("/record/%s/%s", kind, id)
}

var routesContext = &RoutesContext{}

type context struct {
	Routes *RoutesContext
	Data   interface{}
}

func (c *context) WithData(d interface{}) templates.Context {
	return &context{
		Routes: c.Routes,
		Data:   d,
	}
}

var globalContext = &context{
	Routes: routesContext,
}
