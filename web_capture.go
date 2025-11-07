package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// CapturedRequest represents a captured API/network request
type CapturedRequest struct {
	ID              int64             `json:"id"`
	Timestamp       string            `json:"timestamp"`
	Page            string            `json:"page"`
	Type            string            `json:"type"`
	URL             string            `json:"url"`
	Method          string            `json:"method"`
	RequestHeaders  map[string]string `json:"requestHeaders"`
	RequestBody     string            `json:"requestBody,omitempty"`
	Status          int               `json:"status"`
	StatusText      string            `json:"statusText"`
	ResponseHeaders map[string]string `json:"responseHeaders"`
	ResponseBody    string            `json:"responseBody,omitempty"`
	RequestID       string            `json:"-"` // Internal use only, not exported to JSON
}

// CaptureConfig holds configuration for web capture
type CaptureConfig struct {
	URL              string
	OutputDir        string
	CaptureAPIs      bool
	CaptureHTML      bool
	CaptureResources bool
	CaptureScreenshot bool
	WaitTime         time.Duration
}

// WebCapture handles comprehensive website capture
type WebCapture struct {
	config          CaptureConfig
	requests        []CapturedRequest
	requestMutex    sync.Mutex
	requestCounter  int64
	htmlContent     string
	cookies         []*http.Cookie
}

// NewWebCapture creates a new web capture instance
func NewWebCapture(config CaptureConfig) *WebCapture {
	if config.OutputDir == "" {
		config.OutputDir = fmt.Sprintf("capture_%d", time.Now().Unix())
	}
	if config.WaitTime == 0 {
		config.WaitTime = 5 * time.Second
	}
	
	return &WebCapture{
		config:   config,
		requests: make([]CapturedRequest, 0),
	}
}

// Capture performs comprehensive website capture
func (wc *WebCapture) Capture() error {
	// Create output directory
	if err := os.MkdirAll(wc.config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	log.Printf("Starting capture of %s", wc.config.URL)
	log.Printf("Output directory: %s", wc.config.OutputDir)

	// Setup Chrome context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set timeout
	ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Enable network events
	chromedp.ListenTarget(ctx, wc.captureNetworkEvents(ctx))

	// Navigate and capture
	var htmlContent string
	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate(wc.config.URL),
		chromedp.Sleep(wc.config.WaitTime),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return fmt.Errorf("chromedp run error: %v", err)
	}

	wc.htmlContent = htmlContent

	// Capture screenshot if enabled
	if wc.config.CaptureScreenshot {
		var screenshotBuf []byte
		if err := chromedp.Run(ctx,
			chromedp.FullScreenshot(&screenshotBuf, 90),
		); err != nil {
			log.Printf("Screenshot capture error: %v", err)
		} else {
			screenshotPath := filepath.Join(wc.config.OutputDir, "screenshot.png")
			if err := os.WriteFile(screenshotPath, screenshotBuf, 0644); err != nil {
				log.Printf("Failed to save screenshot: %v", err)
			} else {
				log.Printf("Screenshot saved: %s", screenshotPath)
			}
		}
	}

	// Save captured data
	return wc.saveData()
}

// captureNetworkEvents sets up network event listeners
func (wc *WebCapture) captureNetworkEvents(ctx context.Context) func(interface{}) {
	return func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			wc.handleRequestWillBeSent(ev)
		case *network.EventResponseReceived:
			wc.handleResponseReceived(ctx, ev)
		}
	}
}

// handleRequestWillBeSent captures outgoing requests
func (wc *WebCapture) handleRequestWillBeSent(ev *network.EventRequestWillBeSent) {
	if !wc.config.CaptureAPIs {
		return
	}

	wc.requestMutex.Lock()
	defer wc.requestMutex.Unlock()

	wc.requestCounter++
	
	reqHeaders := make(map[string]string)
	for k, v := range ev.Request.Headers {
		if str, ok := v.(string); ok {
			reqHeaders[k] = str
		}
	}

	capturedReq := CapturedRequest{
		ID:             int64(wc.requestCounter),
		Timestamp:      time.Now().Format(time.RFC3339),
		Page:           wc.config.URL,
		Type:           ev.Type.String(),
		URL:            ev.Request.URL,
		Method:         ev.Request.Method,
		RequestHeaders: reqHeaders,
		RequestBody:    ev.Request.PostData,
		RequestID:      ev.RequestID.String(), // Store RequestID for proper matching
	}

	// Store with request ID for later matching with response
	wc.requests = append(wc.requests, capturedReq)
}

