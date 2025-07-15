package main

import (
	"log"
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

	app := &application{
		App: a,
	}

	app.App.Routes = app.routes()

	return app
}
