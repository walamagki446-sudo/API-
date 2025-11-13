package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// ====================================
// DATA STRUCTURES
// ====================================

// Subscription represents a user's subscription with pause/resume support
type Subscription struct {
	UserID          int64     `json:"user_id"`
	Username        string    `json:"username"`
	IP              string    `json:"ip"`
	TotalDays       int       `json:"total_days"`
	DaysRemaining   int       `json:"days_remaining"`
	StartDate       time.Time `json:"start_date"`
	LastChecked     time.Time `json:"last_checked"`
	IsPaused        bool      `json:"is_paused"`
	PausedAt        time.Time `json:"paused_at,omitempty"`
	PausedDaysLeft  int       `json:"paused_days_left,omitempty"`
	Active          bool      `json:"active"`
}

// Config holds bot configuration
type Config struct {
	TelegramToken string
	AdminIDs      []int64
	GroupChatID   int64
	LogChatID     int64
}

// Storage manages subscription data persistence
type Storage struct {
	Subscriptions map[string]*Subscription `json:"subscriptions"` // key: "userid_ip"
	mu            sync.RWMutex
	filename      string
}

var (
	config  Config
	storage *Storage
)

// ====================================
// MAIN
// ====================================

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("[WARNING] .env file not found, using environment variables")
	}

	config = Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		AdminIDs:      getEnvAsInt64Slice("ADMIN_IDS", []int64{}),
		GroupChatID:   getEnvAsInt64("GROUP_CHAT_ID", 0),
		LogChatID:     getEnvAsInt64("LOG_CHAT_ID", 0),
	}

	if config.TelegramToken == "" {
		log.Fatal("[ERROR] TELEGRAM_BOT_TOKEN not set!")
	}

	// Initialize storage
	storage = NewStorage("subscriptions.json")
	if err := storage.Load(); err != nil {
		log.Printf("[WARNING] Could not load existing subscriptions: %v", err)
	}

	// Create bot
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Fatal("[ERROR] Bot startup error:", err)
	}

	bot.Debug = false
	log.Printf("[INFO] Bot started: @%s", bot.Self.UserName)
	
	// Log admin configuration
	if len(config.AdminIDs) > 0 {
		log.Printf("[INFO] Configured %d admin(s): %v", len(config.AdminIDs), config.AdminIDs)
	} else {
		log.Println("[INFO] No admin restrictions - all users can use commands")
	}
	
	// Count active subscriptions only
	activeCount := 0
	for _, sub := range storage.Subscriptions {
		if sub.Active {
			activeCount++
		}
	}
	log.Printf("[INFO] Loaded %d total subscriptions (%d active, %d inactive)", 
		len(storage.Subscriptions), activeCount, len(storage.Subscriptions)-activeCount)

	// Check and update all active subscriptions on startup to catch any missed days
	log.Println("[INFO] Checking subscriptions for missed updates...")
	checkExpiredSubscriptions(bot)
	log.Println("[INFO] Startup subscription check complete")

	// Start expiry checker in background
	go expiryChecker(bot)

	// Start bot updates loop
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		go handleMessage(bot, update.Message)
	}
}

// ====================================
// STORAGE METHODS
// ====================================

func NewStorage(filename string) *Storage {
	return &Storage{
		Subscriptions: make(map[string]*Subscription),
		filename:      filename,
	}
}

func (s *Storage) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's okay
		}
		return err
	}

	return json.Unmarshal(data, &s.Subscriptions)
}

func (s *Storage) Save() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.Subscriptions, "", "  ")
	s.mu.RUnlock()
	
	if err != nil {
		log.Printf("[ERROR] Failed to marshal subscriptions: %v", err)
		return err
	}

	err = os.WriteFile(s.filename, data, 0644)
	if err != nil {
		log.Printf("[ERROR] Failed to write subscriptions to file: %v", err)
		return err
	}
	
	return nil
}

func (s *Storage) GetKey(userID int64, ip string) string {
	return fmt.Sprintf("%d_%s", userID, ip)
}

func (s *Storage) AddSubscription(sub *Subscription) {
	s.mu.Lock()
	key := s.GetKey(sub.UserID, sub.IP)
	s.Subscriptions[key] = sub
	s.mu.Unlock()
	s.Save()
}

func (s *Storage) GetSubscription(userID int64, ip string) *Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := s.GetKey(userID, ip)
	return s.Subscriptions[key]
}

