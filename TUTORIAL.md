# Web Capture Tool - Step-by-Step Tutorial

## Introduction

This tutorial will guide you through using the Web Capture Tool to capture everything from a website. By the end, you'll know how to:

- Capture complete websites
- Extract API endpoints
- Analyze network traffic
- Save resources
- Use different capture modes

## Prerequisites

- Go 1.21 or higher installed
- Chrome or Chromium browser (for full mode)
- Basic command-line knowledge

## Tutorial Steps

### Step 1: Build the Tool

First, build the executable:

```bash
cd /path/to/API-
go build -o web-capture capture_cli.go web_capture.go
```

You should now have a `web-capture` executable.

### Step 2: Basic Capture

Let's start with a simple capture:

```bash
./web-capture -url https://example.com
```

**What happens:**
1. Chrome opens in headless mode
2. Navigates to the URL
3. Waits 5 seconds (default)
4. Captures:
   - HTML content
   - All network requests
   - Page screenshot
   - Resource URLs
5. Saves to `capture_<timestamp>/`

**Check the output:**

```bash
ls capture_*/
# You should see:
# - page.html
# - captured_apis.json
# - screenshot.png
# - resources.json
# - summary.txt
```

### Step 3: View the Captured Data

**View the summary:**
```bash
cat capture_*/summary.txt
```

**Check captured APIs:**
```bash
# Pretty print the JSON
cat capture_*/captured_apis.json | jq '.'

# Count API requests
cat capture_*/captured_apis.json | jq 'length'

# List all unique endpoints
cat capture_*/captured_apis.json | jq '.[].url' | sort | uniq
```

**Open the HTML:**
```bash
# On Linux
xdg-open capture_*/page.html

# On macOS
open capture_*/page.html

# On Windows
start capture_*/page.html
```

### Step 4: Capture with Custom Settings

**Wait longer for slow sites:**
```bash
./web-capture -url https://slow-website.com -wait 10
```

**Use custom output directory:**
```bash
./web-capture -url https://example.com -output my_capture
ls my_capture/
```

**Only capture APIs (no HTML/screenshot):**
```bash
./web-capture -url https://api-site.com \
  -html=false \
  -screenshot=false
```

### Step 5: Simple Mode (Fast)

For static sites or when Chrome isn't available:

```bash
./web-capture -url https://example.com -simple
```

This uses basic HTTP requests (no JavaScript rendering) and is much faster.

### Step 6: Extract Specific Information

**Find all API endpoints:**
```bash
cat capture_*/captured_apis.json | \
  jq -r '.[] | select(.type=="fetch" or .type=="xhr") | .url' | \
  sort | uniq
```

**Find all POST requests:**
```bash
cat capture_*/captured_apis.json | \
  jq '.[] | select(.method=="POST")'
```

**Find requests with specific headers:**
```bash
cat capture_*/captured_apis.json | \
  jq '.[] | select(.requestHeaders.Authorization != null)'
```

**Extract all script URLs:**
```bash
cat capture_*/resources.json | jq -r '.scripts[]'
```

### Step 7: Real-World Example - E-commerce Site

Let's capture an e-commerce product page:

```bash
# Capture product page with longer wait for images/reviews
./web-capture \
  -url https://shop.example.com/product/xyz \
  -wait 8 \
  -output captures/product_xyz
```

**Analyze the capture:**

```bash
cd captures/product_xyz

# Check what APIs were called
cat captured_apis.json | jq -r '.[] | .url' | grep -i api

# Look for price API
cat captured_apis.json | jq '.[] | select(.url | contains("price"))'

# Find product data API
cat captured_apis.json | jq '.[] | select(.url | contains("product"))'

# Extract images
cat resources.json | jq -r '.images[]'
```

### Step 8: Real-World Example - API Reverse Engineering

Capture a dashboard to understand its API:

```bash
./web-capture \
  -url https://dashboard.example.com \
  -wait 10 \
  -output api_analysis
```

**Analyze the APIs:**

