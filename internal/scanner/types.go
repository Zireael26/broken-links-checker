package scanner

type LinkResult struct {
    URL    string `json:"url"`
	Ref    string `json:"ref"`
    Status string `json:"status"`
    Code   int    `json:"code"`
    Depth  int    `json:"depth,omitempty"`
}

type CrawlTask struct {
    URL   string
    Depth int
}
