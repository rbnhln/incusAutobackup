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

type OnboardTarget struct {
	Name  string
	URL   string
	Token string
	Type  config.TargetType
}

type OnboardOptions struct {
	SourceURL   string
	SourceToken string
	Targets     []OnboardTarget
	IABCredDIr  string
	UUID        string
}

type targetSpecs []OnboardTarget

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
		iabCredDir := onboardFlags.String("iabCredDir", defaultCredDir, "Path to store IAB credentials")
		configPath := onboardFlags.String("outConfig", "./", "Path to initial config file (default: ./)")

		var targetList targetSpecs
		onboardFlags.Var(&targetList, "target", `Target spec: name=<n>,url=<u>,token=<t>[,type=incus] (repeatable)`)

		_ = onboardFlags.Parse(os.Args[2:])

		if *sourceURL == "" || *sourceToken == "" {
			logger.Error("onboard requires --sourceURL and --sourceToken")
			os.Exit(1)
		}
		if len(targetList) == 0 {
			logger.Error("onboard requires at least one --target")
			os.Exit(1)
		}

		unique_id := uuid.NewString()

		opts := OnboardOptions{
			SourceURL:   *sourceURL,
			SourceToken: *sourceToken,
			Targets:     []OnboardTarget(targetList),
			IABCredDIr:  *iabCredDir,
			UUID:        unique_id,
		}

		err = Onboard(opts)
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		configFile := filepath.Join(*configPath, "config.json")

		targetHosts := make([]config.TargetHost, 0, len(opts.Targets))
		for _, t := range opts.Targets {
			targetHosts = append(targetHosts, config.TargetHost{
				Name: t.Name,
				Type: t.Type,
				URL:  t.URL,
			})
		}

		newCfg := config.NewPostOnboardConfig(opts.IABCredDIr, opts.SourceURL, unique_id, targetHosts)
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

func (t *targetSpecs) String() string { return "" }

// --target "name=t1,url=https://1.2.3.4:8443,token=abc,type=incus"
func (t *targetSpecs) Set(v string) error {
	parts := strings.Split(v, ",")
	item := OnboardTarget{Type: config.TargetTypeIncus}

	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid target segment %q (expected key=value)", p)
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		val := strings.TrimSpace(kv[1])

		switch key {
		case "name":
			item.Name = val
		case "url":
			item.URL = val
		case "token":
			item.Token = val
		case "type":
			item.Type = config.TargetType(strings.ToLower(val))
		default:
			return fmt.Errorf("unknown target key %q", key)
		}
	}

	if item.Name == "" || item.URL == "" || item.Token == "" {
		return fmt.Errorf("target requires name,url,token")
	}
	if item.Type == "" {
		item.Type = config.TargetTypeIncus
	}

	*t = append(*t, item)
	return nil
}
