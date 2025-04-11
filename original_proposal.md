![][image1]

# Broken Links Finder with Go

## **Multi-threaded, Concurrent and High-throughput**

# **OVERVIEW**

The idea is to build an open source application with an API that utilises the best features Go has to offer to build a high-throughput, highly optimised broken links finder that finds the perfect balance between speed, resource utilisation, and coverage.

In my previous experience with writing a broken links finder, which was built on top of another open-source project, I ran into one major problem, it only worked on static / blog / wordpress sites. Support for dynamic, more modern websites was neither present in the source project, nor implemented over it.

# **GOALS**

1. Build an API to accept requests.  
2. Utilise goroutines tastefully to write a high-throughput broken links finder.  
3. Make it customisable and also idiot-proof.  
4. Should work on all or most types of websites.  
5. Kickass readme and docs.  
6. Multiple types of deployment options. (local with chromel, self-sufficient containerized, containerized with external headless browser / grid support)

# **Motivation**

1. Will serve as an excellent portfolio project.  
2. Researched similar existing projects and nothing similar exists, there are general purpose crawlers and scrapers, but most of them are no longer maintained.

# **SPECIFICATIONS**

In the interest of a good balance between coverage, speed, throughput, resource utilisation and light footprint, there will be no one-size-fits-all approach here. Why? Simply because support for modern, dynamic websites will require we render them before parsing the links present in them. And while using a headless browser will support all (read most) websites, it is a huge overkill for a large chunk of static websites. What about dynamic websites that are optimised for crawlers/SEO? 

A multi-tier hybrid approach is what will solve the problem.

## **How? Here’s how\!**

Tier 1: Use goquery to prase the static / blog / wordpress sites and find all links present in them.

Tier 2: Use go-rod (a lighter alternative to chromedp) and spoof a Google Crawler when sending requests to fetch pre-rendered HTML pages, if the website is SEO/crawler optimized)

Tier 3: Use an optimised and lightweight go-rod instance to render the website/SPA and parse the links.

## **Why three separate tiers?**

**Tabulated Comparison of the Three Tiers**

| Metric | Tier 1: Goquery (Static) | Tier 2: Go-Rod Spoofing (Prerendered) | Tier 3: Go-Rod Rendering (Dynamic) |
| :---- | :---- | :---- | :---- |
| **Description** | Parses raw HTML from http.Get using goquery. | Uses go-rod with Googlebot UA to fetch prerendered HTML. | Uses go-rod to fully render the page and extract DOM links. |
| **Target Sites** | Static sites (e.g., WordPress, plain HTML). | SEO-optimized MPAs (e.g., Next.js, Nuxt.js). | SPAs or JS-heavy sites (e.g., React, Angular). |
| **Speed (per page)** | \~10-50 ms (parsing only). | \~200-500 ms (fetch \+ minimal rendering). | \~500-1000 ms (full JS execution \+ rendering). |
| **Throughput** | \~20-100 pages/sec (50 workers). | \~2-5 pages/sec (50 workers, Chrome-limited). | \~1-2 pages/sec (50 workers, Chrome-limited). |
| **Memory Usage** | \~1-5 MB (Go runtime \+ HTML buffer). | \~50-150 MB (shared Chrome instance). | \~100-200 MB (shared Chrome instance, peak). |
| **CPU Usage** | Negligible (\~1-5% per worker, I/O-bound). | Low (\~10-20% per page, fetch \+ prerendering). | Moderate (\~20-50% per page, JS execution). |
| **Coverage** | \~50-70% (static links only). | \~70-85% (static \+ prerendered links). | \~90-95% (all links, static \+ dynamic). |
| **Concurrency Fit** | Excellent (lightweight goroutines). | Good (Chrome handles \~10-20 pages). | Good (Chrome limits parallelism). |
| **Resource Efficiency** | Excellent (\<10 MB total). | Moderate (\~100-250 MB total). | Moderate (\~100-300 MB total). |
| **Escalation Trigger** | \<5 links, \<script\> tags, sparse DOM. | \<5 links after spoofing. | Final fallback for maximum coverage. |
| **Pros** | Fastest, lightest, no Chrome dependency. | Balances speed and coverage, leverages SEO. | Highest coverage, handles all modern sites. |
| **Cons** | Misses dynamic links (SPAs). | Misses non-SEO SPAs, still uses Chrome. | Slowest, highest resource use per page. |

