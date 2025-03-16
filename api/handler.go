package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
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

	if req.Depth < 0 {
		req.Depth = 0 // Default depth
	}

	if req.Depth > 2 {
		req.Depth = 2 // Limit depth to 2
	}

	if req.Workers <= 0 {
		req.Workers = 50 // Default workers
	}

	scanID := generateScanID()
	results := make(chan scanner.LinkResult, 100)
	h.scans[scanID] = results

	var swg sync.WaitGroup
	swg.Add(1)
	go func() {
		log.Printf("Starting scan %s for %s with depth %d", scanID, req.URL, req.Depth)
		h.scanner.Scan(req.URL, req.Depth, results, &swg)
		reportPath := filepath.Join("reports", scanID+".json")
		
		swg.Wait()
		close(results)
		log.Printf("Scan %s completed, len(results): %d", scanID, len(results))
		log.Printf("Saving report to %s", reportPath)
		if err := report.SaveReport(scanID, req.URL, req.Depth, results, reportPath); err != nil {
			log.Printf("Failed to save report for %s: %v", scanID, err)
		}
		log.Printf("Report saved to %s", reportPath)
		defer delete(h.scans, scanID)
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