func (s *Storage) RemoveSubscription(userID int64, ip string) {
	s.mu.Lock()
	key := s.GetKey(userID, ip)
	delete(s.Subscriptions, key)
	s.mu.Unlock()
	s.Save()
}

func (s *Storage) UpdateSubscription(sub *Subscription) {
	s.mu.Lock()
	key := s.GetKey(sub.UserID, sub.IP)
	s.Subscriptions[key] = sub
	s.mu.Unlock()
	s.Save()
}

func (s *Storage) GetAllActiveSubscriptions() []*Subscription {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var active []*Subscription
	for _, sub := range s.Subscriptions {
		if sub.Active {
			active = append(active, sub)
		}
	}
	return active
}

// ====================================
// MESSAGE HANDLER
// ====================================

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Only admins can use commands
	if len(config.AdminIDs) > 0 && !isAdmin(msg.From.ID) {
		return
	}

	switch msg.Command() {
	case "start":
		handleStart(bot, msg)
	case "auth":
		handleAuth(bot, msg)
	case "deauth":
		handleDeauth(bot, msg)
	case "pause":
		handlePause(bot, msg)
	case "resume":
		handleResume(bot, msg)
	case "list":
		handleList(bot, msg)
	case "status":
		handleStatus(bot, msg)
	case "debug":
		handleDebug(bot, msg)
	default:
		if msg.Command() != "" {
			sendMessage(bot, msg.Chat.ID, "Unknown command. Use /start for help.", false)
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := `*Subscription Management Bot*

*Commands:*

/auth @userid IP days \- Authorize user
/deauth @userid IP \- Deauthorize user
/pause @userid IP \- Pause subscription
/resume @userid IP \- Resume subscription
/list \- List all subscriptions
/status @userid IP \- Check subscription status
/debug \- Show debug information

*Examples:*
` + "```" + `
/auth @john 192.168.1.1 30
/deauth @john 192.168.1.1
/pause @john 192.168.1.1
/resume @john 192.168.1.1
` + "```" + `

*Features:*
• Pause/Resume with accurate day tracking
• Auto\-kick on deauth
• Comprehensive logging
• Expiry notifications

*Setup Required:*
Set GROUP\_CHAT\_ID and LOG\_CHAT\_ID in \.env`

	sendMessage(bot, msg.Chat.ID, helpText, true)
}

func handleAuth(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Parse: /auth @userid IP days
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 3 {
		sendMessage(bot, msg.Chat.ID, "Usage: /auth @userid IP days\nExample: /auth @john 192.168.1.1 30", false)
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	ip := args[1]
	days, err := strconv.Atoi(args[2])
	if err != nil || days <= 0 {
		sendMessage(bot, msg.Chat.ID, "Invalid days! Must be a positive number.", false)
		return
	}

	// For this implementation, we'll use a placeholder userID
	// In production, you'd resolve the username to userID via Telegram API
	userID := int64(hashString(username))

	// Check if subscription already exists
	existing := storage.GetSubscription(userID, ip)
	if existing != nil && existing.Active {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("[WARNING] User @%s with IP %s already has an active subscription!", username, ip), false)
		return
	}

	// Create new subscription
	sub := &Subscription{
		UserID:        userID,
		Username:      username,
		IP:            ip,
		TotalDays:     days,
		DaysRemaining: days,
		StartDate:     time.Now(),
		LastChecked:   time.Now(),
		IsPaused:      false,
		Active:        true,
	}

	storage.AddSubscription(sub)
	
	log.Printf("[AUTH] Created subscription: @%s (ID: %d, IP: %s, Days: %d, Active: %v)", 
		username, userID, ip, days, sub.Active)

	// Send confirmation
	confirmMsg := fmt.Sprintf("*User Authorized*\n\nUser: @%s\nID: `%d`\nIP: `%s`\nDays: `%d`\nExpires: `%s`",
		username, userID, ip, days, time.Now().Add(time.Duration(days)*24*time.Hour).Format("2006-01-02 15:04"))
	sendMessage(bot, msg.Chat.ID, confirmMsg, true)

	// Log to log channel
	if config.LogChatID != 0 {
		logMsg := fmt.Sprintf("*AUTH*\n\n@%s \\(`%d`\\)\n`%s`\n%d days\n%s",
			escapeMarkdownV2(username), userID, escapeMarkdownV2(ip), days,
			escapeMarkdownV2(time.Now().Format("2006-01-02 15:04")))
		sendMessage(bot, config.LogChatID, logMsg, true)
	}

	log.Printf("[AUTH] @%s (ID: %d, IP: %s, Days: %d)", username, userID, ip, days)
}

func handleDeauth(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Parse: /deauth @userid IP
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 2 {
		sendMessage(bot, msg.Chat.ID, "Usage: /deauth @userid IP\nExample: /deauth @john 192.168.1.1", false)
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	ip := args[1]
	userID := int64(hashString(username))

	// Check if subscription exists
	sub := storage.GetSubscription(userID, ip)
	if sub == nil || !sub.Active {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("No active subscription found for @%s with IP %s", username, ip), false)
		return
	}

	// Deactivate subscription
	sub.Active = false
	storage.UpdateSubscription(sub)

	// Try to kick user from group if GROUP_CHAT_ID is set
	kickSuccess := false
	if config.GroupChatID != 0 {
		kickConfig := tgbotapi.KickChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{
				ChatID: config.GroupChatID,
				UserID: userID,
			},
		}
		_, err := bot.Request(kickConfig)
		if err == nil {
			kickSuccess = true
			log.Printf("[KICK] Kicked user %d from group", userID)
		} else {
			log.Printf("[WARNING] Failed to kick user %d: %v", userID, err)
		}
	}

	// Send confirmation with kick status
	var confirmMsg string
	if kickSuccess {
		confirmMsg = fmt.Sprintf("*Unauthorized and kicked @%s \\(%d\\) successfully\\.*\n\nIP: `%s`\n%s",
			escapeMarkdownV2(username), userID, escapeMarkdownV2(ip),
			escapeMarkdownV2(time.Now().Format("2006-01-02 15:04")))
	} else {
		confirmMsg = fmt.Sprintf("*User Deauthorized*\n\n@%s \\(`%d`\\)\n`%s`\n%s",
			escapeMarkdownV2(username), userID, escapeMarkdownV2(ip),
			escapeMarkdownV2(time.Now().Format("2006-01-02 15:04")))
	}
	sendMessage(bot, msg.Chat.ID, confirmMsg, true)

	// Send kick message to group if kicked
	if kickSuccess && config.GroupChatID != 0 {
		groupMsg := fmt.Sprintf("Unauthorized and kicked @%s \\(%d\\) successfully\\.",
			escapeMarkdownV2(username), userID)
		sendMessage(bot, config.GroupChatID, groupMsg, true)
	}

	// Log to log channel
	if config.LogChatID != 0 {
		logMsg := fmt.Sprintf("*DEAUTH*\n\n@%s \\(`%d`\\)\n`%s`\nKicked: %v\n%s",
			escapeMarkdownV2(username), userID, escapeMarkdownV2(ip), kickSuccess,
			escapeMarkdownV2(time.Now().Format("2006-01-02 15:04")))
		sendMessage(bot, config.LogChatID, logMsg, true)
	}

	log.Printf("[DEAUTH] @%s (ID: %d, IP: %s, Kicked: %v)", username, userID, ip, kickSuccess)
}

