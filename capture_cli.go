package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	// Define flags
	targetURL := flag.String("url", "", "Target URL to capture (required)")
	outputDir := flag.String("output", "", "Output directory (default: capture_<timestamp>)")
	captureAPIs := flag.Bool("apis", true, "Capture API requests/responses")
	captureHTML := flag.Bool("html", true, "Capture HTML content")
	captureResources := flag.Bool("resources", true, "Extract resource URLs")
	captureScreenshot := flag.Bool("screenshot", true, "Capture screenshot")
	waitTime := flag.Int("wait", 5, "Wait time in seconds before capturing")
	simpleMode := flag.Bool("simple", false, "Use simple HTTP client mode (no JavaScript rendering)")
	help := flag.Bool("help", false, "Show help")

	flag.Parse()

	if *help || *targetURL == "" {
		printHelp()
		os.Exit(0)
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("═══════════════════════════════════")
	log.Printf("   WEB CAPTURE TOOL")
	log.Printf("═══════════════════════════════════")

	// Use simple mode if requested
	if *simpleMode {
		outputPath := *outputDir
		if outputPath == "" {
			outputPath = fmt.Sprintf("capture_%d", time.Now().Unix())
		}
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
		
		htmlPath := fmt.Sprintf("%s/page.html", outputPath)
		if err := SimpleCaptureHTML(*targetURL, htmlPath); err != nil {
			log.Fatalf("Capture failed: %v", err)
		}
		
		log.Printf("✓ Simple capture completed successfully")
		log.Printf("Output directory: %s", outputPath)
		return
	}

	// Full capture mode with Chrome
	config := CaptureConfig{
		URL:               *targetURL,
		OutputDir:         *outputDir,
		CaptureAPIs:       *captureAPIs,
		CaptureHTML:       *captureHTML,
		CaptureResources:  *captureResources,
		CaptureScreenshot: *captureScreenshot,
		WaitTime:          time.Duration(*waitTime) * time.Second,
	}

	wc := NewWebCapture(config)
	if err := wc.Capture(); err != nil {
		log.Fatalf("Capture failed: %v", err)
	}

	log.Printf("═══════════════════════════════════")
	log.Printf("✓ Capture completed successfully!")
	log.Printf("═══════════════════════════════════")
}

func printHelp() {
	fmt.Println("Web Capture Tool - Capture everything from a website")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run . -url <target-url> [options]")
	fmt.Println()
	fmt.Println("Required:")
	fmt.Println("  -url string")
	fmt.Println("        Target URL to capture")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -output string")
	fmt.Println("        Output directory (default: capture_<timestamp>)")
	fmt.Println("  -apis")
	fmt.Println("        Capture API requests/responses (default: true)")
	fmt.Println("  -html")
	fmt.Println("        Capture HTML content (default: true)")
	fmt.Println("  -resources")
	fmt.Println("        Extract resource URLs (default: true)")
	fmt.Println("  -screenshot")
	fmt.Println("        Capture screenshot (default: true)")
	fmt.Println("  -wait int")
	fmt.Println("        Wait time in seconds before capturing (default: 5)")
	fmt.Println("  -simple")
	fmt.Println("        Use simple HTTP client mode without JavaScript (default: false)")
	fmt.Println("  -help")
	fmt.Println("        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Capture everything from a website")
	fmt.Println("  go run . -url https://example.com")
	fmt.Println()
	fmt.Println("  # Capture with custom output directory")
	fmt.Println("  go run . -url https://example.com -output my_capture")
	fmt.Println()
	fmt.Println("  # Only capture HTML and APIs, no screenshot")
	fmt.Println("  go run . -url https://example.com -screenshot=false")
	fmt.Println()
	fmt.Println("  # Simple mode (fast, no JavaScript rendering)")
	fmt.Println("  go run . -url https://example.com -simple")
	fmt.Println()
	fmt.Println("What gets captured:")
	fmt.Println("  ✓ Full HTML content (including dynamically loaded)")
	fmt.Println("  ✓ All API requests and responses")
	fmt.Println("  ✓ Network traffic details")
	fmt.Println("  ✓ Request/response headers")
	fmt.Println("  ✓ Cookies")
	fmt.Println("  ✓ Resource URLs (scripts, stylesheets, images)")
	fmt.Println("  ✓ Full page screenshot")
	fmt.Println()
}
