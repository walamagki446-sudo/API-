# Subscription Management Bot - Implementation Summary

## Overview
This bot provides a complete subscription management system for Telegram with advanced features like pause/resume, automatic expiry tracking, and comprehensive logging.

## Core Commands

### 1. /auth @username IP days
Authorizes a user with an IP address for a specified number of days.

**Example:**
```
/auth @john 192.168.1.1 30
```

**What happens:**
- Creates a new subscription for the user
- Sets start date to current time
- Sets days remaining to the specified amount
- Logs the auth event to the log channel
- Sends confirmation message

**Response:**
```
✅ User Authorized

👤 User: @john
🆔 ID: 123456789
🌐 IP: 192.168.1.1
📅 Days: 30
⏰ Expires: 2024-12-13 15:30
```

### 2. /deauth @username IP
Deauthorizes a user and kicks them from the group.

**Example:**
```
/deauth @john 192.168.1.1
```

**What happens:**
- Marks subscription as inactive
- Attempts to kick user from GROUP_CHAT_ID
- Sends message to group: "⚠️ Unauthorized and kicked @john (123456789) successfully."
- Logs the deauth event to log channel
- Sends confirmation message

**Response:**
```
⚠️ Unauthorized and kicked @john (123456789) successfully.

🌐 IP: 192.168.1.1
⏰ 2024-11-13 15:30
```

### 3. /pause @username IP
Pauses a subscription - days stop counting.

**Example:**
```
/pause @john 192.168.1.1
```

**How it works:**
- If user has 30 days initially and 12 days have passed, they have 18 days remaining
- Pausing saves: "18 days remaining"
- Updates: IsPaused = true, PausedAt = current time, PausedDaysLeft = 18
- Days do NOT decrease while paused

**Response:**
```
⏸️ Subscription Paused

👤 @john (123456789)
🌐 192.168.1.1
📅 Days left: 18
⏰ 2024-11-13 15:30
```

### 4. /resume @username IP
Resumes a paused subscription from where it left off.

**Example:**
```
/resume @john 192.168.1.1
```

**How it works:**
- Sets IsPaused = false
- Restores DaysRemaining from PausedDaysLeft (18 days)
- Updates LastChecked to current time
- Days continue counting down from 18

**Response:**
```
▶️ Subscription Resumed

👤 @john (123456789)
🌐 192.168.1.1
📅 Days left: 18
⏰ Will expire: 2024-12-01 15:30
```

### 5. /list
Lists all active subscriptions.

**Example:**
```
/list
```

**Response:**
```
📋 Active Subscriptions

1. @john
   🆔 123456789
   🌐 192.168.1.1
   📅 Days: 18/30
   📊 ✅ Active
   ⏰ Expires: 2024-12-01

2. @jane
   🆔 987654321
   🌐 192.168.1.2
   📅 Days: 25/30
   📊 ⏸️ Paused
```

### 6. /status @username IP
Shows detailed status of a specific subscription.

**Example:**
```
/status @john 192.168.1.1
```

**Response:**
```
📊 Subscription Status

👤 User: @john
🆔 ID: 123456789
🌐 IP: 192.168.1.1
📅 Days: 18/30
📊 Status: ✅ Active
⏰ Expires: 2024-12-01 15:30
📅 Started: 2024-11-13 15:30
```

## Day Tracking Logic

### Normal Operation (Active, Not Paused)
1. Every hour, the expiry checker runs
2. For each active, non-paused subscription:
   - Calculate elapsed time since LastChecked
   - Subtract days passed from DaysRemaining
   - Update LastChecked to current time
3. If DaysRemaining reaches 0, subscription expires

### Example Timeline:
```
Day 0:  /auth @john 192.168.1.1 30  → 30 days remaining
Day 12: /pause @john 192.168.1.1     → 18 days remaining (frozen)
Day 20: Still paused                 → 18 days remaining (no change)
Day 25: /resume @john 192.168.1.1    → 18 days remaining (starts counting)
Day 43: Subscription expires         → 0 days remaining (25 + 18 = 43)
```