func handlePause(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Parse: /pause @userid IP
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 2 {
		sendMessage(bot, msg.Chat.ID, "Usage: /pause @userid IP\nExample: /pause @john 192.168.1.1", false)
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	ip := args[1]
	userID := int64(hashString(username))

	// Get subscription
	sub := storage.GetSubscription(userID, ip)
	if sub == nil || !sub.Active {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("No active subscription found for @%s with IP %s", username, ip), false)
		return
	}

	if sub.IsPaused {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("[WARNING] Subscription for @%s is already paused!", username), false)
		return
	}

	// Update days remaining before pausing
	updateDaysRemaining(sub)

	// Pause subscription
	sub.IsPaused = true
	sub.PausedAt = time.Now()
	sub.PausedDaysLeft = sub.DaysRemaining
	storage.UpdateSubscription(sub)

	// Send confirmation
	confirmMsg := fmt.Sprintf("*Subscription Paused*\n\n@%s \\(`%d`\\)\n`%s`\nDays left: `%d`\n%s",
		escapeMarkdownV2(username), userID, escapeMarkdownV2(ip), sub.DaysRemaining,
		escapeMarkdownV2(time.Now().Format("2006-01-02 15:04")))
	sendMessage(bot, msg.Chat.ID, confirmMsg, true)

	// Log to log channel
	if config.LogChatID != 0 {
		logMsg := fmt.Sprintf("*PAUSE*\n\n@%s \\(`%d`\\)\n`%s`\nDays left: `%d`\n%s",
			escapeMarkdownV2(username), userID, escapeMarkdownV2(ip), sub.DaysRemaining,
			escapeMarkdownV2(time.Now().Format("2006-01-02 15:04")))
		sendMessage(bot, config.LogChatID, logMsg, true)
	}

	log.Printf("[PAUSE] @%s (ID: %d, IP: %s, Days left: %d)", username, userID, ip, sub.DaysRemaining)
}

