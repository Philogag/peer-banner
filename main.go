package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/philogag/peer-banner/internal/api"
	"github.com/philogag/peer-banner/internal/ban"
	"github.com/philogag/peer-banner/internal/config"
	"github.com/philogag/peer-banner/internal/detector"
	"github.com/philogag/peer-banner/internal/output"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
	once       = flag.Bool("once", false, "Run detection once and exit")
	dryRun     = flag.Bool("dry-run", false, "Run in dry-run mode (no file writes)")
	version    = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	if *version {
		fmt.Println("qBittorrent Leecher Banner v1.0.0")
		fmt.Println("A tool to detect leecher peers and generate ban lists")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override dry-run if flag is set
	if *dryRun {
		cfg.App.DryRun = true
	}

	// Set log level
	setLogLevel(cfg.App.LogLevel)

	log.Printf("Starting qBittorrent Leecher Banner...")
	log.Printf("Config: %s, Dry Run: %v", *configPath, cfg.App.DryRun)

	// Create ban manager
	banManager, err := ban.NewManager(cfg.App.GetStateFile())
	if err != nil {
		log.Printf("Warning: Failed to create ban manager: %v", err)
	}

	// Create output writer
	writer := output.NewDATWriter(&cfg.Output, banManager)

	// Create detectors for each server
	detectors := make([]*detector.Detector, 0, len(cfg.Servers))
	for _, serverCfg := range cfg.Servers {
		client := api.NewClient(&serverCfg)

		// Test login
		if err := client.Login(); err != nil {
			log.Printf("Warning: Failed to login to %s: %v", serverCfg.Name, err)
			continue
		}

		d, err := detector.NewDetector(client, cfg.Rules, cfg.Whitelist, banManager)
		if err != nil {
			log.Printf("Warning: Failed to create detector for %s: %v", serverCfg.Name, err)
			continue
		}

		detectors = append(detectors, d)
		log.Printf("Connected to qBittorrent server: %s", serverCfg.Name)
	}

	if len(detectors) == 0 {
		log.Fatal("No valid servers available. Exiting.")
	}

	// Run detection once or in a loop
	if *once {
		runDetection(detectors, writer, cfg.App.DryRun)
	} else {
		runLoop(detectors, writer, cfg.App.DryRun, cfg.App.GetInterval())
	}
}

func runDetection(detectors []*detector.Detector, writer *output.DATWriter, dryRun bool) {
	var totalBanned int

	for _, d := range detectors {
		log.Printf("Running detection on %s...", d.Name())

		result, err := d.Detect()
		if err != nil {
			log.Printf("Error during detection: %v", err)
			continue
		}

		// Write result
		if err := writer.Write(result, dryRun); err != nil {
			log.Printf("Error writing DAT file: %v", err)
			continue
		}

		log.Printf("Detection complete: %s", output.GetStats(result))
		totalBanned += result.TotalBanned
	}

	log.Printf("Total banned IPs: %d", totalBanned)
}

func runLoop(detectors []*detector.Detector, writer *output.DATWriter, dryRun bool, interval time.Duration) {
	// Run initial detection
	runDetection(detectors, writer, dryRun)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("Next detection in %v", interval)

	for {
		select {
		case <-ticker.C:
			runDetection(detectors, writer, dryRun)
			log.Printf("Next detection in %v", interval)
		case sig := <-sigChan:
			log.Printf("Received signal %v, shutting down...", sig)
			return
		}
	}
}

func setLogLevel(level string) {
	switch level {
	case "debug":
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "info":
		log.SetFlags(log.LstdFlags)
	case "warn":
		log.SetFlags(log.LstdFlags)
	case "error":
		log.SetFlags(log.LstdFlags)
	default:
		log.SetFlags(log.LstdFlags)
	}
}
