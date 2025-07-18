package main

import (
	"net/http"

	"github.com/cidekar/adele-framework/mux"
	"github.com/go-chi/chi/v5"
)

func (a *application) ApiRoutes() http.Handler {

	r := mux.NewRouter()

	// API Middleware: here is where you can add your Middleware for the API routes. These middleware are
	// called on each API route request.
	r.Route("/api", func(mux chi.Router) {

		// API Routes: here is where you can add your API routes for the application. These
		// routes are loaded by the router.
		// TODO:
		//r.Get("/health", a.Handlers.HealthStatus)
	})

	return r
}
