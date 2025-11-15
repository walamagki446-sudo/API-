#!/bin/bash
# Red Dead Redemption 2 Site Finder - Unix Launcher
# ==================================================

echo ""
echo "========================================================"
echo "  RED DEAD REDEMPTION 2 - SITE FINDER LAUNCHER"
echo "========================================================"
echo ""

# Check if binary exists
if [ ! -f "rdr2_site_finder" ]; then
    echo "ERROR: rdr2_site_finder not found!"
    echo ""
    echo "Please build the tool first:"
    echo "  go build -o rdr2_site_finder rdr2_site_finder.go"
    echo ""
    exit 1
fi

# Make binary executable if needed
chmod +x rdr2_site_finder

echo "Starting site finder..."
echo ""
echo "TIP: Results will be saved to files when complete:"
echo "  - rdr2_sites_YYYYMMDD_HHMMSS.json (structured data)"
echo "  - rdr2_sites_YYYYMMDD_HHMMSS.txt  (readable format)"
echo ""
echo "Press Ctrl+C to stop the search at any time."
echo ""
echo "========================================================"
echo ""

# Run the site finder
./rdr2_site_finder

echo ""
echo "========================================================"
echo "  SEARCH COMPLETED"
echo "========================================================"
echo ""
echo "Check the generated files in this directory."
echo ""
