package main

import (
	"net/http"

	"github.com/alexpls/untils/internal/errortypes"
)

func (a *app) internalServerError(err error, w http.ResponseWriter) bool {
	if err != nil {
		errortypes.InternalServerError(w)
		return true
	}
	return false
}