// handleResponseReceived captures incoming responses
func (wc *WebCapture) handleResponseReceived(ctx context.Context, ev *network.EventResponseReceived) {
	if !wc.config.CaptureAPIs {
		return
	}

	wc.requestMutex.Lock()
	defer wc.requestMutex.Unlock()

	respHeaders := make(map[string]string)
	for k, v := range ev.Response.Headers {
		if str, ok := v.(string); ok {
			respHeaders[k] = str
		}
	}

	// Find matching request using RequestID for accurate matching
	requestID := ev.RequestID.String()
	for i := range wc.requests {
		if wc.requests[i].RequestID == requestID {
			wc.requests[i].Status = int(ev.Response.Status)
			wc.requests[i].StatusText = ev.Response.StatusText
			wc.requests[i].ResponseHeaders = respHeaders

			// Get response body asynchronously
			go wc.getResponseBody(ctx, ev.RequestID, i)
			break
		}
	}
}

// getResponseBody retrieves the response body for a request
func (wc *WebCapture) getResponseBody(ctx context.Context, reqID network.RequestID, requestIndex int) {
	body, err := network.GetResponseBody(reqID).Do(ctx)
	if err == nil && len(body) > 0 {
		wc.requestMutex.Lock()
		defer wc.requestMutex.Unlock()
		
		// Verify the index is still valid
		if requestIndex < len(wc.requests) {
			wc.requests[requestIndex].ResponseBody = string(body)
		}
	}
}

// saveData saves all captured data to files
func (wc *WebCapture) saveData() error {
	// Save HTML content
	if wc.config.CaptureHTML && wc.htmlContent != "" {
		htmlPath := filepath.Join(wc.config.OutputDir, "page.html")
		if err := os.WriteFile(htmlPath, []byte(wc.htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to save HTML: %v", err)
		}
		log.Printf("HTML saved: %s", htmlPath)

		// Extract and save embedded resources
		if wc.config.CaptureResources {
			wc.extractResources()
		}
	}

	// Save API requests
	if wc.config.CaptureAPIs && len(wc.requests) > 0 {
		apisPath := filepath.Join(wc.config.OutputDir, "captured_apis.json")
		data, err := json.MarshalIndent(wc.requests, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal APIs: %v", err)
		}
		if err := os.WriteFile(apisPath, data, 0644); err != nil {
			return fmt.Errorf("failed to save APIs: %v", err)
		}
		log.Printf("APIs saved: %s (%d requests captured)", apisPath, len(wc.requests))
	}

	// Create summary
	summary := wc.createSummary()
	summaryPath := filepath.Join(wc.config.OutputDir, "summary.txt")
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		log.Printf("Failed to save summary: %v", err)
	} else {
		log.Printf("Summary saved: %s", summaryPath)
	}

	return nil
}

// extractResources extracts URLs of resources from HTML
func (wc *WebCapture) extractResources() {
	resources := make(map[string][]string)
	
	// Extract scripts
	scriptRe := regexp.MustCompile(`<script[^>]*src=["']([^"']+)["']`)
	scripts := scriptRe.FindAllStringSubmatch(wc.htmlContent, -1)
	for _, match := range scripts {
		if len(match) > 1 {
			resources["scripts"] = append(resources["scripts"], match[1])
		}
	}

	// Extract stylesheets
	cssRe := regexp.MustCompile(`<link[^>]*href=["']([^"']+\.css[^"']*)["']`)
	css := cssRe.FindAllStringSubmatch(wc.htmlContent, -1)
	for _, match := range css {
		if len(match) > 1 {
			resources["stylesheets"] = append(resources["stylesheets"], match[1])
		}
	}

	// Extract images
	imgRe := regexp.MustCompile(`<img[^>]*src=["']([^"']+)["']`)
	images := imgRe.FindAllStringSubmatch(wc.htmlContent, -1)
	for _, match := range images {
		if len(match) > 1 {
			resources["images"] = append(resources["images"], match[1])
		}
	}

	// Save resources list
	if len(resources) > 0 {
		resourcesPath := filepath.Join(wc.config.OutputDir, "resources.json")
		data, err := json.MarshalIndent(resources, "", "  ")
		if err != nil {
			log.Printf("Failed to marshal resources: %v", err)
			return
		}
		if err := os.WriteFile(resourcesPath, data, 0644); err != nil {
			log.Printf("Failed to save resources: %v", err)
			return
		}
		log.Printf("Resources list saved: %s", resourcesPath)
	}
}