func handleResume(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Parse: /resume @userid IP
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 2 {
		sendMessage(bot, msg.Chat.ID, "Usage: /resume @userid IP\nExample: /resume @john 192.168.1.1", false)
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	ip := args[1]
	userID := int64(hashString(username))

	// Get subscription
	sub := storage.GetSubscription(userID, ip)
	if sub == nil || !sub.Active {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("No active subscription found for @%s with IP %s", username, ip), false)
		return
	}

	if !sub.IsPaused {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("[WARNING] Subscription for @%s is not paused!", username), false)
		return
	}

	// Resume subscription
	sub.IsPaused = false
	sub.LastChecked = time.Now()
	sub.DaysRemaining = sub.PausedDaysLeft
	storage.UpdateSubscription(sub)

	// Send confirmation
	expiryDate := time.Now().Add(time.Duration(sub.DaysRemaining) * 24 * time.Hour)
	confirmMsg := fmt.Sprintf("*Subscription Resumed*\n\n@%s \\(`%d`\\)\n`%s`\nDays left: `%d`\nWill expire: `%s`",
		escapeMarkdownV2(username), userID, escapeMarkdownV2(ip), sub.DaysRemaining,
		escapeMarkdownV2(expiryDate.Format("2006-01-02 15:04")))
	sendMessage(bot, msg.Chat.ID, confirmMsg, true)

	// Log to log channel
	if config.LogChatID != 0 {
		logMsg := fmt.Sprintf("*RESUME*\n\n@%s \\(`%d`\\)\n`%s`\nDays left: `%d`\nWill expire: `%s`",
			escapeMarkdownV2(username), userID, escapeMarkdownV2(ip), sub.DaysRemaining,
			escapeMarkdownV2(expiryDate.Format("2006-01-02 15:04")))
		sendMessage(bot, config.LogChatID, logMsg, true)
	}

	log.Printf("[RESUME] @%s (ID: %d, IP: %s, Days left: %d)", username, userID, ip, sub.DaysRemaining)
}

func handleList(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	subs := storage.GetAllActiveSubscriptions()
	
	log.Printf("[INFO] /list command called, found %d active subscriptions", len(subs))

	if len(subs) == 0 {
		sendMessage(bot, msg.Chat.ID, "No active subscriptions found.", false)
		return
	}

	var sb strings.Builder
	sb.WriteString("*Active Subscriptions*\n\n")

	for i, sub := range subs {
		// Update days remaining
		updateDaysRemaining(sub)

		status := "Active"
		if sub.IsPaused {
			status = "Paused"
		}

		sb.WriteString(fmt.Sprintf("%d\\. @%s\n", i+1, escapeMarkdownV2(sub.Username)))
		sb.WriteString(fmt.Sprintf("   ID: `%d`\n", sub.UserID))
		sb.WriteString(fmt.Sprintf("   IP: `%s`\n", escapeMarkdownV2(sub.IP)))
		sb.WriteString(fmt.Sprintf("   Days: `%d/%d`\n", sub.DaysRemaining, sub.TotalDays))
		sb.WriteString(fmt.Sprintf("   Status: %s\n", status))

		if !sub.IsPaused {
			expiryDate := sub.LastChecked.Add(time.Duration(sub.DaysRemaining) * 24 * time.Hour)
			sb.WriteString(fmt.Sprintf("   Expires: `%s`\n", escapeMarkdownV2(expiryDate.Format("2006-01-02"))))
		}
		sb.WriteString("\n")
	}

	sendMessage(bot, msg.Chat.ID, sb.String(), true)
	log.Printf("[INFO] /list response sent with %d subscriptions", len(subs))
}

