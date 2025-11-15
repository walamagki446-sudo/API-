package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ====================================
// STRUCTS
// ====================================

type SiteResult struct {
	URL           string
	Title         string
	Price         string
	Region        string
	InstantDeliv  bool
	Source        string
	Timestamp     time.Time
}

type SearchStats struct {
	TotalSites    int32
	WithPrice     int32
	InstantDeliv  int32
	CurrentRegion string
	StartTime     time.Time
	mu            sync.Mutex
}

// ====================================
// GLOBALS
// ====================================

var (
	stats       = &SearchStats{StartTime: time.Now()}
	siteResults = make([]SiteResult, 0)
	resultsMu   sync.Mutex
	seenURLs    = make(map[string]bool)
	seenMu      sync.Mutex
)

// Search keywords for Red Dead Redemption 2
var keywords = []string{
	"Red Dead Redemption 2",
	"RDR2",
	"Red Dead Redemption II",
	"Red Dead 2",
	"RDR 2",
	"Red Dead Online",
}

// Additional search terms
var searchTerms = []string{
	"instant delivery",
	"digital download",
	"buy now",
	"cd key",
	"steam key",
	"rockstar key",
	"game key",
	"pc game",
	"online game store",
	"instant access",
	"buy online",
	"digital key",
	"activation code",
	"game code",
	"cheap",
	"discount",
	"sale",
	"best price",
}

// Regions to search from
var regions = []struct {
	Name    string
	Domain  string
	LangCC  string
}{
	{"India", ".co.in", "en-IN"},
	{"Germany", ".de", "de-DE"},
	{"Netherlands", ".nl", "nl-NL"},
	{"United Kingdom", ".co.uk", "en-GB"},
	{"United States", ".com", "en-US"},
	{"France", ".fr", "fr-FR"},
	{"Spain", ".es", "es-ES"},
	{"Italy", ".it", "it-IT"},
	{"Brazil", ".com.br", "pt-BR"},
	{"Australia", ".com.au", "en-AU"},
	{"Canada", ".ca", "en-CA"},
	{"Japan", ".co.jp", "ja-JP"},
	{"South Korea", ".co.kr", "ko-KR"},
	{"Singapore", ".com.sg", "en-SG"},
	{"Malaysia", ".com.my", "en-MY"},
	{"Thailand", ".co.th", "th-TH"},
	{"Indonesia", ".co.id", "id-ID"},
	{"Philippines", ".com.ph", "en-PH"},
	{"Poland", ".pl", "pl-PL"},
	{"Russia", ".ru", "ru-RU"},
	{"Turkey", ".com.tr", "tr-TR"},
	{"Mexico", ".com.mx", "es-MX"},
	{"Argentina", ".com.ar", "es-AR"},
	{"Chile", ".cl", "es-CL"},
	{"Sweden", ".se", "sv-SE"},
	{"Norway", ".no", "no-NO"},
	{"Denmark", ".dk", "da-DK"},
	{"Finland", ".fi", "fi-FI"},
	{"Belgium", ".be", "nl-BE"},
	{"Switzerland", ".ch", "de-CH"},
}

// Known game stores and marketplaces
var knownGameStores = []string{
	"g2a.com", "kinguin.net", "cdkeys.com", "instant-gaming.com",
	"greenmangaming.com", "fanatical.com", "humblebundle.com",
	"gog.com", "steam", "epicgames.com", "gamivo.com",
	"eneba.com", "gamersgate.com", "gamesplanet", "dlgamer.com",
	"voidu.com", "2game.com", "mmoga.com", "allkeyshop.com",
	"gamespot.com", "ign.com", "gamestop", "amazon",
	"walmart", "target", "bestbuy", "newegg",
	"rockstargames.com", "microsoft.com", "playstation.com",
}

// User agents for rotation
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
}

// ====================================
// MAIN
// ====================================