The sections below contain an AI generated (Grok 3 Beta) requirements analysis doc. The same was prepared after brainstorming about my vision for the project for a while with the same AI model.

## 

## 

## 

## 

## 

## 

## 

## 

## 

## 

## **Requirements Analysis Document: Broken Link Checker**

# **1\. Project Overview**

## **1.1 Purpose**

The Broken Link Checker is an API-based tool designed to scan websites for broken links (e.g., 404s, timeouts) up to a specified depth, leveraging Go’s concurrency for speed and a hybrid approach to balance resource usage and link coverage. It targets diverse website types—static blogs (e.g., WordPress), crawler-optimized multi-page apps (MPAs), and single-page apps (SPAs)—and produces a structured JSON report containing all scanned links, categorized into working and broken sections for easy parsing and access.

**1.2 Scope**

* **Input:** API requests with a starting URL and optional parameters (depth, workers).  
* **Output:** A JSON file containing a scan report with all links (working and broken), their status, depth, and detection method.  
* **Functionality:**  
  * Scan links up to depth 2 (configurable via API).  
  * Use a 3-tier hybrid approach: static parsing, crawler spoofing, and full rendering.  
  * Support modern websites (static, prerendered, dynamic).  
  * Expose RESTful APIs for initiating scans and retrieving results.  
* **Platforms**: Cross-platform (Windows, macOS, Linux), server-based with HTTP endpoints.  
* **Dependencies:** Go, goquery, go-rod, local Chrome/Chromium installation.

---

**2\. Stakeholders**

* **Developer:** You, building a portfolio project showcasing Go’s concurrency, API design, and practical web scanning.  
* **Users:** Developers, webmasters, or applications integrating the tool via APIs to identify broken links programmatically.  
* **xAI:** Providing Grok to assist in design and implementation.

---

**3\. Functional Requirements**

**3.1 Core Features**

1. **API Endpoints:**  
   * **POST /scan:** Initiate a link scan.  
     * Request Body: { "url": "https://example.com", "depth": 2, "workers": 50 }  
     * Response: { "scan\_id": "abc123", "status": "started" }  
   * **GET /scan/{scan\_id}:** Retrieve scan status and results.  
     * Response (in progress): { "scan\_id": "abc123", "status": "running" }  
     * Response (complete): { "scan\_id": "abc123", "status": "completed", "report\_path": "reports/abc123.json" }  
2. **Link Scanning:**  
   * Accept a starting URL via the /scan API.  
   * Crawl links up to a configurable depth (default: 2).  
   * Check each link’s HTTP status (e.g., 200 OK, 404 Not Found).  
3. **Hybrid Approach:**  
   * **Tier 1: Static Parsing:**  
     * Use goquery to parse raw HTML from http.Get.  
     * Target static sites (e.g., WordPress, plain HTML).  
   * **Tier 2: Crawler Spoofing:**  
     * Use go-rod with Googlebot User-Agent to fetch prerendered HTML.  
     * Target SEO-optimized MPAs (e.g., Next.js).  
   * **Tier 3: Full Rendering:**  
     * Use go-rod to render pages and extract links from the final DOM.  
     * Target SPAs (e.g., React, Angular).  
4. **Escalation Logic:**  
   * Escalate from Tier 1 to Tier 2 if \<5 links are found, \<script\> tags are present, or DOM is sparse (e.g., empty \<body\>).  
   * Escalate from Tier 2 to Tier 3 if \<5 links persist after spoofing.  
