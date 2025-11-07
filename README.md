# API- - Web Capture Tool

Comprehensive website capture tool that captures **everything** from a website including HTML, APIs, network traffic, and resources.

## Features

✨ **What can be captured:**

- 🌐 **Full HTML Content** - Including dynamically loaded content
- 📡 **API Requests/Responses** - All XHR, Fetch, and API calls
- 🔄 **Network Traffic** - Complete request/response details
- 📋 **Headers** - All request and response headers
- 🍪 **Cookies** - Session cookies and tracking data
- 📦 **Resources** - URLs for scripts, stylesheets, images
- 📸 **Screenshots** - Full page screenshots
- 🎯 **Request Types** - Fetch, XHR, Document, Script, Stylesheet, Image, etc.

## Installation

### Prerequisites

- Go 1.21 or higher
- Chrome/Chromium (for full capture mode)

### Install Dependencies

```bash
go mod download
```

## Usage

### Quick Start - Capture Everything

```bash
# Capture a website with all default options
go run . -url https://example.com
```

This will create a directory `capture_<timestamp>/` containing:
- `page.html` - Full HTML content
- `captured_apis.json` - All API requests and responses
- `screenshot.png` - Full page screenshot
- `resources.json` - List of all resource URLs
- `summary.txt` - Capture summary

### Custom Output Directory

```bash
go run . -url https://example.com -output my_capture
```

### Only Capture Specific Data

```bash
# Only HTML and APIs, no screenshot
go run . -url https://example.com -screenshot=false

# Only APIs
go run . -url https://example.com -html=false -screenshot=false -resources=false
```

### Adjust Wait Time

For websites with lots of dynamic content:

```bash
# Wait 10 seconds for page to fully load
go run . -url https://example.com -wait 10
```

### Simple Mode (Fast, No JavaScript)

For static websites or when Chrome is not available:

```bash
go run . -url https://example.com -simple
```

## Command Line Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-url` | string | **required** | Target URL to capture |
| `-output` | string | `capture_<timestamp>` | Output directory |
| `-apis` | bool | `true` | Capture API requests/responses |
| `-html` | bool | `true` | Capture HTML content |
| `-resources` | bool | `true` | Extract resource URLs |
| `-screenshot` | bool | `true` | Capture screenshot |
| `-wait` | int | `5` | Wait time in seconds |
| `-simple` | bool | `false` | Use simple HTTP mode |
| `-help` | bool | `false` | Show help |

## Output Format

### captured_apis.json

Each API request is captured with complete details:

```json
{
  "id": 1,
  "timestamp": "2025-11-07T19:41:07.502Z",
  "page": "https://example.com",
  "type": "fetch",
  "url": "/api/data",
  "method": "POST",
  "requestHeaders": {
    "Content-Type": "application/json"
  },
  "requestBody": "{\"key\":\"value\"}",
  "status": 200,
  "statusText": "OK",
  "responseHeaders": {
    "content-type": "application/json"
  },
  "responseBody": "{\"result\":\"success\"}"
}
```

### resources.json

Categorized list of all resources:

```json
{
  "scripts": [
    "https://example.com/app.js",
    "https://cdn.example.com/library.js"
  ],
  "stylesheets": [
    "https://example.com/styles.css"
  ],
  "images": [
    "https://example.com/logo.png"
  ]
}
```

## Use Cases

### 1. API Reverse Engineering

Capture all API endpoints and their request/response formats:

```bash
go run . -url https://api-site.com -html=false -screenshot=false
```

### 2. Website Cloning

Capture complete website with all resources:

```bash
go run . -url https://example.com -wait 10
```

### 3. Security Testing

Analyze network traffic and headers:

```bash
go run . -url https://target.com -resources=false
```

### 4. Performance Analysis

Capture all network requests to analyze load times:

```bash
go run . -url https://slow-site.com -wait 15
```

## Building

### Build for Current Platform

```bash
go build -o web-capture
./web-capture -url https://example.com
```

### Build for All Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o web-capture-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o web-capture.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o web-capture-mac
```

## Advanced Usage

### Use as a Library

```go
package main

import (
    "log"
    "time"
)

func main() {
    config := CaptureConfig{
        URL:               "https://example.com",
        OutputDir:         "my_capture",
        CaptureAPIs:       true,
        CaptureHTML:       true,
        CaptureResources:  true,
        CaptureScreenshot: true,
        WaitTime:          5 * time.Second,
    }

    wc := NewWebCapture(config)
    if err := wc.Capture(); err != nil {
        log.Fatal(err)
    }
}
```

### Simple HTML Capture

```go
err := SimpleCaptureHTML("https://example.com", "output.html")
```

## Examples

### Capture E-commerce Site

```bash
# Wait longer for dynamic content to load
go run . -url https://shop.example.com -wait 10 -output shop_capture
```

### Capture SPA (Single Page Application)

```bash
# SPAs need time to render
go run . -url https://app.example.com -wait 8
```

### Capture API-Heavy Site

```bash
# Focus on APIs
go run . -url https://api.example.com -screenshot=false
```

## Troubleshooting

### Chrome Not Found

If you get "Chrome not found" errors:

1. Install Chrome or Chromium
2. Use simple mode: `-simple`

### Timeout Errors

If capture times out:

1. Increase wait time: `-wait 15`
2. Check network connection
3. Try simple mode for static sites

### Memory Issues

For large sites:

1. Disable screenshot: `-screenshot=false`
2. Disable resources: `-resources=false`
3. Use simple mode: `-simple`

## Project Structure

- `web_capture.go` - Main capture library with Chrome support
- `capture_cli.go` - Command-line interface
- `zalando.go` - Original Zalando bot (legacy)
- `go.mod` - Dependencies

## Dependencies

- `chromedp/chromedp` - Chrome DevTools Protocol
- `chromedp/cdproto` - Chrome DevTools Protocol types
- Standard Go libraries

## Contributing

Contributions welcome! Please feel free to submit issues or pull requests.

## License

MIT License

## Author

Built for comprehensive website data capture and analysis.