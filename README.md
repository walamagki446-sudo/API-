# Telegram Subscription Management Bot

A professional Telegram bot for managing user subscriptions with pause/resume functionality, automatic expiry tracking, and comprehensive logging.

## Features

- 🔐 **User Authorization** - Authorize users with IP addresses and subscription periods
- ⏸️ **Pause/Resume** - Pause subscriptions and resume from exactly where you left off
- 🚫 **Auto-Deauth** - Automatically kick users from group on deauth or expiry
- 📊 **Status Tracking** - Real-time subscription status and remaining days
- 📝 **Comprehensive Logging** - All events logged to a dedicated channel
- ⏰ **Expiry Notifications** - Automatic notifications when subscriptions expire (sent twice with warnings)

## Commands

- `/auth @username IP days` - Authorize a user with IP and subscription days
- `/deauth @username IP` - Deauthorize user and kick from group
- `/pause @username IP` - Pause subscription (stops day counting)
- `/resume @username IP` - Resume subscription from paused state
- `/list` - List all active subscriptions
- `/status @username IP` - Check detailed subscription status
- `/start` - Display help message

## Setup

1. **Install Go** (version 1.21 or higher)

2. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd API-
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Configure environment variables**
   
   Copy `.env.example` to `.env` and fill in your values:
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` with your configuration:
   ```
   TELEGRAM_BOT_TOKEN=your_bot_token_here
   ADMIN_ID=your_telegram_user_id
   GROUP_CHAT_ID=group_chat_id_to_kick_from
   LOG_CHAT_ID=log_channel_id
   ```

5. **Run the bot**
   ```bash
   go run bot.go
   ```

## Usage Examples

### Authorize a User
```
/auth @john 192.168.1.1 30
```
Authorizes user @john with IP 192.168.1.1 for 30 days.

### Pause a Subscription
```
/pause @john 192.168.1.1
```
If a user has 18 days left out of 30, pausing will freeze it at 18 days.

### Resume a Subscription
```
/resume @john 192.168.1.1
```
Continues counting from the 18 days that were left when paused.

### Deauthorize a User
```
/deauth @john 192.168.1.1
```
Deactivates subscription, kicks user from group, and logs the event.

## How It Works

### Day Tracking with Pause/Resume

- When authorized, a subscription starts with the specified number of days
- Days are counted down automatically (checked hourly)
- When paused, the current days remaining are saved and counting stops
- When resumed, counting continues from the saved days remaining
- Expiry happens when days remaining reaches 0

### Automatic Actions

- **On Deauth**: User is kicked from the group (if GROUP_CHAT_ID is configured)
- **On Expiry**: User is kicked from the group and notifications sent to log channel (twice with warnings)
- **All Events**: Logged to the log channel with timestamp and details

### Data Persistence

Subscriptions are stored in `subscriptions.json` and persist across bot restarts.

## Configuration

### Required Environment Variables

- `TELEGRAM_BOT_TOKEN` - Your Telegram bot token from @BotFather

### Optional Environment Variables

- `ADMIN_ID` - Telegram user ID who can use commands (if not set, anyone can use)
- `GROUP_CHAT_ID` - Chat ID where users will be kicked on deauth/expiry
- `LOG_CHAT_ID` - Chat ID where all events will be logged

## Log Channel Events

The bot logs the following events to the log channel:

- ✅ **AUTH** - When a user is authorized
- ⚠️ **DEAUTH** - When a user is deauthorized
- ⏸️ **PAUSE** - When a subscription is paused
- ▶️ **RESUME** - When a subscription is resumed
- ⚠️ **SUBSCRIPTION EXPIRED** - When a subscription expires (sent twice)

## Technical Details

- Written in Go
- Uses `go-telegram-bot-api/telegram-bot-api/v5` for Telegram integration
- JSON-based storage for persistence
- Hourly background task for expiry checking
- Thread-safe subscription management

## License

MIT License