## Expiry Process

When a subscription expires:

1. **Automatic Detection**: Hourly background task checks all active subscriptions
2. **Deactivation**: Sets Active = false
3. **Group Kick**: Attempts to kick user from GROUP_CHAT_ID
4. **Log Notification**: Sends to LOG_CHAT_ID (TWICE with warnings):

```
⚠️ SUBSCRIPTION EXPIRED ⚠️

👤 @john
🆔 123456789
🌐 192.168.1.1
⏰ Expired: 2024-12-01 15:30
👢 Kicked: true
```

This message is sent twice as requested for emphasis.

## Logging

All events are logged to LOG_CHAT_ID with the following format:

### Auth Log
```
✅ AUTH

👤 @john (123456789)
🌐 192.168.1.1
📅 30 days
⏰ 2024-11-13 15:30
```

### Deauth Log
```
⚠️ DEAUTH

👤 @john (123456789)
🌐 192.168.1.1
👢 Kicked: true
⏰ 2024-11-13 15:30
```

### Pause Log
```
⏸️ PAUSE

👤 @john (123456789)
🌐 192.168.1.1
📅 Days left: 18
⏰ 2024-11-13 15:30
```

### Resume Log
```
▶️ RESUME

👤 @john (123456789)
🌐 192.168.1.1
📅 Days left: 18
⏰ Will expire: 2024-12-01 15:30
```

### Expiry Log (sent twice)
```
⚠️ SUBSCRIPTION EXPIRED ⚠️

👤 @john
🆔 123456789
🌐 192.168.1.1
⏰ Expired: 2024-12-01 15:30
👢 Kicked: true
```

## Configuration

### Environment Variables

Required:
- `TELEGRAM_BOT_TOKEN` - Your bot token from @BotFather

Optional but recommended:
- `ADMIN_ID` - Telegram user ID who can use commands
- `GROUP_CHAT_ID` - Chat ID where users will be kicked on deauth/expiry
- `LOG_CHAT_ID` - Chat ID where all events are logged

### Getting Chat IDs

1. **For Groups**: Add the bot to the group and use /status command, check the logs
2. **For Channels**: Forward a message from the channel to @userinfobot
3. **For User ID**: Message @userinfobot

## Data Persistence

Subscriptions are stored in `subscriptions.json`:

```json
{
  "123456789_192.168.1.1": {
    "user_id": 123456789,
    "username": "john",
    "ip": "192.168.1.1",
    "total_days": 30,
    "days_remaining": 18,
    "start_date": "2024-11-13T15:30:00Z",
    "last_checked": "2024-11-25T15:30:00Z",
    "is_paused": false,
    "active": true
  }
}
```

This file persists across bot restarts.

## Architecture

### Thread Safety
- All storage operations use mutex locks (RWMutex)
- Prevent race conditions when multiple commands run simultaneously
- Safe for concurrent access

### Background Tasks
- Expiry checker runs every 1 hour
- Checks all active, non-paused subscriptions
- Updates days remaining based on elapsed time

### Error Handling
- Graceful handling of missing subscriptions
- Proper error messages for invalid commands
- Logging of all errors and events

## Security

- Only users with ADMIN_ID can use commands (if configured)
- No hardcoded credentials
- Environment-based configuration
- No SQL injection risks (uses JSON storage)
- Passed CodeQL security scan with 0 alerts

## Testing

All core functionality is tested:
- ✅ Storage operations (add, get, update, remove)
- ✅ Days remaining calculation
- ✅ Pause/Resume logic
- ✅ Hash function consistency
- ✅ Markdown escaping

Run tests: `go test -v`

## Professional Design

- Clean, intuitive commands
- Clear status messages
- Emoji-enhanced formatting
- Comprehensive error messages
- Easy navigation
- Professional logging