func main() {
	rand.Seed(time.Now().UnixNano())
	
	printBanner()
	
	fmt.Println("\n🚀 Starting Red Dead Redemption 2 Site Finder")
	fmt.Println("📊 Target: 5,000 sites with instant delivery")
	fmt.Println("🌍 Searching across 30+ regions worldwide")
	fmt.Println("💰 Extracting prices from discovered sites")
	fmt.Println("⚡ Using deep search with multiple keywords\n")
	
	stats.StartTime = time.Now()
	
	// Start progress display goroutine
	stopProgress := make(chan bool)
	go displayProgress(stopProgress)
	
	// Multi-threaded search
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // Limit concurrent searches
	
	// Search across all regions
	for _, region := range regions {
		// For each keyword combination
		for _, keyword := range keywords {
			for _, term := range searchTerms {
				wg.Add(1)
				semaphore <- struct{}{}
				
				go func(reg struct{ Name, Domain, LangCC string }, kw, srch string) {
					defer wg.Done()
					defer func() { <-semaphore }()
					
					stats.mu.Lock()
					stats.CurrentRegion = reg.Name
					stats.mu.Unlock()
					
					// Search using multiple methods
					searchGoogle(reg, kw, srch)
					searchBing(reg, kw, srch)
					searchDuckDuckGo(reg, kw, srch)
					
					// Add small delay to avoid rate limiting
					time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)
				}(region, keyword, term)
				
				// Check if we've reached target
				if atomic.LoadInt32(&stats.TotalSites) >= 5000 {
					break
				}
			}
			if atomic.LoadInt32(&stats.TotalSites) >= 5000 {
				break
			}
		}
		if atomic.LoadInt32(&stats.TotalSites) >= 5000 {
			break
		}
	}
	
	// Also search known game stores directly
	for _, store := range knownGameStores {
		if atomic.LoadInt32(&stats.TotalSites) >= 5000 {
			break
		}
		
		wg.Add(1)
		semaphore <- struct{}{}
		
		go func(storeDomain string) {
			defer wg.Done()
			defer func() { <-semaphore }()
			
			for _, keyword := range keywords {
				searchDirectStore(storeDomain, keyword)
				time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)
			}
		}(store)
	}
	
	wg.Wait()
	stopProgress <- true
	
	// Display final results
	displayFinalResults()
	
	// Save results to file
	saveResults()
	
	fmt.Println("\n✅ Search completed!")
}

// ====================================
// SEARCH FUNCTIONS
// ====================================

func searchGoogle(region struct{ Name, Domain, LangCC string }, keyword, term string) {
	query := fmt.Sprintf("%s %s %s", keyword, term, region.Name)
	searchURL := fmt.Sprintf("https://www.google%s/search?q=%s&num=50&hl=%s",
		region.Domain, url.QueryEscape(query), region.LangCC)
	
	results := fetchSearchResults(searchURL, "google")
	processResults(results, region.Name, "Google")
}

func searchBing(region struct{ Name, Domain, LangCC string }, keyword, term string) {
	query := fmt.Sprintf("%s %s %s", keyword, term, region.Name)
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=50&setlang=%s",
		url.QueryEscape(query), region.LangCC)
	
	results := fetchSearchResults(searchURL, "bing")
	processResults(results, region.Name, "Bing")
}

func searchDuckDuckGo(region struct{ Name, Domain, LangCC string }, keyword, term string) {
	query := fmt.Sprintf("%s %s %s", keyword, term, region.Name)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s&kl=%s",
		url.QueryEscape(query), strings.ToLower(region.LangCC))
	
	results := fetchSearchResults(searchURL, "duckduckgo")
	processResults(results, region.Name, "DuckDuckGo")
}

func searchDirectStore(storeDomain, keyword string) {
	// Try to construct search URL for known stores
	searchURL := fmt.Sprintf("https://%s/search?q=%s", storeDomain, url.QueryEscape(keyword))
	
	results := fetchSearchResults(searchURL, "direct")
	processResults(results, "Direct", storeDomain)
}

// ====================================
// HTTP & PARSING
// ====================================

