# API-

## Repository Contents

This repository contains multiple tools and utilities:

### 1. Red Dead Redemption 2 Site Finder (`rdr2_site_finder.go`)
A comprehensive tool for discovering game sites selling Red Dead Redemption 2 across 30+ regions worldwide.

**Features:**
- 🌍 Multi-region search (India, Asia, Germany, Netherlands, USA, UK, and 24+ more)
- 💰 Automatic price extraction in multiple currencies
- ⚡ Instant delivery detection
- 🔍 Deep search using Google, Bing, and DuckDuckGo
- 📊 Real-time progress tracking in CMD
- 🎯 Targets 5,000+ unique sites
- 💾 JSON and TXT output formats

**Quick Start:**
```bash
go build -o rdr2_site_finder rdr2_site_finder.go
./rdr2_site_finder
```

See [RDR2_SITE_FINDER_README.md](RDR2_SITE_FINDER_README.md) and [USAGE_GUIDE.md](USAGE_GUIDE.md) for detailed documentation.

### 2. Zalando Account Checker (`zalando.go`)
A Telegram bot for checking Zalando Sweden accounts.

**Features:**
- Account validation
- Mass checking
- Proxy support
- Elevated risk bypass

---

## Building & Running

### Prerequisites
- Go 1.21 or higher

### Build All Tools
```bash
# Build RDR2 Site Finder
go build -o rdr2_site_finder rdr2_site_finder.go

# Build Zalando Checker
go build -o zalando zalando.go
```

### Run with Helper Scripts

**Windows:**
```cmd
run_site_finder.bat
```

**Linux/Mac:**
```bash
./run_site_finder.sh
```

---

## Documentation

- [RDR2 Site Finder README](RDR2_SITE_FINDER_README.md) - Complete feature documentation
- [Usage Guide](USAGE_GUIDE.md) - Step-by-step usage instructions

---

## License

MIT License - See individual files for details