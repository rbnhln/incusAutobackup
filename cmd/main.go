package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rbnhln/incusAutobackup/internal/config"
	"github.com/rbnhln/incusAutobackup/internal/vcs"
)

type OnboardOptions struct {
	SourceURL   string
	SourceToken string
	TargetURL   string
	TargetToken string
	IABCredDIr  string
	UUID        string
}

var (
	version = vcs.Version()
)

type application struct {
	logger *slog.Logger
	config config.Config
}

func main() {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	if len(os.Args) > 1 && os.Args[1] == "onboard" {
		cfgDir, err := os.UserConfigDir()
		if err != nil {
			home, _ := os.UserHomeDir()
			cfgDir = filepath.Join(home, ".config")
		}
		defaultCredDir := filepath.Join(cfgDir, "incusAutobackup")

		onboardFlags := flag.NewFlagSet("onboard", flag.ExitOnError)
		sourceURL := onboardFlags.String("sourceURL", "", "URL for the Source host")
		sourceToken := onboardFlags.String("sourceToken", "", "Token for the source host")
		targetURL := onboardFlags.String("targetURL", "", "URL for the Target host")
		targetToken := onboardFlags.String("targetToken", "", "Token for the target host")
		iabCredDir := onboardFlags.String("iabCredDir", defaultCredDir, "Path to store IAB credentials")
		configPath := onboardFlags.String("outConfig", "./", "Path to initial config file (default: ./)")

		_ = onboardFlags.Parse(os.Args[2:])

		unique_id := uuid.NewString()

		opts := OnboardOptions{
			SourceURL:   *sourceURL,
			SourceToken: *sourceToken,
			TargetURL:   *targetURL,
			TargetToken: *targetToken,
			IABCredDIr:  *iabCredDir,
			UUID:        unique_id,
		}

		err = Onboard(opts)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		configFile := filepath.Join(*configPath, "config.json")

		newCfg := config.NewPostOnboardConfig(opts.IABCredDIr, opts.SourceURL, opts.TargetURL, unique_id)
		err = config.Write(configFile, newCfg)
		if err != nil {
			logger.Error("failed to write config", "error", err)
			os.Exit(1)
		}

		logger.Info("Onboarding completed! Bye.")
		os.Exit(0)
	}

	dryRunPrune := flag.Bool("dryRunPrune", false, "do not perform the pruning step")
	dryRuneCopy := flag.Bool("dryRunCopy", false, "do not perform the copy and snapshot step")
	dryRun := flag.Bool("dryRun", false, "Do not perform any pruning, copy or snapshot actions")
	iosfix := flag.Bool("iOSfix", true, "applies the source retention policy to the target")
	displayVersion := flag.Bool("version", false, "Display version and exit")
	logLevel := flag.String("log-level", "info", "Log level: debug|info|warn|error")

	flag.Parse()

	if *dryRun {
		*dryRunPrune = true
		*dryRuneCopy = true
	}

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     parseLogLevel(*logLevel),
		AddSource: true,
	}))

	cfg, err := config.Load("./config.json")
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	cfg.IAB.DryRunPrune = *dryRunPrune
	cfg.IAB.DryRunCopy = *dryRuneCopy
	cfg.IAB.IncusOSfix = *iosfix

	app := &application{
		config: *cfg,
		logger: logger,
	}

	err = app.serve()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func parseLogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
