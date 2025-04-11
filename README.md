# NexusProbe

![NexusProbe Logo](https://via.placeholder.com/150?text=NexusProbe) <!-- Replace with your logo URL -->

**Explore the web's nexus, detect every flaw.**

NexusProbe is an API-driven Broken Link Checker built in Go, designed to scan websites for broken links with speed, efficiency, and precision. Using a 3-tier hybrid approach—static parsing, crawler spoofing, and full rendering—it tackles everything from WordPress blogs to modern SPAs, delivering comprehensive results in a structured JSON report. Whether you're a developer, webmaster, or SEO enthusiast, NexusProbe keeps your site's links in check.

---

## Features

- **API-Based**: Start scans and retrieve results via RESTful endpoints (`POST /scan`, `GET /scan/{id}`).
- **Hybrid Scanning with Intelligent Escalation**:
  - **Tier 1: Static** (~10-50ms/page): Fast parsing with `goquery` for simple HTML sites.
  - **Tier 2: Spoofed** (~200-500ms/page): Googlebot spoofing with `go-rod` for prerendered MPAs.
  - **Tier 3: Rendered** (~500-1000ms/page): Full DOM rendering with `go-rod` for SPAs.
  - **Smart Escalation**: Automatically switches between tiers based on site complexity and detected links.
- **High Performance**:
  - **Concurrency**: Go goroutines with a configurable worker pool (default: 50) for high throughput.
  - **Resource Efficiency**: Uses the lightest approach needed for each page to optimize speed.
  - **Throughput**: ~20-100 pages/sec for static sites, ~2-5 pages/sec for spoofed, ~1-2 pages/sec for rendered.
- **Advanced Configuration**:
  - **Rate Limiting**: Configurable requests-per-second to avoid overloading target servers.
  - **Domain Filtering**: Option to limit scanning to the starting domain only.
  - **Site Type Override**: Force a specific tier (static, spoofed, rendered) when needed.
  - **Retry Mechanism**: Configurable retry count and delay for transient network errors.
- **Comprehensive Reports**:
  - **JSON Structure**: Clean organization with `working_links` and `broken_links` sections.
  - **Performance Metrics**: Detailed statistics about scan efficiency and escalation patterns.
  - **Status Tracking**: Real-time status updates via API.
- **Deployment Options**:
  - **Local**: Run with a local Chrome/Chromium installation.
  - **Docker**: Self-sufficient containerized deployment with bundled Chrome.
  - **Docker Compose**: Easy orchestration with volume mapping for reports.
- **Sci-Fi Flair**: Inspired by a probe scanning the web's nexus, built with a futuristic vibe.

---

## Installation

### Prerequisites

- **Go**: 1.22 or later ([install](https://golang.org/doc/install)).
- **Chrome/Chromium**: Required for `go-rod` rendering ([download](https://www.google.com/chrome/)).

### Local Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/<yourusername>/nexusprobe.git
   cd nexusprobe
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

### Docker Setup

1. Build and run with Docker:
   ```bash
   docker build -t nexusprobe .
   docker run -p 8080:8080 -v $(pwd)/reports:/app/reports nexusprobe
   ```

2. Or use Docker Compose:
   ```bash
   docker-compose up -d
   ```

---

## Usage

### Start a Scan

Send a POST request to initiate a scan:

```bash
curl -X POST -H "Content-Type: application/json" \
     -d '{
       "url": "https://example.com", 
       "depth": 2, 
       "workers": 50,
       "rate_limit": 10,
       "domain_only": true,
       "site_type": "",
       "retries": 2,
       "retry_delay": 1000
     }' \
     http://localhost:8080/scan
```

#### Request Parameters

- **url** (required): The starting URL to scan.
- **depth** (optional): Maximum crawl depth (default: 2, max: 3).
- **workers** (optional): Number of concurrent workers (default: 50, max: 100).
- **rate_limit** (optional): Maximum requests per second (0 for unlimited).
- **domain_only** (optional): Limit scan to the same domain as the starting URL.
- **site_type** (optional): Force a specific scanning tier ("static", "spoofed", or "rendered").
- **retries** (optional): Number of retry attempts for failed requests (default: 2).
- **retry_delay** (optional): Delay between retries in milliseconds (default: 1000).

#### Response

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

#### Running Response

```json
{
  "scan_id": "20250312T100000",
  "status": "running"
}
```

#### Completed Response

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
  "duration": "1m45.234s",
  "working_links": [
    {
      "url": "https://example.com/about",
      "status": "OK",
      "code": 200,
      "depth": 1,
      "tier": "static"
    },
    {
      "url": "https://example.com/contact",
      "status": "OK",
      "code": 200,
      "depth": 1,
      "tier": "spoofed"
    }
  ],
  "broken_links": [
    {
      "url": "https://example.com/bad",
      "status": "Broken",
      "code": 404,
      "depth": 1,
      "tier": "static"
    },
    {
      "url": "https://example.com/oops",
      "status": "Error",
      "code": 0,
      "depth": 2,
      "tier": "rendered"
    }
  ],
  "performance": {
    "static_scans": 450,
    "spoofed_scans": 23,
    "rendered_scans": 5,
    "escalations_to_spoofed": 12,
    "escalations_to_rendered": 3,
    "timeout_errors": 2,
    "network_errors": 1,
    "retry_attempts": 3,
    "successful_retries": 2
  }
}
```

---

## How It Works

### 3-Tier Hybrid Scanning

NexusProbe uses an intelligent, progressive escalation approach to balance speed and coverage:

1. **Tier 1: Static Parsing**
   - Uses `goquery` to extract links from HTTP responses
   - Fastest method (~10-50ms per page)
   - Great for static websites and blogs
   - ~50-70% link coverage

2. **Tier 2: Crawler Spoofing**
   - Uses `go-rod` with Googlebot user-agent
   - Captures prerendered content for SEO-optimized sites
   - Medium speed (~200-500ms per page)
   - ~70-85% link coverage

3. **Tier 3: Full Rendering**
   - Uses `go-rod` to fully render pages with JavaScript
   - Captures dynamically generated links in SPAs
   - Slowest method (~500-1000ms per page)
   - ~90-95% link coverage

### Escalation Logic

NexusProbe automatically escalates through tiers when:
- Fewer than 5 links are found in the current tier
- The URL matches patterns typical of SPAs or dynamic content
- The page has mostly empty content but includes script tags

This intelligent approach ensures:
- Maximum speed for simple pages
- Appropriate rendering for complex pages
- Optimal balance between thoroughness and resource usage

---

## Contributing

We welcome contributions to NexusProbe! To get started:

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

NexusProbe is licensed under the MIT License (LICENSE). Feel free to use, modify, and distribute it as you see fit.

---

## About

Built as a portfolio project to showcase Go's concurrency, API design, and hybrid web scanning. Inspired by sci-fi probes exploring vast networks, NexusProbe blends efficiency with a touch of futuristic flair.

Questions? Issues? Open an issue or reach out!

Happy scanning!
