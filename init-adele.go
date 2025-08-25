package main

import (
	"log"
	"myapp/handlers"
	"myapp/middleware"
	"myapp/models"
	"os"

	"github.com/cidekar/adele-framework"
)

func initApplication() *application {
	path, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	a := &adele.Adele{}
	err = a.New(path)
	if err != nil {
		log.Fatal(err)
	}

	a.AppName = "myapp"

	myMiddleware := &middleware.Middleware{
		App: a,
	}

	myHandlers := &handlers.Handlers{
		App: a,
	}

	app := &application{
		App:        a,
		Handlers:   myHandlers,
		Mail:       &a.Mail,
		Middleware: myMiddleware,
	}

	app.App.Routes = app.routes()

	app.Models = models.New(a)

	return app
}