func handleStatus(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Parse: /status @userid IP
	args := strings.Fields(msg.CommandArguments())
	if len(args) != 2 {
		sendMessage(bot, msg.Chat.ID, "Usage: /status @userid IP\nExample: /status @john 192.168.1.1", false)
		return
	}

	username := strings.TrimPrefix(args[0], "@")
	ip := args[1]
	userID := int64(hashString(username))

	// Get subscription
	sub := storage.GetSubscription(userID, ip)
	if sub == nil {
		sendMessage(bot, msg.Chat.ID, fmt.Sprintf("No subscription found for @%s with IP %s", username, ip), false)
		return
	}

	// Update days remaining
	if sub.Active && !sub.IsPaused {
		updateDaysRemaining(sub)
	}

	var statusMsg strings.Builder
	statusMsg.WriteString("*Subscription Status*\n\n")
	statusMsg.WriteString(fmt.Sprintf("User: @%s\n", escapeMarkdownV2(sub.Username)))
	statusMsg.WriteString(fmt.Sprintf("ID: `%d`\n", sub.UserID))
	statusMsg.WriteString(fmt.Sprintf("IP: `%s`\n", escapeMarkdownV2(sub.IP)))
	statusMsg.WriteString(fmt.Sprintf("Days: `%d/%d`\n", sub.DaysRemaining, sub.TotalDays))

	if sub.Active {
		if sub.IsPaused {
			statusMsg.WriteString("Status: Paused\n")
			statusMsg.WriteString(fmt.Sprintf("Paused on: `%s`\n", escapeMarkdownV2(sub.PausedAt.Format("2006-01-02 15:04"))))
		} else {
			statusMsg.WriteString("Status: Active\n")
			expiryDate := sub.LastChecked.Add(time.Duration(sub.DaysRemaining) * 24 * time.Hour)
			statusMsg.WriteString(fmt.Sprintf("Expires: `%s`\n", escapeMarkdownV2(expiryDate.Format("2006-01-02 15:04"))))
		}
	} else {
		statusMsg.WriteString("Status: Deactivated\n")
	}

	statusMsg.WriteString(fmt.Sprintf("Started: `%s`\n", escapeMarkdownV2(sub.StartDate.Format("2006-01-02 15:04"))))

	sendMessage(bot, msg.Chat.ID, statusMsg.String(), true)
}

func handleDebug(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	storage.mu.RLock()
	defer storage.mu.RUnlock()
	
	var debugMsg strings.Builder
	debugMsg.WriteString("*Debug Information*\n\n")
	debugMsg.WriteString(fmt.Sprintf("Total subscriptions: `%d`\n\n", len(storage.Subscriptions)))
	
	if len(storage.Subscriptions) == 0 {
		debugMsg.WriteString("No subscriptions in storage\\.\n")
	} else {
		debugMsg.WriteString("*All Subscriptions:*\n")
		for key, sub := range storage.Subscriptions {
			debugMsg.WriteString(fmt.Sprintf("\nKey: `%s`\n", escapeMarkdownV2(key)))
			debugMsg.WriteString(fmt.Sprintf("  User: @%s \\(ID: %d\\)\n", escapeMarkdownV2(sub.Username), sub.UserID))
			debugMsg.WriteString(fmt.Sprintf("  IP: `%s`\n", escapeMarkdownV2(sub.IP)))
			debugMsg.WriteString(fmt.Sprintf("  Active: `%v`\n", sub.Active))
			debugMsg.WriteString(fmt.Sprintf("  Paused: `%v`\n", sub.IsPaused))
			debugMsg.WriteString(fmt.Sprintf("  Days: `%d/%d`\n", sub.DaysRemaining, sub.TotalDays))
		}
	}
	
	sendMessage(bot, msg.Chat.ID, debugMsg.String(), true)
	log.Printf("[DEBUG] Debug info sent to admin %d", msg.From.ID)
}

// ====================================
// EXPIRY CHECKER (Background Task)
// ====================================

func expiryChecker(bot *tgbotapi.BotAPI) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// Also create a backup ticker every 6 hours
	backupTicker := time.NewTicker(6 * time.Hour)
	defer backupTicker.Stop()

	log.Println("[INFO] Expiry checker started (runs every 1 hour)")
	log.Println("[INFO] Backup saver started (runs every 6 hours)")

	for {
		select {
		case <-ticker.C:
			checkExpiredSubscriptions(bot)
		case <-backupTicker.C:
			// Force save all subscriptions as backup
			if err := storage.Save(); err != nil {
				log.Printf("[ERROR] Backup save failed: %v", err)
			} else {
				log.Println("[INFO] Backup save completed successfully")
			}
		}
	}
}

