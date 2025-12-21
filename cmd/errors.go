package main

import "net/http"

func (a *app) internalServerError(err error, w http.ResponseWriter) bool {
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return true
	}
	return false
}

func (a *app) badRequest(err error, w http.ResponseWriter) bool {
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return true
	}
	return false
}

func (a *app) notFound(w http.ResponseWriter) {
	http.Error(w, "Not found", http.StatusNotFound)
}
