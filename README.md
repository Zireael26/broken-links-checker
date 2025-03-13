# Broken Links Checker

**Explore the web’s nexus, detect every flaw.**

BLC is an API-driven Broken Link Checker built in Go, designed to scan websites for broken links with speed, efficiency, and precision. Using a 3-tier hybrid approach—static parsing, crawler spoofing, and full rendering—it tackles everything from WordPress blogs to modern SPAs, delivering comprehensive results in a structured JSON report. Whether you’re a developer, webmaster, or SEO enthusiast, BLC keeps your site’s links in check.

---

## Features

- **API-Based**: Start scans and retrieve results via RESTful endpoints (`POST /scan`, `GET /scan/{id}`).
- **Hybrid Scanning**:
  - **Static**: Fast parsing with `goquery` for simple HTML sites.
  - **Spoofed**: Googlebot spoofing with `go-rod` for prerendered MPAs.
  - **Rendered**: Full DOM rendering with `go-rod` for SPAs.
- **Concurrency**: Go goroutines with a configurable worker pool (default: 50) for high throughput.
- **JSON Reports**: Structured output with `working_links` and `broken_links` sections, saved to `reports/`.
- **Depth Control**: Scan up to a configurable depth (default: 2).
- **Sci-Fi Flair**: Inspired by a probe scanning the web’s nexus, built with a futuristic vibe.

---

## Installation

### Prerequisites

- **Go**: 1.22 or later ([install](https://golang.org/doc/install)).
- **Chrome/Chromium**: Required for `go-rod` rendering ([download](https://www.google.com/chrome/)).

### Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/<yourusername>/broken-links-checker.git
   cd broken-links-checker
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Build and run:
   ```bash
   go run cmd/blc/main.go
   ```
   The API will start on `http://localhost:8080`.

---

## Usage

### Start a Scan

Send a POST request to initiate a scan:

```bash
curl -X POST -H "Content-Type: application/json" \
     -d '{"url": "https://example.com", "depth": 2, "workers": 50}' \
     http://localhost:8080/scan
```

Response:

```json
{
  "scan_id": "20250312T100000",
  "status": "started"
}
```

### Check Scan Status

Poll the scan status with the `scan_id`:

```bash
curl http://localhost:8080/scan/20250312T100000
```

Running Response:

```json
{
  "scan_id": "20250312T100000",
  "status": "running"
}
```

Completed Response:

```json
{
  "scan_id": "20250312T100000",
  "status": "completed",
  "report_path": "reports/20250312T100000.json"
}
```

### Sample Report

The JSON report in `reports/<scan_id>.json` looks like:

```json
{
  "scan_id": "20250312T100000",
  "start_url": "https://example.com",
  "timestamp": "2025-03-12T10:00:00Z",
  "depth": 2,
  "total_links": 6,
  "working_links": [
    {
      "url": "https://example.com/about",
      "status": "OK",
      "code": 200,
      "depth": 1
    },
    {
      "url": "https://example.com/contact",
      "status": "OK",
      "code": 200,
      "depth": 1
    }
  ],
  "broken_links": [
    {
      "url": "https://example.com/bad",
      "status": "Broken",
      "code": 404,
      "depth": 1
    },
    {
      "url": "https://example.com/oops",
      "status": "Error",
      "code": 0,
      "depth": 2
    }
  ]
}
```

---

## Contributing

We welcome contributions to BLC! To get started:

1. Fork the repository.
2. Create a branch: `git checkout -b feature/your-feature`.
3. Commit your changes: `git commit -m "Add your feature"`.
4. Push to your fork: `git push origin feature/your-feature`.
5. Open a Pull Request.

Please follow these guidelines:

- Use `go fmt` and `golangci-lint` for code style.
- Add tests for new features in `internal/<package>/*_test.go`.
- Update this README if you change usage or features.

---

## License

BLC is licensed under the GNU General Public License v3. You are free to use, modify, and distribute it, provided that any modifications are also licensed under the GPL v3. For full terms, see the LICENSE file.

---

## About

Built as a portfolio project to showcase Go’s concurrency, API design, and hybrid web scanning. Inspired by sci-fi probes exploring vast networks, BLC blends efficiency with a touch of futuristic flair.

Questions? Issues? Open an issue or reach out!

Happy scanning!
