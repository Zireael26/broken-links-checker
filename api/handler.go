package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Zireael26/broken-links-checker/internal/report"
	"github.com/Zireael26/broken-links-checker/internal/scanner"
)

type Handler struct {
	scanner *scanner.Scanner
	scans   map[string]chan scanner.LinkResult // In-memory scan tracking
}

func NewHandler(s *scanner.Scanner) *Handler {
	return &Handler{
		scanner: s,
		scans:   make(map[string]chan scanner.LinkResult),
	}
}

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL     string `json:"url"`
		Depth   int    `json:"depth,omitempty"`
		Workers int    `json:"workers,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	if req.Depth <= 0 {
		req.Depth = 1 // Default depth
	}
	if req.Workers <= 0 {
		req.Workers = 50 // Default workers
	}

	scanID := generateScanID()
	results := make(chan scanner.LinkResult, 100)
	h.scans[scanID] = results

	go func() {
		defer delete(h.scans, scanID)
		h.scanner.Scan(req.URL, req.Depth, req.Workers, results)
		reportPath := filepath.Join("reports", scanID+".json")
		if err := report.SaveReport(scanID, req.URL, req.Depth, results, reportPath); err != nil {
			// Log error (add logger later)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"scan_id": scanID,
		"status":  "started",
	})
}

func (h *Handler) GetScanStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scanID := strings.TrimPrefix(r.URL.Path, "/scan/")
	if scanID == "" {
		http.Error(w, "Scan ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if _, exists := h.scans[scanID]; exists {
		json.NewEncoder(w).Encode(map[string]string{
			"scan_id": scanID,
			"status":  "running",
		})
		return
	}

	reportPath := filepath.Join("reports", scanID+".json")
	if _, err := http.Dir("reports").Open(reportPath); err == nil {
		json.NewEncoder(w).Encode(map[string]string{
			"scan_id":     scanID,
			"status":      "completed",
			"report_path": reportPath,
		})
	} else {
		http.Error(w, "Scan not found", http.StatusNotFound)
	}
}

func generateScanID() string {
	return time.Now().Format("20060102T150405") // Simple timestamp-based ID
}
