package main

import "net/http"

func (a *app) internalServerError(err error, w http.ResponseWriter) bool {
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return true
	}
	return false
}
