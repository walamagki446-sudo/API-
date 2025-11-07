# Zalando Sweden Bot

A Telegram bot for Zalando Sweden with account checking and auto-purchasing (autohitter) capabilities.

## Features

### 🔐 Account Checker
- Verify Zalando Sweden account credentials
- Mass check multiple accounts from file
- Elevated risk bypass
- Fast CPM with proxy support
- Skipped accounts tracking

### 🎯 Autohitter (/hit command)
Automated purchase system for Zalando Sweden with human-like behavior:

**Flow:**
1. Login with account credentials
2. (Optional) Browse random products for ~10 seconds
3. Add product to cart from provided URL
4. Navigate through checkout (address → delivery → payment)
5. Select INSTABOX/BUDBEE pickup point (not closest)
6. Select Faktura (invoice) payment method
7. Complete order and extract order ID

**Features:**
- Realistic human delays and behavior
- Anti-detection measures (random User-Agents, delays, headers)
- INSTABOX/BUDBEE pickup point preference
- Configurable phone number
- Proxy rotation support
- Comprehensive error reporting

## Commands

### Basic Commands
- `/start` - Show welcome message and available commands
- `/check email:password` - Verify a single account
- `/mass` - Start mass checking (send .txt file with email:password combos)
- `/stop` - Stop ongoing mass check
- `/status` - View real-time stats of mass check
- `/proxies` - View proxy info
- `/debug` - Download debug logs

### Autohitter Command
```
/hit email:password product_url size [browse] [phone_number]
```

**Arguments:**
- `email:password` - Account credentials (required)
- `product_url` - Full Zalando product URL (required)
- `size` - Product size: S, M, L, XL, XXL, or ONE_SIZE (required)
- `browse` - Optional: Enable browse mode (simulates human browsing for 10s)
- `phone_number` - Optional: Custom phone number (default: 0767541615)

**Examples:**
```
# Basic usage
/hit test@email.com:password123 https://www.zalando.se/product-link.html M

# With browse mode
/hit test@email.com:password123 https://www.zalando.se/product-link.html L browse

# With custom phone number
/hit test@email.com:password123 https://www.zalando.se/product-link.html XL 0701234567

# All options
/hit test@email.com:password123 https://www.zalando.se/product-link.html M browse 0701234567
```

**Valid Sizes:**
- `S` - Small
- `M` - Medium
- `L` - Large
- `XL` - Extra Large
- `XXL` - Double Extra Large
- `ONE_SIZE` - One size fits all

## Setup

### Prerequisites
- Go 1.19 or higher
- Telegram Bot Token (from @BotFather)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/walamagki446-sudo/API-.git
cd API-
```

2. Install dependencies:
```bash
go mod download
```

3. Create `.env` file:
```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
ADMIN_ID=your_telegram_user_id
```

4. (Optional) Create `proxies.txt` with your proxies:
```
# Format: host:port:username:password
# Or: host:port (for unauthenticated proxies)
# Or: socks5://host:port
proxy1.example.com:8080:user:pass
proxy2.example.com:8080:user:pass
```

5. Build and run:
```bash
go build -o zalando-bot
./zalando-bot
```

## Configuration

### Environment Variables
- `TELEGRAM_BOT_TOKEN` - Your Telegram bot token (required)
- `ADMIN_ID` - Your Telegram user ID for admin access (optional)

### Proxy Support
The bot supports multiple proxy formats:
- SOCKS5 with auth: `socks5://user:pass@host:port`
- SOCKS5 without auth: `socks5://host:port` or `host:port`
- HTTP with auth: `http://user:pass@host:port`
- Format with colon separation: `host:port:username:password`

## Technical Details

### Autohitter Implementation
- Uses captured API endpoints from real Zalando checkout flow
- GraphQL mutations for add-to-cart operations
- Proper address and delivery location payloads
- Purchase session management
- Order ID extraction from multiple response formats

### Anti-Detection
- Random User-Agent rotation
- Human-like delays (500-2000ms between actions)
- Realistic browsing patterns
- Cookie jar management
- Proper HTTP headers and referrers

### Supported Payment Methods
- Faktura (Invoice/BNPL)
- Automatic detection of payment method availability

### Delivery Options
- INSTABOX pickup points (preferred)
- BUDBEE pickup points (preferred)
- Automatic selection (skips closest point for realism)

## Disclaimer

This tool is for educational purposes only. Use at your own risk. The authors are not responsible for any misuse or violation of Zalando's terms of service.

## License

MIT