// createSummary creates a summary of the capture
func (wc *WebCapture) createSummary() string {
	var sb strings.Builder
	
	sb.WriteString("═══════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf("   WEB CAPTURE SUMMARY\n"))
	sb.WriteString(fmt.Sprintf("   %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString("═══════════════════════════════════\n\n")
	
	sb.WriteString(fmt.Sprintf("Target URL: %s\n", wc.config.URL))
	sb.WriteString(fmt.Sprintf("Output Directory: %s\n\n", wc.config.OutputDir))
	
	sb.WriteString("Captured Data:\n")
	if wc.config.CaptureHTML {
		sb.WriteString(fmt.Sprintf("  ✓ HTML Content (%d bytes)\n", len(wc.htmlContent)))
	}
	if wc.config.CaptureAPIs {
		sb.WriteString(fmt.Sprintf("  ✓ API Requests (%d requests)\n", len(wc.requests)))
		
		// Count by type
		typeCount := make(map[string]int)
		for _, req := range wc.requests {
			typeCount[req.Type]++
		}
		for reqType, count := range typeCount {
			sb.WriteString(fmt.Sprintf("    - %s: %d\n", reqType, count))
		}
	}
	if wc.config.CaptureScreenshot {
		sb.WriteString("  ✓ Screenshot\n")
	}
	if wc.config.CaptureResources {
		sb.WriteString("  ✓ Resource URLs\n")
	}
	
	sb.WriteString("\n")
	sb.WriteString("Files Created:\n")
	if wc.config.CaptureHTML {
		sb.WriteString("  - page.html\n")
	}
	if wc.config.CaptureAPIs {
		sb.WriteString("  - captured_apis.json\n")
	}
	if wc.config.CaptureScreenshot {
		sb.WriteString("  - screenshot.png\n")
	}
	if wc.config.CaptureResources {
		sb.WriteString("  - resources.json\n")
	}
	sb.WriteString("  - summary.txt\n")
	
	return sb.String()
}

// CaptureWebsite is a convenience function to capture a website with default settings
func CaptureWebsite(targetURL string) error {
	config := CaptureConfig{
		URL:               targetURL,
		OutputDir:         fmt.Sprintf("capture_%d", time.Now().Unix()),
		CaptureAPIs:       true,
		CaptureHTML:       true,
		CaptureResources:  true,
		CaptureScreenshot: true,
		WaitTime:          5 * time.Second,
	}

	wc := NewWebCapture(config)
	return wc.Capture()
}

// DownloadResource downloads a resource from a URL
func DownloadResource(resourceURL, outputPath string) error {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(resourceURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", resourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status downloading %s: %s", resourceURL, resp.Status)
	}

	// Create directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save file
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// Simple HTTP client-based capture (fallback if Chrome is not available)
func SimpleCaptureHTML(targetURL, outputPath string) error {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(targetURL)
	if err != nil {
		return fmt.Errorf("failed to fetch URL: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if err := os.WriteFile(outputPath, body, 0644); err != nil {
		return fmt.Errorf("failed to save HTML: %v", err)
	}

	log.Printf("HTML saved to: %s (%d bytes)", outputPath, len(body))
	
	// Save cookies if any
	parsedURL, _ := url.Parse(targetURL)
	cookies := jar.Cookies(parsedURL)
	if len(cookies) > 0 {
		cookiesData, _ := json.MarshalIndent(cookies, "", "  ")
		cookiesPath := filepath.Join(filepath.Dir(outputPath), "cookies.json")
		os.WriteFile(cookiesPath, cookiesData, 0644)
		log.Printf("Cookies saved to: %s", cookiesPath)
	}

	return nil
}
