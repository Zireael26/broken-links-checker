package api

import (
	"net/http"
)

func NewRouter(handler *Handler) *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("POST /scan", handler.Scan)
	router.HandleFunc("GET /scan/", handler.GetScanResults) // Updated to use enhanced results endpoint
	return router
}
