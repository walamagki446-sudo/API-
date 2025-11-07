#!/bin/bash

# Example Usage Scripts for Web Capture Tool

echo "═══════════════════════════════════"
echo "   WEB CAPTURE TOOL - EXAMPLES"
echo "═══════════════════════════════════"
echo ""

# Build the tool first
echo "Building the tool..."
go build -o web-capture capture_cli.go web_capture.go
echo "✓ Build complete"
echo ""

# Example 1: Basic capture
echo "Example 1: Capture everything from a website"
echo "Command: ./web-capture -url https://example.com"
echo ""

# Example 2: Custom output
echo "Example 2: Capture with custom output directory"
echo "Command: ./web-capture -url https://example.com -output my_site_capture"
echo ""

# Example 3: Only APIs
echo "Example 3: Only capture API requests (no HTML/screenshot)"
echo "Command: ./web-capture -url https://api.example.com -html=false -screenshot=false"
echo ""

# Example 4: Long wait time for SPAs
echo "Example 4: Capture SPA with longer wait time"
echo "Command: ./web-capture -url https://app.example.com -wait 10"
echo ""

# Example 5: Simple mode
echo "Example 5: Simple HTTP capture (fast, no JavaScript)"
echo "Command: ./web-capture -url https://example.com -simple"
echo ""

# Example 6: E-commerce site
echo "Example 6: Capture e-commerce site with products"
echo "Command: ./web-capture -url https://shop.example.com -wait 8 -output shop_data"
echo ""

# Example 7: API reverse engineering
echo "Example 7: Focus on API endpoints only"
echo "Command: ./web-capture -url https://website.com -html=false -screenshot=false -resources=false"
echo ""

echo "═══════════════════════════════════"
echo "To run any example, copy the command after 'Command:'"
echo "═══════════════════════════════════"
