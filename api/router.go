package api

import (
	"net/http"
)

func NewRouter(handler *Handler) *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/scan", handler.Scan)
	router.HandleFunc("/scan/", handler.GetScanStatus) // Matches /scan/{id}
	return router
}
