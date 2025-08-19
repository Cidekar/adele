package middleware

import (
	"net/http"
	"os"
	"strconv"

	"github.com/justinas/nosurf"
)

// Setup and return CSRF token setup
func (a *Middleware) NoSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	secure, _ := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Domain:   os.Getenv("COOKIE_DOMAIN"),
	})

	return csrfHandler
}
