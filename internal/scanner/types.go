package scanner

type LinkResult struct {
    URL    string `json:"url"`
	Ref    string `json:"ref"`
    Status string `json:"status"`
    Code   int    `json:"code"`
    Depth  int    `json:"depth,omitempty"`
    Tier   string `json:"tier,omitempty"`  // static, spoofed, or rendered
}

type CrawlTask struct {
	BaseURL string
    URL     string
    Depth   int
    Type    string
}

var LinkTypes = map[string]string{
    "a[href]":       "href",
    "link[href]":    "href",
    "script[src]":   "src",
    "img[src]":      "src",
    "source[src]":   "src",
    "video[src]":    "src",
    "audio[src]":    "src",
    "iframe[src]":   "src",
    "embed[src]":    "src",
    "object[data]":  "data",
    "param[value]":  "value",
    "track[src]":    "src",
}
