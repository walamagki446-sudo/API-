# Web Capture Tool - Project Summary

## Overview

This project provides a comprehensive solution to capture **everything** from a website, addressing the requirement to capture "every single thing from a site - APIs, HTML, etc."

## What It Captures

The tool captures the following data from any website:

1. **HTML Content** - Complete page HTML, including dynamically loaded content
2. **API Requests/Responses** - All network API calls with full request and response data
3. **Network Traffic** - Complete details of all HTTP requests
4. **Headers** - Request and response headers for all network calls
5. **Cookies** - Session and tracking cookies
6. **Resources** - URLs for all external resources (scripts, stylesheets, images)
7. **Screenshots** - Full-page screenshots of the rendered site

## Implementation Details

### Core Components

1. **web_capture.go** - Main capture library
   - Chrome DevTools Protocol integration
   - Network event interception
   - Request/response matching
   - Resource extraction
   - Screenshot capture

2. **capture_cli.go** - Command-line interface
   - User-friendly CLI with flags
   - Multiple capture modes
   - Help system
   - Error handling

3. **Go Modules**
   - chromedp - Chrome automation
   - cdproto - Chrome DevTools Protocol
   - Standard Go libraries

### Architecture

```
┌─────────────┐
│   User CLI  │
└──────┬──────┘
       │
       v
┌──────────────────┐
│  WebCapture      │
│  - Config        │
│  - Capture()     │
└──────┬───────────┘
       │
       v
┌──────────────────────┐
│  Chrome/Chromium     │
│  - Navigate          │
│  - Intercept Network │
│  - Capture Events    │
└──────┬───────────────┘
       │
       v
┌──────────────────────┐
│  Output              │
│  - page.html         │
│  - captured_apis.json│
│  - screenshot.png    │
│  - resources.json    │
│  - summary.txt       │
└──────────────────────┘
```

## Usage Modes

### 1. Full Capture Mode (Default)
Uses Chrome/Chromium with DevTools Protocol:
```bash
./web-capture -url https://example.com
```

**Captures:**
- Dynamic JavaScript-rendered content
- All network requests via CDP
- Full screenshots
- Complete resource list

### 2. Simple Mode
Fast HTTP-only capture:
```bash
./web-capture -url https://example.com -simple
```

**Captures:**
- Static HTML content
- Cookies
- Basic HTTP headers

## Output Format

All captures are saved to a timestamped directory (or custom directory) with:

### captured_apis.json
```json
[
  {
    "id": 1,
    "timestamp": "2025-11-07T19:41:07.502Z",
    "page": "https://example.com",
    "type": "fetch",
    "url": "/api/endpoint",
    "method": "POST",
    "requestHeaders": {...},
    "requestBody": "...",
    "status": 200,
    "statusText": "OK",
    "responseHeaders": {...},
    "responseBody": "..."
  }
]
```

### resources.json
```json
{
  "scripts": ["https://..."],
  "stylesheets": ["https://..."],
  "images": ["https://..."]
}
```

### page.html
Complete HTML source code

### screenshot.png
Full-page screenshot (PNG format)

### summary.txt
Human-readable summary of the capture

## Key Features

### 1. Comprehensive Capture
- Captures all types of network requests (fetch, XHR, scripts, etc.)
- Includes both request and response data
- Preserves headers and cookies

### 2. Multiple Modes
- **Full mode**: Complete capture with JavaScript rendering
- **Simple mode**: Fast HTTP-only capture

### 3. Flexible Configuration
- Adjustable wait times for dynamic content
- Selective capture (APIs only, HTML only, etc.)
- Custom output directories

### 4. User-Friendly
- Clear command-line interface
- Helpful error messages
- Comprehensive documentation

## Use Cases

1. **API Reverse Engineering**
   - Capture all API endpoints
   - Analyze request/response formats
   - Document undocumented APIs

2. **Web Scraping**
   - Extract complete page content
   - Download resources
   - Archive websites

3. **Security Testing**
   - Analyze headers and cookies
   - Identify security issues
   - Test authentication flows

4. **Performance Analysis**
   - Monitor network requests
   - Analyze load times
   - Identify bottlenecks

5. **Competitive Analysis**
   - Study competitor websites
   - Analyze technology stack
   - Monitor changes over time

## Documentation

- **README.md** - Main documentation and overview
- **USAGE.md** - Detailed usage guide with examples
- **QUICKREF.md** - Quick reference card
- **examples.sh** - Example commands

## Technical Improvements

### Security
- No vulnerabilities detected by CodeQL
- Safe handling of user input
- Proper error handling

### Code Quality
- Proper request/response matching using RequestID
- Thread-safe operations with mutex locks
- Error handling for all critical operations
- Resource cleanup and goroutine management

### Performance
- Concurrent processing where appropriate
- Configurable timeouts
- Efficient resource usage

## Building

```bash
# Build for current platform
go build -o web-capture capture_cli.go web_capture.go

# Cross-compile for other platforms
GOOS=linux GOARCH=amd64 go build -o web-capture-linux
GOOS=windows GOARCH=amd64 go build -o web-capture.exe
GOOS=darwin GOARCH=amd64 go build -o web-capture-mac
```

## Dependencies

- Go 1.21+
- Chrome/Chromium (for full mode)
- chromedp/chromedp
- chromedp/cdproto
- Standard Go libraries

## Future Enhancements

Potential improvements:
- Download resources automatically
- Parallel page captures
- Interactive mode
- WebSocket capture
- HAR file export
- Browser session persistence
- Proxy support
- Custom JavaScript injection

## Conclusion

This tool provides a complete solution for capturing all data from websites, including HTML, APIs, network traffic, and resources. It supports multiple capture modes, provides extensive configuration options, and includes comprehensive documentation. The implementation is secure, efficient, and user-friendly, making it suitable for various use cases from API reverse engineering to web scraping.

## Quick Start

```bash
# 1. Build
go build -o web-capture capture_cli.go web_capture.go

# 2. Run
./web-capture -url https://example.com

# 3. Check output
ls capture_*/
```

That's it! You now have everything from the website captured in an organized directory.
