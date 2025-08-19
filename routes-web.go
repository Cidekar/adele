package main

import (
	"fmt"
	"net/http"

	"github.com/cidekar/adele-framework/mux"
	"github.com/go-chi/chi/v5"
)

func (a *application) WebRoutes() http.Handler {

	r := mux.NewRouter()

	// Web Middleware: here is where you can add your Middleware for the web routes.
	// These middleware are called on each web route request.

	r.Use(a.Middleware.NoSurf)

	r.Route("/", func(mux chi.Router) {

		// Web Routes: here is where you can add your web routes for the application. These
		// routes are loaded by the router.

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Welcome to Adele! \n")
		})

	})
	return r
}
