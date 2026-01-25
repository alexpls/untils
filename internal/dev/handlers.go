package dev

import (
	"net/http"

	"github.com/alexpls/untils/internal/models"
)

type Handlers struct {
}

func NewHandlers() *Handlers {
	return &Handlers{}
}

func (h *Handlers) PaletteGet(w http.ResponseWriter, r *http.Request, _ *models.User) {
	component := PalettePage()
	component.Render(r.Context(), w)
}
