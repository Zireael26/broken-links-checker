package api

import (
	"net/http"
)

func NewRouter(handler *Handler) *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("POST /scan", handler.Scan)
	router.HandleFunc("GET /scan/", handler.GetScanStatus) // Matches /scan/{id}
	return router
}