5. **Concurrency:**  
   * Use Go goroutines with a configurable worker pool (default: 50\) to process links in parallel.  
   * Manage tasks via a buffered channel for smooth throughput.  
6. **Scan Report:**  
   * Save results as a JSON file (e.g., reports/\<scan\_id\>.json).  
   * Ensure working\_links and broken\_links sections are distinct and easily parseable.

`{`

  `"scan_id": "abc123",`

  `"start_url": "https://example.com",`

  `"timestamp": "2025-03-12T10:00:00Z",`

  `"depth": 2,`

  `"total_links": 5000,`

  `"working_links": [`

    `{`

      `"url": "https://example.com/about",`

      `"status": "OK",`

      `"code": 200,`

      `"depth": 1,`

      `"tier": "static"`

    `}`

  `],`

  `"broken_links": [`

    `{`

      `"url": "https://example.com/bad",`

      `"status": "Broken",`

      `"code": 404,`

      `"depth": 1,`

      `"tier": "static"`

    `}`

  `]`

`}`

**3.2 Optional Features**

1. **Site Type Override:**  
   * Accept an optional site\_type in /scan (e.g., "blog", "mpa", "spa") to bypass hybrid escalation.  
2. **Rate Limiting:**  
   * Accept an optional rate parameter (e.g., 20 req/sec) to control resource usage.  
3. **Domain Filtering:**  
   * Accept an optional domain\_only boolean to limit crawling to the starting URL’s domain.

---

**4\. Non-Functional Requirements**

**4.1 Performance**

* **Speed:**  
  * Target \~50-100 links/sec with 50 workers for a \~5,000-link scan (70% static, 20% spoofed, 10% rendered).  
  * Per-page: \~10-50 ms (static), \~200-500 ms (spoofed), \~500-1000 ms (rendered).  
* **Throughput:** Maximize link checks per second, constrained by Chrome’s \~10-20 page concurrency limit.  
* **API Latency:**  
  * /scan: \<500 ms response time.  
  * /scan/{scan\_id}: \<200 ms response time.

**4.2 Resource Usage**

* **Memory:**  
  * Total peak: \~100-300 MB (shared Chrome instance \+ Go runtime \+ JSON buffering).  
  * Per tier: \~1-5 MB (static), \~50-150 MB (spoofed), \~100-200 MB (rendered).  
* **CPU:**  
  * Low (5-10%) for static parsing, moderate (50-100% of 1-2 cores) during rendering bursts.  
* **Efficiency:** Minimize resource use by preferring static parsing and limiting rendering.

**4.3 Coverage**

* **Goal:** \~90-95% of links detected (static \+ prerendered \+ dynamic).  
* **Tiers:** \~50-70% (static), \~70-85% (spoofed), \~90-95% (rendered).

**4.4 Usability**

* **API:** RESTful, intuitive endpoints with clear JSON schemas.  
* **Report:** Well-structured JSON with distinct working\_links and broken\_links sections for easy parsing.

**4.5 Reliability**

* **Uptime:** Server runs continuously, handling multiple concurrent scans without crashing.  
* **Error Handling:** Gracefully manage network errors, timeouts, and Chrome failures, logging them in the report.

**4.6 Portability**

* Deployable on any OS with Go and Chrome, packaged as a single binary.

**4.7 Maintainability**

* Modular code with separate API, scanning, and reporting logic, easy to extend (e.g., add profiling).

---

# **5\. Constraints**

1. **Chrome Dependency:** Requires a local Chrome/Chromium installation (\~100-200 MB disk space).  
2. **Network:** Dependent on internet speed and site responsiveness; timeouts (10s HTTP, 3-5s rendering) limit delays.  
3. **Concurrency Limits:** Chrome caps parallel page rendering at \~10-20, constraining throughput beyond 50 workers.  
4. **Dynamic Content:** Cannot detect links loaded post-render (e.g., via “Load More” buttons) without interaction simulation.  
5. **Storage:** Assumes sufficient disk space for JSON reports (e.g., \~1-5 MB per \~5,000-link scan).

