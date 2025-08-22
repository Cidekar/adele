package handlers

import (
	"net/http"

	"github.com/cidekar/adele-framework"
)

type Handlers struct {
	App *adele.Adele
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	err := h.App.Helpers.Render(w, r, "home", nil, nil)
	if err != nil {
		h.App.Log.Error("error rendering:", err)
	}
}

func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	err := h.App.Helpers.Render(w, r, "404", nil, nil)
	if err != nil {
		h.App.Log.Error("error rendering:", err)
	}
}
