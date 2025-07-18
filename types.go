package main

import (
	"myapp/handlers"
	"myapp/middleware"

	"github.com/cidekar/adele-framework"
)

type application struct {
	App        *adele.Adele
	Handlers   *handlers.Handlers
	Middleware *middleware.Middleware
}