---

# **6\. Assumptions**

1. **Site Accessibility:** Websites are publicly accessible and respond to HTTP requests.  
2. **Link Distribution:** Depth-2 sites have \~5,000 unique links (e.g., 100 at depth 0, 50 at depth 1, 20 at depth 2), with overlap handled by deduplication.  
3. **API Usage:** Users or integrating apps can handle asynchronous scan requests and retrieve results via scan\_id.  
4. **SEO Practices:** Many modern sites prerender for crawlers, making spoofing effective.  
5. **File System:** Server has write access to a reports/ directory for JSON storage.

---

**7\. System Architecture**

**7.1 Components**

* **API Server:** HTTP server (e.g., net/http) exposing RESTful endpoints.  
* **Scan Manager:** Coordinates scans, assigns scan\_id, and tracks progress.  
* **Worker Pool:** Goroutines process tasks from a buffered channel, capped by a semaphore for rate limiting.  
* **Hybrid Engine:**  
  * **Goquery**: Parses static HTML.  
  * **Go-Rod:** Handles spoofing and rendering with a shared Chrome instance.  
* **Link Manager:** Deduplicates URLs (sync.Map) and collects results.  
* **Report Generator:** Saves scan data as structured JSON.

**7.2 Flow**

1. Client sends POST /scan with URL and options.  
2. Server assigns scan\_id, starts scan in a goroutine, and responds with status.  
3. Workers fetch and check links:  
   * Try goquery parsing.  
   * Escalate to go-rod spoofing if coverage is low.  
   * Escalate to go-rod rendering if needed.  
4. Results are saved as reports/\<scan\_id\>.json.  
5. Client queries GET /scan/{scan\_id} to check status or retrieve report path.

---

**8\. Success Criteria**

* **Functional:**  
  * Correctly identifies \~90-95% of links (working and broken) up to depth 2 across static, MPA, and SPA sites.  
  * Produces a JSON report with distinct working\_links and broken\_links sections.  
* **Performance:** Completes a \~5,000-link scan in \~50-100 seconds with 50 workers.  
* **Resource:** Stays under \~300 MB memory and \~100% of 2 cores peak usage.  
* **API:** Responsive (\<500 ms for /scan, \<200 ms for /scan/{scan\_id}), reliable for concurrent requests.

---

**9\. Risks & Mitigations**

| Risk | Impact | Mitigation |
| :---- | :---- | :---- |
| Chrome setup complexity | Deployment issues | Documented setup in README |
| Slow SPA rendering | Reduced throughput | Tight timeouts (5s), selective use |
| Misclassification of tiers | Lower coverage | Robust escalation heuristics |
| Network timeouts | Incomplete scans | Configurable HTTP timeouts (10s) |
| Memory spikes from Chrome | Server strain | Single browser instance, disable extras |
| API overload | Server downtime | Rate limiting, worker caps |
| JSON file size | Disk space | Limit depth, deduplicate links |

---

**10\. Future Enhancements**

* **Profiling:** Cache site types for recurring scans (in-memory or database).  
* **Async Notifications:** Webhook support for scan completion.  
* **Authentication:** API keys for secure access.  
* **UI:** Optional web dashboard to visualize reports.  
* **Proxy Support:** Support to use a proxy to scan the URLs so as to avoid a false positive for a DDOS attack / getting blocked..

[image1]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAnAAAAAHCAYAAACIq3DzAAAAQUlEQVR4Xu3WMQ0AIADAMFziCFGYgx8FLOnRZwo25l4HAICO8QYAAP5m4AAAYgwcAECMgQMAiDFwAAAxBg4AIOYClIUh9UOLBN8AAAAASUVORK5CYII=>