func fetchSearchResults(searchURL, engine string) []string {
	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil
	}
	
	// Rotate user agent
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil
	}
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}
	
	var links []string
	
	switch engine {
	case "google":
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists && strings.HasPrefix(href, "/url?q=") {
				// Extract actual URL from Google redirect
				parts := strings.Split(href, "?q=")
				if len(parts) > 1 {
					actualURL := strings.Split(parts[1], "&")[0]
					decodedURL, _ := url.QueryUnescape(actualURL)
					if isValidGameURL(decodedURL) {
						links = append(links, decodedURL)
					}
				}
			}
		})
		
	case "bing":
		doc.Find("li.b_algo h2 a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists && isValidGameURL(href) {
				links = append(links, href)
			}
		})
		
	case "duckduckgo":
		doc.Find("a.result__a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists && isValidGameURL(href) {
				links = append(links, href)
			}
		})
		
	case "direct":
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists && isValidGameURL(href) {
				links = append(links, href)
			}
		})
	}
	
	return links
}

func processResults(urls []string, region, source string) {
	for _, siteURL := range urls {
		// Check if we've already seen this URL
		seenMu.Lock()
		if seenURLs[siteURL] {
			seenMu.Unlock()
			continue
		}
		seenURLs[siteURL] = true
		seenMu.Unlock()
		
		// Fetch site details
		result := analyzeSite(siteURL, region, source)
		if result != nil {
			resultsMu.Lock()
			siteResults = append(siteResults, *result)
			resultsMu.Unlock()
			
			atomic.AddInt32(&stats.TotalSites, 1)
			if result.Price != "" {
				atomic.AddInt32(&stats.WithPrice, 1)
			}
			if result.InstantDeliv {
				atomic.AddInt32(&stats.InstantDeliv, 1)
			}
			
			// Display result immediately
			displayResult(result)
		}
		
		// Check if we've reached target
		if atomic.LoadInt32(&stats.TotalSites) >= 5000 {
			break
		}
	}
}

func analyzeSite(siteURL, region, source string) *SiteResult {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	
	req, err := http.NewRequest("GET", siteURL, nil)
	if err != nil {
		return nil
	}
	
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil
	}
	
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil
	}
	
	result := &SiteResult{
		URL:       siteURL,
		Region:    region,
		Source:    source,
		Timestamp: time.Now(),
	}
	
	// Extract title
	result.Title = doc.Find("title").Text()
	if result.Title == "" {
		result.Title = doc.Find("h1").First().Text()
	}
	result.Title = strings.TrimSpace(result.Title)
	
	// Extract price
	result.Price = extractPrice(doc)
	
	// Check for instant delivery indicators
	pageText := strings.ToLower(doc.Text())
	result.InstantDeliv = checkInstantDelivery(pageText)
	
	return result
}

func extractPrice(doc *goquery.Document) string {
	// Common price selectors
	priceSelectors := []string{
		".price", "#price", "[itemprop='price']", ".product-price",
		".sale-price", ".regular-price", ".cost", ".amount",
		"[class*='price']", "[id*='price']", "span.money",
	}
	
	for _, selector := range priceSelectors {
		if price := doc.Find(selector).First().Text(); price != "" {
			price = strings.TrimSpace(price)
			// Check if it looks like a price
			if isPriceFormat(price) {
				return price
			}
		}
	}
	
	// Try regex on page text
	doc.Find("body").Each(func(i int, s *goquery.Selection) {
		text := s.Text()
		if price := extractPriceWithRegex(text); price != "" {
			return
		}
	})
	
	return ""
}

func extractPriceWithRegex(text string) string {
	// Price patterns
	patterns := []string{
		`[$€£¥₹]\s*\d+[.,]?\d*`,                    // $19.99, €29.99, £15.99, ¥1999, ₹899
		`\d+[.,]\d+\s*[$€£¥₹]`,                     // 19.99$, 29,99€
		`(?i)price[:|\s]+[$€£¥₹]?\s*\d+[.,]?\d*`,  // Price: $19.99
		`(?i)[$€£¥₹]?\s*\d+[.,]?\d*\s*USD|EUR|GBP|JPY|INR`, // $19.99 USD
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindString(text); match != "" {
			return strings.TrimSpace(match)
		}
	}
	
	return ""
}

