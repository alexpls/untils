package main

import (
	"net/http"

	"github.com/alexpls/untils/internal/components/public"
)

func (a *app) home(w http.ResponseWriter, r *http.Request) {
	public.HomePage().Render(r.Context(), w)
}
