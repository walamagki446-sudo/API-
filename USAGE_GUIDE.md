# Usage Guide - Red Dead Redemption 2 Site Finder

## Quick Start

### 1. Build the Tool
```bash
go build -o rdr2_site_finder rdr2_site_finder.go
```

On Windows, this will create `rdr2_site_finder.exe`

### 2. Run the Tool
```bash
./rdr2_site_finder
```

On Windows:
```cmd
rdr2_site_finder.exe
```

## What to Expect

When you run the tool, you'll see:

### Startup Banner
```
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║     RED DEAD REDEMPTION 2 - SITE FINDER v1.0            ║
║                                                           ║
║     🎮 Multi-Region Game Store Discovery                 ║
║     💰 Automatic Price Extraction                        ║
║     ⚡ Instant Delivery Detection                        ║
║     🌍 30+ Countries Worldwide                           ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
```

### Real-Time Progress
The tool displays a live progress bar:
```
⏱️  Time: 45s | 🌍 Region: Germany | 📊 Found: 1247 sites | 💰 With Price: 892 | ⚡ Instant: 634 | Rate: 27.7/s
```

- **Time**: How long the search has been running
- **Region**: Currently searching region
- **Found**: Total unique sites discovered
- **With Price**: Sites where price was successfully extracted
- **Instant**: Sites offering instant/digital delivery
- **Rate**: Sites discovered per second

### Live Results
Each discovered site is displayed immediately:
```
⚡ [1247] 🌍 Germany      | 💰 €29.99          | https://www.instant-gaming.com/en/2318-buy-red-dead-redemption-2/
  [1248] 🌍 India        | 💰 ₹2,499          | https://www.gamestheshop.com/red-dead-redemption-2
⚡ [1249] 🌍 Netherlands | 💰 €34.99          | https://www.bol.com/nl/p/red-dead-redemption-2-pc/9200000108854321/
```

- **⚡** = Site offers instant delivery
- **Number** = Site count
- **🌍 Region** = Country where site was found
- **💰 Price** = Extracted price (if available)
- **URL** = Direct link to the product page

### Final Summary
After completion (or reaching 5,000 sites), you'll see:
```
╔═══════════════════════════════════════════════════════════╗
║                    FINAL RESULTS                         ║
╚═══════════════════════════════════════════════════════════╝

📊 Total Sites Found:        5000
💰 Sites with Prices:        3421 (68.4%)
⚡ Instant Delivery Sites:   2789 (55.8%)
⏱️  Total Time:               180.52 seconds
🚀 Average Rate:             27.70 sites/second

🎯 TOP INSTANT DELIVERY SITES WITH PRICES:
═══════════════════════════════════════════════════════════

1. 💰 $29.99
   🌐 https://www.instant-gaming.com/en/2318-buy-red-dead-redemption-2/
   🌍 Region: United States | Source: Google

2. 💰 €34.99
   🌐 https://www.g2a.com/red-dead-redemption-2-steam-key-global
   🌍 Region: Germany | Source: Google
```

## Output Files

Two files are automatically created:

### 1. JSON File: `rdr2_sites_YYYYMMDD_HHMMSS.json`
Contains complete structured data for programmatic use:
```json
{
  "search_stats": {
    "total_sites": 5000,
    "with_price": 3421,
    "instant_delivery": 2789,
    "duration_seconds": 180.52,
    "timestamp": "2025-11-15T12:30:45Z"
  },
  "sites": [
    {
      "URL": "https://www.instant-gaming.com/...",
      "Title": "Red Dead Redemption 2 - Steam Key",
      "Price": "$29.99",
      "Region": "United States",
      "InstantDeliv": true,
      "Source": "Google",
      "Timestamp": "2025-11-15T12:28:12Z"
    }
  ]
}
```

### 2. Text File: `rdr2_sites_YYYYMMDD_HHMMSS.txt`
Human-readable format perfect for CMD viewing:
```
═══════════════════════════════════════════════════════════
   RED DEAD REDEMPTION 2 - SITE FINDER RESULTS
   Generated: 2025-11-15 12:30:45
═══════════════════════════════════════════════════════════

Total Sites: 5000
With Prices: 3421
Instant Delivery: 2789

═══════════════════════════════════════════════════════════

1. PRICE: $29.99 [⚡ INSTANT DELIVERY]
   URL: https://www.instant-gaming.com/...
   Region: United States | Source: Google

2. PRICE: €34.99 [⚡ INSTANT DELIVERY]
   URL: https://www.g2a.com/...
   Region: Germany | Source: Google
```

