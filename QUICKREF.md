# Quick Reference Card

## Build

```bash
go build -o web-capture capture_cli.go web_capture.go
```

## Basic Commands

| Command | Description |
|---------|-------------|
| `./web-capture -url <URL>` | Capture everything |
| `./web-capture -url <URL> -simple` | Fast HTTP capture (no JS) |
| `./web-capture -url <URL> -wait 10` | Wait 10 seconds before capture |
| `./web-capture -url <URL> -output dir` | Custom output directory |
| `./web-capture -help` | Show all options |

## What Gets Captured

✓ HTML content (including dynamic)  
✓ API requests/responses  
✓ Network traffic  
✓ Headers & cookies  
✓ Resource URLs  
✓ Full page screenshot  

## Output Files

- `page.html` - Full HTML
- `captured_apis.json` - All API calls
- `screenshot.png` - Page screenshot
- `resources.json` - Resource URLs
- `summary.txt` - Capture summary

## Common Use Cases

### API Reverse Engineering
```bash
./web-capture -url https://site.com -html=false -screenshot=false
```

### Website Cloning
```bash
./web-capture -url https://site.com -wait 10
```

### Fast Static Capture
```bash
./web-capture -url https://site.com -simple
```

### SPA/Dynamic Sites
```bash
./web-capture -url https://app.com -wait 15
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-url` | *required* | Target URL |
| `-output` | `capture_<timestamp>` | Output directory |
| `-apis` | `true` | Capture APIs |
| `-html` | `true` | Capture HTML |
| `-resources` | `true` | Extract resources |
| `-screenshot` | `true` | Take screenshot |
| `-wait` | `5` | Wait seconds |
| `-simple` | `false` | Simple HTTP mode |

## Troubleshooting

**Chrome not found?** → Use `-simple` flag  
**Timeout?** → Increase `-wait` time  
**Memory issues?** → Disable `-screenshot`  

## Examples

See `examples.sh` for more examples or run:
```bash
./examples.sh
```

For detailed usage, see `USAGE.md`