func isPriceFormat(s string) bool {
	// Check if string contains price-like patterns
	pricePattern := regexp.MustCompile(`[$€£¥₹]\s*\d+|(\d+[.,]\d+)`)
	return pricePattern.MatchString(s)
}

func checkInstantDelivery(text string) bool {
	instantKeywords := []string{
		"instant delivery", "instant access", "immediate delivery",
		"digital delivery", "instant download", "digital download",
		"auto delivery", "automatic delivery", "delivered instantly",
		"instant key", "immediate access", "available immediately",
	}
	
	for _, keyword := range instantKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	
	return false
}

func isValidGameURL(urlStr string) bool {
	// Filter out invalid URLs
	if urlStr == "" || strings.HasPrefix(urlStr, "#") {
		return false
	}
	
	// Must be http/https
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return false
	}
	
	// Filter out common non-game sites
	excludeDomains := []string{
		"google.", "facebook.", "twitter.", "instagram.", "youtube.",
		"reddit.", "wikipedia.", "linkedin.", "pinterest.",
	}
	
	for _, domain := range excludeDomains {
		if strings.Contains(urlStr, domain) {
			return false
		}
	}
	
	return true
}

// ====================================
// DISPLAY & OUTPUT
// ====================================

func printBanner() {
	banner := `
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
`
	fmt.Println(banner)
}

func displayProgress(stop chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			elapsed := time.Since(stats.StartTime).Seconds()
			rate := float64(atomic.LoadInt32(&stats.TotalSites)) / elapsed
			
			stats.mu.Lock()
			currentRegion := stats.CurrentRegion
			stats.mu.Unlock()
			
			fmt.Printf("\r\033[K⏱️  Time: %.0fs | 🌍 Region: %-15s | 📊 Found: %d sites | 💰 With Price: %d | ⚡ Instant: %d | Rate: %.1f/s",
				elapsed,
				currentRegion,
				atomic.LoadInt32(&stats.TotalSites),
				atomic.LoadInt32(&stats.WithPrice),
				atomic.LoadInt32(&stats.InstantDeliv),
				rate,
			)
		}
	}
}

func displayResult(result *SiteResult) {
	instantIcon := " "
	if result.InstantDeliv {
		instantIcon = "⚡"
	}
	
	priceDisplay := "N/A"
	if result.Price != "" {
		priceDisplay = result.Price
	}
	
	fmt.Printf("\n%s [%4d] 🌍 %-12s | 💰 %-15s | %s",
		instantIcon,
		atomic.LoadInt32(&stats.TotalSites),
		result.Region,
		priceDisplay,
		result.URL,
	)
}