## Viewing Results in CMD

### View the Text File
```cmd
type rdr2_sites_20251115_123045.txt
```

Or with pagination:
```cmd
more rdr2_sites_20251115_123045.txt
```

### Search for Specific Prices
```cmd
findstr /C:"$" rdr2_sites_20251115_123045.txt
```

### Filter Instant Delivery Only
```cmd
findstr /C:"INSTANT DELIVERY" rdr2_sites_20251115_123045.txt
```

### Count Total Sites
```cmd
find /C "URL:" rdr2_sites_20251115_123045.txt
```

## Tips for Best Results

### 1. Internet Connection
- Use a stable, fast connection
- VPN might help access region-restricted content
- Avoid running during peak network usage

### 2. Avoiding Rate Limits
- The tool has built-in delays to minimize rate limiting
- If you get rate limited, wait a few minutes and run again
- Consider using a different network or VPN

### 3. Maximizing Discoveries
- Let the tool run to completion (typically 3-5 minutes)
- Run at different times to discover time-sensitive deals
- The tool automatically rotates user agents

### 4. Processing Results
- Use the JSON file for data analysis/automation
- Use the TXT file for manual review in CMD
- Both files are timestamped for easy organization

## Troubleshooting

### "Too many requests" or Rate Limiting
**Solution**: The tool has built-in delays. If you still hit limits:
- Wait 5-10 minutes before running again
- Use a VPN to change your IP address
- Run during off-peak hours

### No Results or Very Few Sites
**Possible causes**:
- Network connectivity issues
- Search engines temporarily blocking automated queries
- Firewall or antivirus blocking requests

**Solution**:
- Check internet connection
- Temporarily disable antivirus/firewall
- Try running as administrator (Windows)

### Price Not Detected
**Why**: Some sites use JavaScript to load prices dynamically, which this tool cannot parse (requires full browser automation)

**Workaround**: The URL is provided - you can manually visit high-value sites

### Low Instant Delivery Detection
**Why**: Sites use various terms for instant delivery

**Note**: The tool searches for 12+ variations of instant delivery terms

## Advanced Usage

### Analyzing JSON Output with jq (Linux/Mac)
```bash
# Get all prices
jq '.sites[].Price' rdr2_sites_*.json

# Filter instant delivery sites
jq '.sites[] | select(.InstantDeliv == true)' rdr2_sites_*.json

# Count sites by region
jq '.sites | group_by(.Region) | map({region: .[0].Region, count: length})' rdr2_sites_*.json

# Get cheapest price (requires cleanup)
jq '.sites[] | select(.Price != "") | .Price' rdr2_sites_*.json | sort
```

### Analyzing JSON Output with PowerShell (Windows)
```powershell
# Load JSON
$data = Get-Content rdr2_sites_*.json | ConvertFrom-Json

# View stats
$data.search_stats

# Get all instant delivery sites
$data.sites | Where-Object { $_.InstantDeliv -eq $true }

# Group by region
$data.sites | Group-Object Region | Select-Object Name, Count

# Filter by price
$data.sites | Where-Object { $_.Price -like "*$*" }
```

## Performance Expectations

- **Time to 5,000 sites**: 3-5 minutes (typical)
- **Discovery rate**: 20-35 sites/second
- **Memory usage**: ~50-100 MB
- **Network traffic**: Moderate (depends on connection)
- **CPU usage**: Low to moderate

## Security & Privacy

- The tool makes standard HTTP requests (like a web browser)
- No data is sent to external servers (except search engines)
- All results are stored locally
- User-agent rotation prevents tracking
- No cookies or persistent data stored

## Legal Considerations

✅ **Allowed**:
- Searching public search engines
- Viewing publicly accessible websites
- Extracting publicly displayed prices
- Personal use and research

⚠️ **Respect**:
- robots.txt files (for direct store searches)
- Rate limits and API restrictions
- Terms of Service of websites
- Copyright and trademark laws

❌ **Avoid**:
- Commercial use without permission
- Overwhelming servers with requests
- Bypassing paywalls or authentication
- Redistributing copyrighted content

## Support & Contribution

- Found a bug? Open an issue on GitHub
- Have a feature request? Submit a pull request
- Want to improve search results? Contribute to keyword lists
- Need help? Check the README or create a discussion

---

**Happy searching! 🎮💰⚡**
