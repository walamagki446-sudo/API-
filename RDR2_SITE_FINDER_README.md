# Red Dead Redemption 2 Site Finder

A powerful Go-based tool that searches for Red Dead Redemption 2 game sites across multiple regions worldwide, with automatic price extraction and instant delivery detection.

## Features

- 🌍 **Multi-Region Search**: Searches across 30+ countries including India, Asia, Germany, Netherlands, USA, UK, and more
- 💰 **Automatic Price Extraction**: Detects and extracts prices in multiple currencies (USD, EUR, GBP, JPY, INR, etc.)
- ⚡ **Instant Delivery Detection**: Identifies sites offering instant/digital delivery
- 🔍 **Deep Search**: Uses Google, Bing, and DuckDuckGo with advanced search queries
- 📊 **Real-time Progress**: Live progress display in CMD showing stats and discoveries
- 🎯 **Target-Oriented**: Aims to discover 5,000+ unique sites
- 🏪 **Known Store Support**: Directly searches major game stores (G2A, Kinguin, Steam, Epic, etc.)
- 💾 **Multiple Output Formats**: Results saved in both JSON and TXT formats

## Installation

### Prerequisites
- Go 1.21 or higher

### Build
```bash
go mod tidy
go build -o rdr2_site_finder rdr2_site_finder.go
```

## Usage

### Run the Site Finder
```bash
./rdr2_site_finder
```

On Windows:
```cmd
rdr2_site_finder.exe
```

### Output

The tool will:
1. Display a real-time progress bar with stats
2. Show each discovered site immediately with price and region
3. Save results to two files:
   - `rdr2_sites_YYYYMMDD_HHMMSS.json` - Complete JSON data
   - `rdr2_sites_YYYYMMDD_HHMMSS.txt` - Human-readable text format

### Example Output

```
⏱️  Time: 45s | 🌍 Region: Germany       | 📊 Found: 1247 sites | 💰 With Price: 892 | ⚡ Instant: 634 | Rate: 27.7/s

⚡ [1247] 🌍 Germany      | 💰 €29.99          | https://www.instant-gaming.com/en/2318-buy-red-dead-redemption-2/
  [1248] 🌍 India        | 💰 ₹2,499          | https://www.gamestheshop.com/red-dead-redemption-2
⚡ [1249] 🌍 Netherlands | 💰 €34.99          | https://www.bol.com/nl/p/red-dead-redemption-2-pc/9200000108854321/
```

## Search Strategy

The tool employs multiple search strategies:

1. **Keyword Variations**: 
   - "Red Dead Redemption 2", "RDR2", "Red Dead Redemption II", "Red Dead 2", "RDR 2", "Red Dead Online"

2. **Search Terms**:
   - instant delivery, digital download, cd key, steam key, game key, cheap, discount, sale, etc.

3. **Search Engines**:
   - Google (with region-specific domains)
   - Bing
   - DuckDuckGo

4. **Direct Store Searches**:
   - 30+ known game stores and marketplaces

5. **Regional Targeting**:
   - Each search is performed for specific regions to discover local stores

## Configuration

The tool uses hardcoded configuration optimized for comprehensive searching:

- **Concurrent Searches**: 20 simultaneous search threads
- **Target Sites**: 5,000 unique sites
- **Request Delays**: 100-300ms between requests to avoid rate limiting
- **Timeout**: 15s for search results, 10s for site analysis
- **User Agent Rotation**: 5+ different user agents

## Output Format

### JSON Output
```json
{
  "search_stats": {
    "total_sites": 5000,
    "with_price": 3421,
    "instant_delivery": 2789,
    "duration_seconds": 180.5,
    "timestamp": "2025-01-15T10:30:45Z"
  },
  "sites": [
    {
      "URL": "https://example.com/rdr2",
      "Title": "Red Dead Redemption 2 - PC Game",
      "Price": "$29.99",
      "Region": "United States",
      "InstantDeliv": true,
      "Source": "Google",
      "Timestamp": "2025-01-15T10:28:12Z"
    }
  ]
}
```

### TXT Output
```
═══════════════════════════════════════════════════════════
   RED DEAD REDEMPTION 2 - SITE FINDER RESULTS
   Generated: 2025-01-15 10:30:45
═══════════════════════════════════════════════════════════

Total Sites: 5000
With Prices: 3421
Instant Delivery: 2789

═══════════════════════════════════════════════════════════

1. PRICE: $29.99 [⚡ INSTANT DELIVERY]
   URL: https://example.com/rdr2
   Region: United States | Source: Google

2. PRICE: €34.99 [⚡ INSTANT DELIVERY]
   URL: https://instant-gaming.com/en/2318-buy-red-dead-redemption-2/
   Region: Germany | Source: Google
```

## Known Game Stores

The tool automatically searches these major game stores:
- G2A.com
- Kinguin.net
- CDKeys.com
- Instant-Gaming.com
- Green Man Gaming
- Fanatical
- Humble Bundle
- GOG.com
- Steam
- Epic Games
- Gamivo
- Eneba
- GamersGate
- Gamesplanet
- DLGamer
- And 15+ more...

## Performance

Expected performance on a typical system:
- **Discovery Rate**: 20-30 sites per second
- **Target Time**: 3-5 minutes to reach 5,000 sites
- **Memory Usage**: ~50-100 MB
- **Network Usage**: Moderate (depends on connection)

## Troubleshooting

### Rate Limiting
If you encounter rate limiting:
- The tool automatically handles retries
- Delays are built in between requests
- Use a VPN or proxy if needed

### Low Results
If fewer sites are found:
- Check your internet connection
- Some search engines may temporarily block automated queries
- Try running at different times of day

### No Prices Found
If prices aren't being extracted:
- Sites may use JavaScript to load prices (requires browser automation)
- Some stores may require user login
- Regional restrictions may apply

## Legal Notice

This tool is for educational and research purposes only. Ensure you comply with:
- Search engine Terms of Service
- Website robots.txt files
- Rate limiting and scraping policies
- Local laws regarding web scraping

## License

MIT License - See repository for details

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues for bugs and feature requests.