func checkExpiredSubscriptions(bot *tgbotapi.BotAPI) {
	subs := storage.GetAllActiveSubscriptions()
	now := time.Now()
	
	if len(subs) == 0 {
		log.Println("[INFO] No active subscriptions to check")
		return
	}
	
	log.Printf("[INFO] Checking %d active subscriptions for expiry", len(subs))

	for _, sub := range subs {
		// Skip paused subscriptions
		if sub.IsPaused {
			log.Printf("[INFO] Skipping paused subscription: @%s", sub.Username)
			continue
		}

		// Update days remaining
		updateDaysRemaining(sub)

		// Check if expired
		if sub.DaysRemaining <= 0 {
			log.Printf("[EXPIRY] Subscription expired: @%s (ID: %d, IP: %s)", sub.Username, sub.UserID, sub.IP)

			// Deactivate subscription
			sub.Active = false
			storage.UpdateSubscription(sub)

			// Try to kick from group
			kickSuccess := false
			if config.GroupChatID != 0 {
				kickConfig := tgbotapi.KickChatMemberConfig{
					ChatMemberConfig: tgbotapi.ChatMemberConfig{
						ChatID: config.GroupChatID,
						UserID: sub.UserID,
					},
				}
				_, err := bot.Request(kickConfig)
				if err == nil {
					kickSuccess = true
					log.Printf("[KICK] Kicked expired user %d from group", sub.UserID)
				} else {
					log.Printf("[ERROR] Failed to kick user %d: %v", sub.UserID, err)
				}
			}

			// Prepare expiry message
			expiryMsg := fmt.Sprintf("*SUBSCRIPTION EXPIRED*\n\n@%s\n`%d`\n`%s`\nExpired: `%s`\nKicked: %v",
				escapeMarkdownV2(sub.Username), sub.UserID, escapeMarkdownV2(sub.IP),
				escapeMarkdownV2(now.Format("2006-01-02 15:04")), kickSuccess)

			// Send expiry notification to log channel (TWICE with warning)
			if config.LogChatID != 0 {
				// Send first notification
				sendMessage(bot, config.LogChatID, expiryMsg, true)
				time.Sleep(2 * time.Second)
				// Send second notification (as requested)
				sendMessage(bot, config.LogChatID, expiryMsg, true)
				log.Printf("[INFO] Expiry notification sent to log channel for @%s", sub.Username)
			}
			
			// Also notify all admins directly about the expiry
			if len(config.AdminIDs) > 0 {
				for _, adminID := range config.AdminIDs {
					sendMessage(bot, adminID, expiryMsg, true)
					log.Printf("[INFO] Expiry notification sent to admin %d", adminID)
				}
			}
		}
	}
	
	log.Println("[INFO] Expiry check completed")
}

func updateDaysRemaining(sub *Subscription) {
	if sub.IsPaused || !sub.Active {
		return
	}

	now := time.Now()
	elapsed := now.Sub(sub.LastChecked)
	
	// Calculate days passed with precision - use 24 hours exactly
	daysPassed := int(elapsed.Hours() / 24)
	
	// Safety check: if more than 30 days have passed since last check, 
	// something is wrong - log a warning but still process
	if daysPassed > 30 {
		log.Printf("[WARNING] Large time gap detected for @%s: %d days since last check", 
			sub.Username, daysPassed)
	}

	if daysPassed > 0 {
		sub.DaysRemaining -= daysPassed
		if sub.DaysRemaining < 0 {
			sub.DaysRemaining = 0
		}
		sub.LastChecked = now
		
		// Always save to disk after updating
		storage.UpdateSubscription(sub)
		
		log.Printf("[UPDATE] @%s: %d days passed, %d days remaining", 
			sub.Username, daysPassed, sub.DaysRemaining)
	}
}

// ====================================
// HELPER FUNCTIONS
// ====================================

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string, markdown bool) {
	msg := tgbotapi.NewMessage(chatID, text)
	if markdown {
		msg.ParseMode = "MarkdownV2"
	}
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func escapeMarkdownV2(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	var val int64
	fmt.Sscanf(valStr, "%d", &val)
	return val
}

// getEnvAsInt64Slice parses comma-separated admin IDs from environment variable
func getEnvAsInt64Slice(key string, defaultVal []int64) []int64 {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal
	}
	
	// Split by comma and parse each ID
	parts := strings.Split(valStr, ",")
	var ids []int64
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id int64
		if _, err := fmt.Sscanf(part, "%d", &id); err == nil {
			ids = append(ids, id)
		}
	}
	
	if len(ids) == 0 {
		return defaultVal
	}
	return ids
}

// isAdmin checks if a user ID is in the admin list
func isAdmin(userID int64) bool {
	for _, adminID := range config.AdminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// Simple hash function for username to userID conversion
// In production, you'd use Telegram's actual user resolution
func hashString(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}
