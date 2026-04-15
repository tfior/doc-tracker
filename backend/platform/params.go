package platform

import (
	"net/http"
	"strconv"
)

// ParsePagination reads page and per_page query parameters, applies defaults,
// and writes a 400 response if either value is invalid.
// Returns ok=false if the response has already been written.
func ParsePagination(w http.ResponseWriter, r *http.Request) (page, perPage int, ok bool) {
	page, perPage = 1, 50

	if p := r.URL.Query().Get("page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 {
			Error(w, http.StatusBadRequest, "invalid_input", "invalid page")
			return 0, 0, false
		}
		page = v
	}

	if p := r.URL.Query().Get("per_page"); p != "" {
		v, err := strconv.Atoi(p)
		if err != nil || v < 1 || v > 200 {
			Error(w, http.StatusBadRequest, "invalid_input", "per_page must be between 1 and 200")
			return 0, 0, false
		}
		perPage = v
	}

	return page, perPage, true
}
