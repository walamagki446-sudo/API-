# Web Capture Tool - Detailed Usage Guide

## Table of Contents

1. [Quick Start](#quick-start)
2. [Capturing Different Types of Sites](#capturing-different-types-of-sites)
3. [Understanding the Output](#understanding-the-output)
4. [Advanced Scenarios](#advanced-scenarios)
5. [Troubleshooting](#troubleshooting)

## Quick Start

### Install and Build

```bash
# Clone the repository (if you haven't already)
git clone https://github.com/walamagki446-sudo/API-.git
cd API-

# Install dependencies
go mod download

# Build the tool
go build -o web-capture capture_cli.go web_capture.go

# Run with help to see all options
./web-capture -help
```

### Your First Capture

```bash
# Capture everything from a website
./web-capture -url https://example.com
```

This creates a directory `capture_<timestamp>/` with:
- `page.html` - The full HTML
- `captured_apis.json` - All API calls
- `screenshot.png` - Page screenshot
- `resources.json` - Resource URLs
- `summary.txt` - Capture summary

## Capturing Different Types of Sites

### Static Websites

For simple static sites, use simple mode for faster capture:

```bash
./web-capture -url https://static-site.com -simple
```

### Single Page Applications (SPAs)

SPAs need time to load JavaScript and fetch data:

```bash
# Wait 10 seconds for app to fully load
./web-capture -url https://app.example.com -wait 10
```

### E-commerce Sites

Capture product pages with all dynamic content:

```bash
./web-capture -url https://shop.example.com/product/123 -wait 8 -output product_capture
```

### API-Heavy Sites

Focus on capturing API endpoints:

```bash
./web-capture -url https://dashboard.example.com -screenshot=false -wait 10
```

### News/Media Sites

Capture articles with images:

```bash
./web-capture -url https://news.example.com/article -wait 5 -output article_capture
```

### Social Media

For sites with infinite scroll or delayed loading:

```bash
./web-capture -url https://social.example.com/profile -wait 15
```

## Understanding the Output

### Captured APIs JSON Structure

The `captured_apis.json` file contains an array of all network requests:

```json
[
  {
    "id": 1,
    "timestamp": "2025-11-07T19:41:07.502Z",
    "page": "https://example.com",
    "type": "fetch",              // Request type: fetch, xhr, document, script, etc.
    "url": "/api/user/profile",   // Endpoint URL
    "method": "GET",              // HTTP method
    "requestHeaders": {           // All request headers
      "Authorization": "Bearer token...",
      "Content-Type": "application/json"
    },
    "requestBody": "",            // POST/PUT body if present
    "status": 200,                // HTTP status code
    "statusText": "OK",           // Status text
    "responseHeaders": {          // All response headers
      "content-type": "application/json",
      "cache-control": "no-cache"
    },
    "responseBody": "{...}"       // Complete response body
  }
]
```

### Request Types

- **fetch** - Fetch API requests
- **xhr** - XMLHttpRequest calls
- **document** - HTML page loads
- **script** - JavaScript files
- **stylesheet** - CSS files
- **image** - Image resources
- **font** - Font files
- **media** - Video/audio
- **other** - Other resource types

### Resources JSON Structure

```json
{
  "scripts": [
    "https://example.com/app.js",
    "https://cdn.example.com/vendor.js"
  ],
  "stylesheets": [
    "https://example.com/styles.css"
  ],
  "images": [
    "https://example.com/logo.png",
    "https://example.com/product.jpg"
  ]
}
```

## Advanced Scenarios

### Reverse Engineering an API

1. **Capture the traffic:**
   ```bash
   ./web-capture -url https://app.example.com -html=false -screenshot=false
   ```

2. **Analyze the APIs:**
   - Open `captured_apis.json`
   - Look for endpoints with type "fetch" or "xhr"
   - Examine request/response formats
   - Note authentication headers

3. **Test the endpoints:**
   - Use the captured headers and body formats
   - Replicate requests with curl or Postman

### Downloading All Resources

1. **First, capture the site:**
   ```bash
   ./web-capture -url https://example.com -output site_capture
   ```

2. **Extract resource URLs:**
   ```bash
   # Resources are listed in resources.json
   cat site_capture/resources.json
   ```

3. **Download resources (example script):**
   ```bash
   # You can write a script to download all URLs
   jq -r '.scripts[]' site_capture/resources.json | while read url; do
     wget "$url" -P site_capture/scripts/
   done
   ```

### Capturing Multiple Pages

```bash
# Create a script to capture multiple pages
for page in home about products contact; do
  ./web-capture -url https://example.com/$page -output captures/$page
done
```

### Monitoring API Changes

```bash
# Capture today
./web-capture -url https://api.example.com -output capture_$(date +%Y%m%d)

# Compare with previous capture
diff capture_20251107/captured_apis.json capture_20251108/captured_apis.json
```

### Performance Testing

```bash
# Capture with timing
time ./web-capture -url https://example.com

# Analyze API response times from the JSON timestamps
```

## Troubleshooting

### Chrome/Chromium Not Found

**Problem:** Error about Chrome not being found

**Solutions:**
1. Install Chrome or Chromium:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install chromium-browser
   
   # macOS
   brew install chromium
   ```

2. Or use simple mode:
   ```bash
   ./web-capture -url https://example.com -simple
   ```

### Timeout Errors

**Problem:** Capture times out before completing

**Solutions:**
1. Increase wait time:
   ```bash
   ./web-capture -url https://slow-site.com -wait 15
   ```

2. Check your internet connection

3. For very slow sites, try simple mode:
   ```bash
   ./web-capture -url https://slow-site.com -simple
   ```

### Memory Issues

**Problem:** System runs out of memory on large sites

**Solutions:**
1. Disable screenshot:
   ```bash
   ./web-capture -url https://large-site.com -screenshot=false
   ```

2. Disable resources:
   ```bash
   ./web-capture -url https://large-site.com -resources=false
   ```

3. Use simple mode:
   ```bash
   ./web-capture -url https://large-site.com -simple
   ```

### Empty API Responses

**Problem:** API responses are empty in the JSON

**Possible causes:**
- Response arrived too late (increase wait time)
- Response was binary data (not captured as text)
- Request failed or was blocked

**Solution:**
```bash
./web-capture -url https://example.com -wait 10
```

### SSL/TLS Errors

**Problem:** Certificate verification errors

**Note:** The tool uses Chrome which validates certificates. For development/testing, Chrome's security settings would need to be modified (not recommended).

## Tips and Best Practices

### 1. Start with Simple Mode

If you're just testing, start with simple mode:
```bash
./web-capture -url https://example.com -simple
```

### 2. Adjust Wait Time Based on Site

- Static sites: 2-3 seconds
- Regular sites: 5 seconds (default)
- SPAs/Complex sites: 8-15 seconds

### 3. Organize Your Captures

```bash
# Use descriptive output directories
./web-capture -url https://example.com -output captures/example_homepage_20251107
```

### 4. Use Version Control for Captures

```bash
# Track changes over time
git init my_captures
./web-capture -url https://example.com -output my_captures/capture1
cd my_captures && git add . && git commit -m "Initial capture"
```

### 5. Automate Recurring Captures

```bash
# Create a cron job for daily captures
0 2 * * * cd /path/to/API- && ./web-capture -url https://example.com -output daily_$(date +\%Y\%m\%d)
```

## Use Cases

### Web Scraping
```bash
./web-capture -url https://site-to-scrape.com -simple -output scraped_data
```

### API Documentation
```bash
./web-capture -url https://app.example.com -html=false -screenshot=false
# Then analyze captured_apis.json for endpoint documentation
```

### Competitive Analysis
```bash
./web-capture -url https://competitor.com -output competitor_analysis
```

### Security Testing
```bash
./web-capture -url https://target.com
# Analyze headers, cookies, and API security
```

### Website Archiving
```bash
./web-capture -url https://important-site.com -output archives/$(date +%Y%m%d)
```

## Getting Help

If you encounter issues:

1. Check this guide first
2. Run with `-help` flag for quick reference
3. Check the README.md for overview
4. Create an issue on GitHub with:
   - Command you ran
   - Error message
   - Expected vs actual behavior