func displayFinalResults() {
	elapsed := time.Since(stats.StartTime).Seconds()
	
	fmt.Println("\n\n╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                    FINAL RESULTS                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Printf("\n📊 Total Sites Found:        %d\n", atomic.LoadInt32(&stats.TotalSites))
	fmt.Printf("💰 Sites with Prices:        %d (%.1f%%)\n",
		atomic.LoadInt32(&stats.WithPrice),
		float64(atomic.LoadInt32(&stats.WithPrice))/float64(atomic.LoadInt32(&stats.TotalSites))*100)
	fmt.Printf("⚡ Instant Delivery Sites:   %d (%.1f%%)\n",
		atomic.LoadInt32(&stats.InstantDeliv),
		float64(atomic.LoadInt32(&stats.InstantDeliv))/float64(atomic.LoadInt32(&stats.TotalSites))*100)
	fmt.Printf("⏱️  Total Time:               %.2f seconds\n", elapsed)
	fmt.Printf("🚀 Average Rate:             %.2f sites/second\n",
		float64(atomic.LoadInt32(&stats.TotalSites))/elapsed)
	
	// Show top instant delivery sites with prices
	fmt.Println("\n🎯 TOP INSTANT DELIVERY SITES WITH PRICES:")
	fmt.Println("═══════════════════════════════════════════════════════════")
	
	count := 0
	resultsMu.Lock()
	for _, result := range siteResults {
		if result.InstantDeliv && result.Price != "" {
			fmt.Printf("\n%d. 💰 %s\n", count+1, result.Price)
			fmt.Printf("   🌐 %s\n", result.URL)
			fmt.Printf("   🌍 Region: %s | Source: %s\n", result.Region, result.Source)
			count++
			if count >= 50 {
				break
			}
		}
	}
	resultsMu.Unlock()
	
	if count == 0 {
		fmt.Println("No instant delivery sites with prices found in current batch.")
	}
}

func saveResults() {
	filename := fmt.Sprintf("rdr2_sites_%s.json", time.Now().Format("20060102_150405"))
	
	resultsMu.Lock()
	defer resultsMu.Unlock()
	
	data, err := json.MarshalIndent(struct {
		SearchStats struct {
			TotalSites   int32     `json:"total_sites"`
			WithPrice    int32     `json:"with_price"`
			InstantDeliv int32     `json:"instant_delivery"`
			Duration     float64   `json:"duration_seconds"`
			Timestamp    time.Time `json:"timestamp"`
		} `json:"search_stats"`
		Sites []SiteResult `json:"sites"`
	}{
		SearchStats: struct {
			TotalSites   int32     `json:"total_sites"`
			WithPrice    int32     `json:"with_price"`
			InstantDeliv int32     `json:"instant_delivery"`
			Duration     float64   `json:"duration_seconds"`
			Timestamp    time.Time `json:"timestamp"`
		}{
			TotalSites:   atomic.LoadInt32(&stats.TotalSites),
			WithPrice:    atomic.LoadInt32(&stats.WithPrice),
			InstantDeliv: atomic.LoadInt32(&stats.InstantDeliv),
			Duration:     time.Since(stats.StartTime).Seconds(),
			Timestamp:    time.Now(),
		},
		Sites: siteResults,
	}, "", "  ")
	
	if err != nil {
		fmt.Printf("\n❌ Error saving results: %v\n", err)
		return
	}
	
	if err := os.WriteFile(filename, data, 0644); err != nil {
		fmt.Printf("\n❌ Error writing file: %v\n", err)
		return
	}
	
	fmt.Printf("\n💾 Results saved to: %s\n", filename)
	
	// Also create a simple text file with site + price
	textFilename := fmt.Sprintf("rdr2_sites_%s.txt", time.Now().Format("20060102_150405"))
	var textContent strings.Builder
	textContent.WriteString("═══════════════════════════════════════════════════════════\n")
	textContent.WriteString("   RED DEAD REDEMPTION 2 - SITE FINDER RESULTS\n")
	textContent.WriteString(fmt.Sprintf("   Generated: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	textContent.WriteString("═══════════════════════════════════════════════════════════\n\n")
	textContent.WriteString(fmt.Sprintf("Total Sites: %d\n", atomic.LoadInt32(&stats.TotalSites)))
	textContent.WriteString(fmt.Sprintf("With Prices: %d\n", atomic.LoadInt32(&stats.WithPrice)))
	textContent.WriteString(fmt.Sprintf("Instant Delivery: %d\n\n", atomic.LoadInt32(&stats.InstantDeliv)))
	textContent.WriteString("═══════════════════════════════════════════════════════════\n\n")
	
	for i, result := range siteResults {
		instantTag := ""
		if result.InstantDeliv {
			instantTag = " [⚡ INSTANT DELIVERY]"
		}
		
		priceTag := "N/A"
		if result.Price != "" {
			priceTag = result.Price
		}
		
		textContent.WriteString(fmt.Sprintf("%d. PRICE: %s%s\n", i+1, priceTag, instantTag))
		textContent.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		textContent.WriteString(fmt.Sprintf("   Region: %s | Source: %s\n\n", result.Region, result.Source))
	}
	
	os.WriteFile(textFilename, []byte(textContent.String()), 0644)
	fmt.Printf("💾 Text results saved to: %s\n", textFilename)
}
