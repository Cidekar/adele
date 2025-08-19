package main

import (
	"myapp/handlers"
	"myapp/middleware"

	"github.com/cidekar/adele-framework"
	"github.com/cidekar/adele-framework/mailer"
)

type application struct {
	App        *adele.Adele
	Handlers   *handlers.Handlers
	Mail       *mailer.Mail
	Middleware *middleware.Middleware
}
