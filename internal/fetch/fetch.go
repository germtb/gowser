package fetch

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Use a realistic User-Agent to avoid being blocked by sites like Google
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// User-Agent for text-only/lite mode - identifies as a basic text browser
const liteUserAgent = "Lynx/2.8.9rel.1 libwww-FM/2.14 SSL-MM/1.4.1"

// liteURLMappings maps domains to their known lite/text versions
var liteURLMappings = map[string]func(string) string{
	"reddit.com": func(u string) string {
		return strings.Replace(u, "reddit.com", "old.reddit.com", 1)
	},
	"www.reddit.com": func(u string) string {
		return strings.Replace(u, "www.reddit.com", "old.reddit.com", 1)
	},
	"twitter.com": func(u string) string {
		return strings.Replace(u, "twitter.com", "nitter.net", 1)
	},
	"x.com": func(u string) string {
		return strings.Replace(u, "x.com", "nitter.net", 1)
	},
	"en.wikipedia.org": func(u string) string {
		return strings.Replace(u, "en.wikipedia.org/wiki/", "en.m.wikipedia.org/wiki/", 1)
	},
	"www.npr.org": func(u string) string {
		return strings.Replace(u, "www.npr.org", "text.npr.org", 1)
	},
	"cnn.com": func(u string) string {
		return strings.Replace(u, "cnn.com", "lite.cnn.com", 1)
	},
	"www.cnn.com": func(u string) string {
		return strings.Replace(u, "www.cnn.com", "lite.cnn.com", 1)
	},
}

// FetchHTML fetches the HTML content from the given URL.
func FetchHTML(targetURL string) (string, error) {
	return FetchHTMLWithOptions(targetURL, false)
}

// FetchHTMLLite fetches HTML preferring lite/text versions when available.
func FetchHTMLLite(targetURL string) (string, error) {
	return FetchHTMLWithOptions(targetURL, true)
}

// FetchHTMLWithOptions fetches HTML with optional lite mode.
func FetchHTMLWithOptions(targetURL string, liteMode bool) (string, error) {
	// In lite mode, try to convert to a known lite URL
	if liteMode {
		targetURL = tryLiteURL(targetURL)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	if liteMode {
		// Use text browser user agent
		req.Header.Set("User-Agent", liteUserAgent)
		// Request only basic HTML, no JS
		req.Header.Set("Accept", "text/html")
		// Indicate we don't want JavaScript
		req.Header.Set("X-Requested-With", "")
		// Some sites check for this to serve simplified versions
		req.Header.Set("Save-Data", "on")
	} else {
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	return string(body), nil
}

// tryLiteURL attempts to convert a URL to its lite/text version if known.
func tryLiteURL(targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}

	host := parsed.Host
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Check if we have a lite mapping for this domain
	if mapper, ok := liteURLMappings[host]; ok {
		return mapper(targetURL)
	}

	return targetURL
}

// ResolveRedirectURL handles redirect URLs from search engines like DuckDuckGo.
// It extracts the actual destination URL from the redirect parameters.
func ResolveRedirectURL(targetURL string) string {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return targetURL
	}

	// Handle DuckDuckGo redirect URLs
	// Format: https://duckduckgo.com/l/?uddg=<encoded-url>&rut=<token>
	if strings.Contains(parsed.Host, "duckduckgo.com") && parsed.Path == "/l/" {
		if uddg := parsed.Query().Get("uddg"); uddg != "" {
			return uddg
		}
	}

	// Handle Google redirect URLs
	// Format: https://www.google.com/url?q=<encoded-url>&sa=...
	if strings.Contains(parsed.Host, "google.com") && parsed.Path == "/url" {
		if q := parsed.Query().Get("q"); q != "" {
			return q
		}
	}

	return targetURL
}

// FetchHTMLWithJS fetches HTML after JavaScript execution using a headless browser.
// This is useful for JS-heavy sites that don't work with simple HTTP fetching.
// Requires Chrome/Chromium to be installed on the system.
func FetchHTMLWithJS(targetURL string) (string, error) {
	return FetchHTMLWithJSOptions(targetURL, 5*time.Second)
}

// FetchHTMLWithJSOptions fetches HTML with JS execution and configurable wait time.
func FetchHTMLWithJSOptions(targetURL string, waitTime time.Duration) (string, error) {
	// Find Chrome/Chromium path automatically
	path, found := launcher.LookPath()
	if !found {
		return "", fmt.Errorf("Chrome/Chromium not found. Please install Chrome or Chromium")
	}

	// Launch browser in headless mode
	u := launcher.New().
		Bin(path).
		Headless(true).
		// Disable GPU for better compatibility
		Set("disable-gpu").
		// Reduce resource usage
		Set("disable-extensions").
		Set("disable-dev-shm-usage").
		Set("no-sandbox").
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	// Create a new page
	page := browser.MustPage(targetURL)
	defer page.MustClose()

	// Wait for the page to load
	page.MustWaitLoad()

	// Wait additional time for JS to execute (SPAs, lazy loading, etc.)
	if waitTime > 0 {
		time.Sleep(waitTime)
	}

	// Try to wait for network to be idle (no requests for 500ms)
	page.MustWaitIdle()

	// Get the rendered HTML
	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("getting page HTML: %w", err)
	}

	return html, nil
}

// FetchHTMLWithJSWaitFor fetches HTML and waits for a specific element to appear.
// Useful for SPAs where you need to wait for specific content to load.
func FetchHTMLWithJSWaitFor(targetURL string, selector string, timeout time.Duration) (string, error) {
	path, found := launcher.LookPath()
	if !found {
		return "", fmt.Errorf("Chrome/Chromium not found. Please install Chrome or Chromium")
	}

	u := launcher.New().
		Bin(path).
		Headless(true).
		Set("disable-gpu").
		Set("disable-extensions").
		Set("disable-dev-shm-usage").
		Set("no-sandbox").
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage(targetURL)
	defer page.MustClose()

	page.MustWaitLoad()

	// Wait for specific element if selector provided
	if selector != "" {
		page.Timeout(timeout).MustElement(selector)
	}

	page.MustWaitIdle()

	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("getting page HTML: %w", err)
	}

	return html, nil
}

// CheckBrowserAvailable checks if Chrome/Chromium is available on the system.
func CheckBrowserAvailable() bool {
	_, found := launcher.LookPath()
	return found
}

// Ensure proto is used (for future enhancements like screenshots, PDF, etc.)
var _ = proto.TargetTargetID("")
