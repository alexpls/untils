package main

import (
	"net/http"

	appcomponents "github.com/alexpls/untils/internal/components/app"
	"github.com/alexpls/untils/internal/db/sqlc"
)

func (a *app) dashboardGet(w http.ResponseWriter, r *http.Request, u *sqlc.User) {
	appcomponents.DashboardPage().Render(r.Context(), w)
}
