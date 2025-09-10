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
	secure, err := strconv.ParseBool(os.Getenv("COOKIE_SECURE"))

	if err != nil {
		secure = true
		a.App.Log.Warn("cookie secure setting not recognized—CSRF token defaulted to secure")
	}

	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
		Domain:   os.Getenv("COOKIE_DOMAIN"),
	})

	return csrfHandler
}
