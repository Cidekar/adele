package main

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cidekar/adele-framework/mux"
)

func (a *application) routes() *mux.Mux {
	fileServer := http.FileServer(http.Dir("./public"))

	// Wrapper function to clean the path and check for traveral attempts
	// with the aim to block any path containing traversal characters.
	// Example:
	//   Blocks any .. path traversal attempts
	//   Blocks /public/../../../etc/passwd
	//   Blocks /public/../.env
	secureFileServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(r.URL.Path)

		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
	a.App.Routes.Method("Get", "/public/*", http.StripPrefix("/public", secureFileServer))
	a.App.Routes.Mount("/", a.WebRoutes())
	a.App.Routes.Mount("/api", a.ApiRoutes())
	return a.App.Routes
}
