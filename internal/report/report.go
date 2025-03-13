package report

import (
	"encoding/json"
	"os"
	"time"

	"github.com/Zireael26/broken-links-checker/internal/scanner"
)

type Report struct {
    ScanID       string             `json:"scan_id"`
    StartURL     string             `json:"start_url"`
    Timestamp    string             `json:"timestamp"`
    Depth        int                `json:"depth"`
    TotalLinks   int                `json:"total_links"`
    WorkingLinks []scanner.LinkResult `json:"working_links"`
    BrokenLinks  []scanner.LinkResult `json:"broken_links"`
}

func SaveReport(scanID, startURL string, depth int, results <-chan scanner.LinkResult, path string) error {
    var working, broken []scanner.LinkResult
    for result := range results {
		// Accept all 2xx codes as working
        if result.Code % 200 < 100 {
            working = append(working, result)
        } else {
            broken = append(broken, result)
        }
    }

    report := Report{
        ScanID:       scanID,
        StartURL:     startURL,
        Timestamp:    time.Now().Format(time.RFC3339),
        Depth:        depth,
        TotalLinks:   len(working) + len(broken),
        WorkingLinks: working,
        BrokenLinks:  broken,
    }

    file, err := os.Create(path)
    if err != nil {
        return err
    }
    defer file.Close()

    enc := json.NewEncoder(file)
    enc.SetIndent("", "  ")
    return enc.Encode(report)
}