```bash
cd api_analysis

# Get all unique API endpoints
cat captured_apis.json | \
  jq -r '.[] | select(.type=="fetch") | .url' | \
  sort | uniq > endpoints.txt

# View request/response for each endpoint
cat captured_apis.json | jq '.[] | {
  url: .url,
  method: .method,
  status: .status,
  request: .requestBody,
  response: .responseBody
}'

# Find authentication headers
cat captured_apis.json | \
  jq '.[] | .requestHeaders | select(.Authorization != null)'
```

### Step 9: Real-World Example - SPA Analysis

For Single Page Applications:

```bash
./web-capture \
  -url https://app.example.com \
  -wait 15 \
  -output spa_capture
```

**Why wait longer?**
- SPAs load JavaScript first
- Then make API calls
- Then render content
- 15 seconds ensures everything loads

### Step 10: Batch Captures

Capture multiple pages:

```bash
#!/bin/bash
# Save as batch_capture.sh

urls=(
  "https://example.com/page1"
  "https://example.com/page2"
  "https://example.com/page3"
)

for url in "${urls[@]}"; do
  name=$(echo "$url" | sed 's/https:\/\///' | sed 's/\//_/g')
  ./web-capture -url "$url" -output "captures/$name"
  echo "Captured: $url"
done
```

Run it:
```bash
chmod +x batch_capture.sh
./batch_capture.sh
```

## Common Patterns

### Pattern 1: Monitoring API Changes

```bash
# Daily capture
./web-capture -url https://api.example.com \
  -output daily/$(date +%Y%m%d) \
  -html=false -screenshot=false

# Compare with yesterday
diff daily/$(date -d yesterday +%Y%m%d)/captured_apis.json \
     daily/$(date +%Y%m%d)/captured_apis.json
```

### Pattern 2: Extracting All Resources

```bash
# Capture site
./web-capture -url https://example.com -output site

# Download all scripts
mkdir -p site/downloaded/scripts
cat site/resources.json | jq -r '.scripts[]' | while read url; do
  wget "$url" -P site/downloaded/scripts/
done

# Download all stylesheets
mkdir -p site/downloaded/css
cat site/resources.json | jq -r '.stylesheets[]' | while read url; do
  wget "$url" -P site/downloaded/css/
done
```

### Pattern 3: Building API Documentation

```bash
# Capture site
./web-capture -url https://app.example.com -output api_docs

# Generate markdown documentation
cat api_docs/captured_apis.json | jq -r '
  .[] | 
  select(.type=="fetch") | 
  "## \(.method) \(.url)\n\n**Request:**\n```json\n\(.requestBody)\n```\n\n**Response:**\n```json\n\(.responseBody)\n```\n"
' > api_documentation.md
```

## Troubleshooting

### Chrome Not Found

```bash
# Use simple mode instead
./web-capture -url https://example.com -simple
```

### Timeout Errors

```bash
# Increase wait time
./web-capture -url https://slow-site.com -wait 20
```

### Large Files

```bash
# Disable screenshot to save space
./web-capture -url https://large-site.com -screenshot=false
```

## Next Steps

Now you know how to:
- ✓ Capture websites completely
- ✓ Extract API endpoints
- ✓ Analyze network traffic
- ✓ Save and process resources
- ✓ Use different capture modes
- ✓ Automate captures

**Experiment with:**
- Different websites
- Various configuration options
- Batch processing
- API analysis
- Resource extraction

## Additional Resources

- README.md - Full documentation
- USAGE.md - Detailed usage guide
- QUICKREF.md - Quick reference
- examples.sh - More examples

## Tips

1. **Start simple** - Use simple mode first to test
2. **Adjust wait times** - Dynamic sites need longer waits
3. **Check output** - Always verify captures worked
4. **Use jq** - For analyzing JSON data
5. **Automate** - Create scripts for repeated tasks

## Conclusion

You now have a powerful tool for capturing everything from websites. Use it responsibly and respect website terms of service and robots.txt files.

Happy capturing! 🚀
