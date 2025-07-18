package main

import (
	"net/http"

	"github.com/cidekar/adele-framework/mux"
)

func (a *application) routes() *mux.Mux {
	fileServer := http.FileServer(http.Dir("./public"))
	a.App.Routes.Method("Get", "/public/*", http.StripPrefix("/public", fileServer))
	a.App.Routes.Mount("/", a.WebRoutes())
	a.App.Routes.Mount("/api", a.ApiRoutes())
	return a.App.Routes
}
