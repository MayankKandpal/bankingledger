package handler

import (
	"net/http"
	"strconv"
)

// parseLimitOffset reads ?limit= and ?offset= query params with safe defaults.
// limit is clamped to [1, maxLimit]; offset is clamped to >= 0.
func parseLimitOffset(r *http.Request, defaultLimit, maxLimit int) (int, int) {
	limit := defaultLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if limit < 1 {
		limit = defaultLimit
	}

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}
