# Quick Start Guide

## Prerequisites
- Go 1.21 or higher
- A Telegram bot token (get from [@BotFather](https://t.me/BotFather))
- Your Telegram user ID (get from [@userinfobot](https://t.me/userinfobot))

## Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd API-
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure the bot**
   
   Create a `.env` file:
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` with your settings:
   ```env
   TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ADMIN_ID=123456789
   GROUP_CHAT_ID=-1001234567890
   LOG_CHAT_ID=-1009876543210
   ```

4. **Run the bot**
   ```bash
   go run bot.go
   ```
   
   Or build and run:
   ```bash
   go build -o subscription-bot bot.go
   ./subscription-bot
   ```

## First Steps

1. **Start a chat with your bot**
   - Open Telegram and search for your bot
   - Send `/start` to see available commands

2. **Add bot to your group**
   - Add the bot to your group where you want to kick users
   - Make the bot an administrator with "Ban users" permission

3. **Create a log channel**
   - Create a new channel for logs
   - Add the bot as an administrator
   - Forward a message from the channel to [@userinfobot](https://t.me/userinfobot) to get the channel ID

4. **Test the bot**
   ```
   /auth @testuser 192.168.1.1 30
   /status @testuser 192.168.1.1
   /list
   ```

## Common Use Cases

### Authorize a new user for 30 days
```
/auth @john 192.168.1.1 30
```

### Temporarily pause a subscription
```
/pause @john 192.168.1.1
```

### Resume after pause
```
/resume @john 192.168.1.1
```

### Remove a user immediately
```
/deauth @john 192.168.1.1
```

### Check all active subscriptions
```
/list
```

## Troubleshooting

### Bot doesn't respond
- Check if `TELEGRAM_BOT_TOKEN` is correct
- Check if you're the admin (verify `ADMIN_ID`)
- Check bot logs for errors

### Can't kick users
- Make sure bot is admin in the group
- Verify `GROUP_CHAT_ID` is correct (should be negative for groups)
- Bot needs "Ban users" permission

### No log messages
- Check if `LOG_CHAT_ID` is set
- Make sure bot is admin in the log channel
- Channel ID should be negative and start with -100

### Getting Chat/Channel IDs

**For Groups:**
1. Add bot to group
2. Send a message in the group
3. Visit: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
4. Look for `"chat":{"id":-1234567890}` in the response

**For Channels:**
1. Add bot to channel as admin
2. Forward a message from the channel to [@userinfobot](https://t.me/userinfobot)
3. The bot will show the channel ID

**For Your User ID:**
1. Message [@userinfobot](https://t.me/userinfobot)
2. It will reply with your user ID

## Running in Production

### Using systemd (Linux)

Create `/etc/systemd/system/subscription-bot.service`:
```ini
[Unit]
Description=Telegram Subscription Bot
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/API-
ExecStart=/path/to/API-/subscription-bot
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable subscription-bot
sudo systemctl start subscription-bot
sudo systemctl status subscription-bot
```

### Using Docker

Create `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o subscription-bot bot.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/subscription-bot .
COPY --from=builder /app/.env .
CMD ["./subscription-bot"]
```

Build and run:
```bash
docker build -t subscription-bot .
docker run -d --name subscription-bot subscription-bot
```

## Support

For issues or questions, check:
- README.md for detailed setup instructions
- USAGE.md for command documentation and examples
- GitHub Issues for bug reports

## Security Notes

- Never commit `.env` file (already in `.gitignore`)
- Keep your bot token secret
- Only trusted users should have `ADMIN_ID`
- Regularly backup `subscriptions.json`
