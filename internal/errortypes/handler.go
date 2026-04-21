package errortypes

import (
	"errors"
	"net/http"
)

func InternalServerError(w http.ResponseWriter) {
	http.Error(w, "Internal server error", http.StatusInternalServerError)
}

// HandleError responds with a nicely formatted error, if one is given.
// Returns true if an error has been responded to, false otherwise.
func HandleError(err error, w http.ResponseWriter) bool {
	if err == nil {
		return false
	}

	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		http.Error(w, httpErr.HTTPMessage(), httpErr.HTTPCode())
		return true
	}

	InternalServerError(w)

	return true
}
