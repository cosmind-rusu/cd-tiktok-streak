package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

const cdrusuDefaultConfigFile = "config.json"
const cdrusuPlaywrightInstallVersion = "v0.5700.1"

var (
	conversationListSelectors = []string{
		"[data-e2e='dm-new-conversation-list']",
		"[data-e2e='dm-conversation-list']",
	}
	conversationItemSelectors = []string{
		"[data-e2e='dm-new-conversation-item']",
		"[data-e2e='dm-conversation-item']",
	}
	nicknameSelectors = []string{
		"[data-e2e='dm-new-conversation-nickname']",
		"p[class*='PInfoNickname']",
		"[data-e2e='conversation-nickname']",
	}
	conversationSearchSelectors = []string{
		"[data-e2e='dm-search-input'] input",
		"[data-e2e='dm-search-input']",
		"input[placeholder*='Search']",
		"input[placeholder*='search']",
		"input[placeholder*='Buscar']",
		"input[placeholder*='buscar']",
		"input[aria-label*='Search']",
		"input[aria-label*='search']",
		"input[type='search']",
	}
	editorSelectors = []string{
		"[contenteditable='true']",
		"div[role='textbox']",
		"[data-e2e='chat-input']",
	}
	passkeyDismissSelectors = []string{
		"button:has-text('Maybe later')",
		"button:has-text('Maybe Later')",
		"button:has-text('Not now')",
		"button:has-text('Remind me later')",
	}
)

type Config struct {
	RunOnce        bool     `json:"run_once"`
	Headless       bool     `json:"headless"`
	TargetUsers    []string `json:"target_users"`
	Message        string   `json:"message"`
	Schedule       Schedule `json:"schedule"`
	CookiesFile    string   `json:"cookies_file"`
	LogFile        string   `json:"log_file"`
	UserAgent      string   `json:"user_agent"`
	MessagesURL    string   `json:"messages_url"`
	BrowserChannel string   `json:"browser_channel"`
	Locale         string   `json:"locale"`
	TimezoneID     string   `json:"timezone_id"`
}

type Schedule struct {
	Enabled bool   `json:"enabled"`
	Time    string `json:"time"`
}

type ExportedCookie struct {
	Name           string   `json:"name"`
	Value          string   `json:"value"`
	Domain         string   `json:"domain"`
	Path           string   `json:"path"`
	Secure         bool     `json:"secure"`
	HTTPOnly       bool     `json:"httpOnly"`
	SameSite       string   `json:"sameSite"`
	ExpirationDate *float64 `json:"expirationDate"`
}

func cdrusuDefaultConfig() Config {
	return Config{
		RunOnce:     true,
		Headless:    true,
		TargetUsers: []string{"username1"},
		Message:     ".",
		Schedule: Schedule{
			Enabled: false,
			Time:    "00:02",
		},
		CookiesFile:    "cookies.json",
		LogFile:        "cd-tiktok-streak.log",
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
		MessagesURL:    "https://www.tiktok.com/messages",
		BrowserChannel: "chrome",
		Locale:         "en-US",
		TimezoneID:     "Europe/Madrid",
	}
}

func main() {
	configPath := flag.String("config", cdrusuDefaultConfigFile, "Path to config JSON")
	runOnceOverride := flag.Bool("run-once", false, "Run immediately once regardless of schedule")
	flag.Parse()

	cfg, created, err := cdrusuLoadConfig(*configPath)
	if err != nil {
		log.Fatalf("could not load config: %v", err)
	}
	if created {
		log.Fatalf("created %s with default values; update it and run again", *configPath)
	}
	cfg = cdrusuResolveConfigPaths(cfg, *configPath)

	logger, closeLog, err := cdrusuBuildLogger(cfg.LogFile)
	if err != nil {
		log.Fatalf("could not initialize logger: %v", err)
	}
	defer closeLog()

	if *runOnceOverride {
		cfg.RunOnce = true
		cfg.Schedule.Enabled = false
	}

	logger.Printf("starting bot; headless=%t run_once=%t targets=%d", cfg.Headless, cfg.RunOnce, len(cfg.TargetUsers))

	if err := cdrusuValidateConfig(cfg); err != nil {
		logger.Fatalf("invalid config: %v", err)
	}

	if cfg.RunOnce || !cfg.Schedule.Enabled {
		if err := cdrusuRunBot(cfg, logger); err != nil {
			logger.Fatalf("run failed: %v", err)
		}
		return
	}

	if err := cdrusuRunScheduled(cfg, logger); err != nil {
		logger.Fatalf("scheduler failed: %v", err)
	}
}

func cdrusuLoadConfig(path string) (Config, bool, error) {
	cfg := cdrusuDefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			payload, marshalErr := json.MarshalIndent(cfg, "", "  ")
			if marshalErr != nil {
				return Config{}, false, marshalErr
			}
			if writeErr := os.WriteFile(path, payload, 0o644); writeErr != nil {
				return Config{}, false, writeErr
			}
			return cfg, true, nil
		}
		return Config{}, false, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, false, err
	}

	return cfg, false, nil
}

func cdrusuValidateConfig(cfg Config) error {
	if len(cfg.TargetUsers) == 0 {
		return errors.New("target_users cannot be empty")
	}
	if strings.TrimSpace(cfg.Message) == "" {
		return errors.New("message cannot be empty")
	}
	if strings.TrimSpace(cfg.CookiesFile) == "" {
		return errors.New("cookies_file cannot be empty")
	}
	if _, err := cdrusuLoadLocation(cfg.TimezoneID); err != nil {
		return fmt.Errorf("invalid timezone_id: %w", err)
	}
	if !cfg.RunOnce && cfg.Schedule.Enabled {
		if _, err := cdrusuParseDailyTime(cfg.Schedule.Time); err != nil {
			return fmt.Errorf("invalid schedule.time: %w", err)
		}
	}
	return nil
}

func cdrusuResolveConfigPaths(cfg Config, configPath string) Config {
	configDir := filepath.Dir(configPath)
	if absConfigPath, err := filepath.Abs(configPath); err == nil {
		configDir = filepath.Dir(absConfigPath)
	}

	cfg.CookiesFile = cdrusuResolvePath(configDir, cfg.CookiesFile)
	cfg.LogFile = cdrusuResolvePath(configDir, cfg.LogFile)
	return cfg
}

func cdrusuResolvePath(baseDir string, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(baseDir, trimmed)
}

func cdrusuBuildLogger(logPath string) (*log.Logger, func(), error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	logger.SetOutput(&teeWriter{stdout: os.Stdout, file: file})

	return logger, func() { _ = file.Close() }, nil
}

type teeWriter struct {
	stdout *os.File
	file   *os.File
}

func (w *teeWriter) Write(p []byte) (int, error) {
	if _, err := w.stdout.Write(p); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func cdrusuRunScheduled(cfg Config, logger *log.Logger) error {
	targetClock, err := cdrusuParseDailyTime(cfg.Schedule.Time)
	if err != nil {
		return err
	}
	location, err := cdrusuLoadLocation(cfg.TimezoneID)
	if err != nil {
		return err
	}

	var lastRun string
	for {
		now := time.Now().In(location)
		nextRun := cdrusuNextScheduledRun(now, targetClock.Hour(), targetClock.Minute())

		wait := time.Until(nextRun)
		logger.Printf(
			"next run scheduled for %s (%s)",
			nextRun.Format(time.RFC3339),
			location.String(),
		)
		time.Sleep(wait)

		today := time.Now().In(location).Format("2006-01-02")
		if today == lastRun {
			continue
		}

		if err := cdrusuRunBot(cfg, logger); err != nil {
			logger.Printf("scheduled run failed: %v", err)
		} else {
			logger.Printf("scheduled run completed")
		}
		lastRun = today
	}
}

func cdrusuParseDailyTime(value string) (time.Time, error) {
	return time.Parse("15:04", value)
}

func cdrusuLoadLocation(timezoneID string) (*time.Location, error) {
	trimmed := strings.TrimSpace(timezoneID)
	if trimmed == "" {
		return time.Local, nil
	}

	location, err := time.LoadLocation(trimmed)
	if err != nil {
		return nil, err
	}
	return location, nil
}

func cdrusuNextScheduledRun(now time.Time, hour int, minute int) time.Time {
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !now.Before(nextRun) {
		nextRun = time.Date(now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, now.Location())
	}
	return nextRun
}

func cdrusuOpenMessagesPage(page playwright.Page, url string) error {
	if _, err := page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(60_000),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err == nil {
		return nil
	}

	_, err := page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(90_000),
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	return err
}

func cdrusuRunBot(cfg Config, logger *log.Logger) error {
	cookies, err := cdrusuLoadCookies(cfg.CookiesFile)
	if err != nil {
		return err
	}

	pw, err := playwright.Run()
	if err != nil {
		if strings.Contains(err.Error(), "please install the driver") {
			return fmt.Errorf("playwright is not installed; run `go run github.com/playwright-community/playwright-go/cmd/playwright@%s install` first", cdrusuPlaywrightInstallVersion)
		}
		return fmt.Errorf("could not start playwright: %w", err)
	}
	defer cdrusuCloseWithTimeout(logger, "playwright", 5*time.Second, pw.Stop)

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.Headless),
		Channel:  playwright.String(cfg.BrowserChannel),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--mute-audio",
			"--no-sandbox",
		},
	})
	if err != nil {
		return fmt.Errorf("could not launch browser: %w", err)
	}
	defer cdrusuCloseWithTimeout(logger, "browser", 5*time.Second, func() error {
		return browser.Close()
	})

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(cfg.UserAgent),
		Locale:    playwright.String(cfg.Locale),
		TimezoneId: func() *string {
			if strings.TrimSpace(cfg.TimezoneID) == "" {
				return nil
			}
			return playwright.String(cfg.TimezoneID)
		}(),
	})
	if err != nil {
		return fmt.Errorf("could not create browser context: %w", err)
	}
	defer cdrusuCloseWithTimeout(logger, "context", 5*time.Second, func() error {
		return context.Close()
	})

	if err := context.AddCookies(cookies); err != nil {
		return fmt.Errorf("could not add cookies: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return fmt.Errorf("could not create page: %w", err)
	}
	defer cdrusuCloseWithTimeout(logger, "page", 5*time.Second, func() error {
		return page.Close()
	})

	if err := cdrusuOpenMessagesPage(page, cfg.MessagesURL); err != nil {
		return fmt.Errorf("could not open messages page: %w", err)
	}

	cdrusuDismissPasskeyPopup(page, logger)

	if _, err := cdrusuWaitForAnySelector(page, conversationListSelectors, 30_000); err != nil {
		logger.Printf("warning: conversation list did not appear cleanly: %v", err)
	}

	successes := 0
	for _, user := range cfg.TargetUsers {
		logger.Printf("processing %q", user)
		if err := cdrusuOpenConversation(page, user); err != nil {
			logger.Printf("warning: could not open conversation for %q: %v", user, err)
			continue
		}
		if err := cdrusuSendMessage(page, cfg.Message); err != nil {
			logger.Printf("warning: could not send message to %q: %v", user, err)
			continue
		}
		successes++
		logger.Printf("message sent to %q", user)
		time.Sleep(2 * time.Second)
	}

	if successes == 0 {
		return errors.New("no messages were sent")
	}

	logger.Printf("finished; %d/%d messages sent", successes, len(cfg.TargetUsers))
	return nil
}

func cdrusuCloseWithTimeout(logger *log.Logger, name string, timeout time.Duration, closeFn func() error) {
	done := make(chan error, 1)
	go func() {
		done <- closeFn()
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Printf("warning: could not close %s cleanly: %v", name, err)
		}
	case <-time.After(timeout):
		logger.Printf("warning: timed out while closing %s", name)
	}
}

func cdrusuLoadCookies(path string) ([]playwright.OptionalCookie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read cookies file %q: %w", path, err)
	}

	var exported []ExportedCookie
	if err := json.Unmarshal(data, &exported); err != nil {
		return nil, fmt.Errorf("cookies file is not valid JSON: %w", err)
	}

	if len(exported) == 0 {
		return nil, errors.New("cookies file is empty")
	}

	result := make([]playwright.OptionalCookie, 0, len(exported))
	for _, item := range exported {
		cookie := playwright.OptionalCookie{
			Name:  item.Name,
			Value: item.Value,
		}
		if item.Domain != "" {
			cookie.Domain = playwright.String(item.Domain)
		} else {
			cookie.Domain = playwright.String(".tiktok.com")
		}
		if item.Path != "" {
			cookie.Path = playwright.String(item.Path)
		} else {
			cookie.Path = playwright.String("/")
		}
		cookie.Secure = playwright.Bool(item.Secure)
		cookie.HttpOnly = playwright.Bool(item.HTTPOnly)
		if item.ExpirationDate != nil {
			cookie.Expires = item.ExpirationDate
		}
		if sameSite := cdrusuNormalizeSameSite(item.SameSite); sameSite != nil {
			cookie.SameSite = sameSite
		}
		result = append(result, cookie)
	}

	return result, nil
}

func cdrusuNormalizeSameSite(value string) *playwright.SameSiteAttribute {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lax":
		return playwright.SameSiteAttributeLax
	case "strict":
		return playwright.SameSiteAttributeStrict
	case "none", "no_restriction":
		return playwright.SameSiteAttributeNone
	default:
		return nil
	}
}

func cdrusuDismissPasskeyPopup(page playwright.Page, logger *log.Logger) {
	selector, err := cdrusuWaitForAnySelector(page, passkeyDismissSelectors, 5_000)
	if err != nil {
		return
	}
	if err := page.Locator(selector).First().Click(); err != nil {
		logger.Printf("warning: could not dismiss passkey popup: %v", err)
	}
}

func cdrusuOpenConversation(page playwright.Page, username string) error {
	if err := cdrusuSearchConversation(page, username); err == nil {
		time.Sleep(1500 * time.Millisecond)
	}

	itemSelector, err := cdrusuWaitForAnySelector(page, conversationItemSelectors, 20_000)
	if err != nil {
		return err
	}

	items := page.Locator(itemSelector)
	count, err := items.Count()
	if err != nil {
		return err
	}

	username = cdrusuNormalizeConversationKey(username)
	for i := 0; i < count; i++ {
		item := items.Nth(i)
		name, readErr := cdrusuReadConversationName(item)
		if readErr != nil {
			continue
		}
		if !cdrusuConversationMatches(username, name) {
			continue
		}
		if err := item.Click(); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("conversation not found in visible list")
}

func cdrusuConversationMatches(target string, visibleName string) bool {
	normalizedVisible := cdrusuNormalizeConversationKey(visibleName)
	if normalizedVisible == "" || target == "" {
		return false
	}
	if normalizedVisible == target {
		return true
	}
	return strings.Contains(normalizedVisible, target) || strings.Contains(target, normalizedVisible)
}

func cdrusuNormalizeConversationKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.ReplaceAll(value, "\u200d", "")

	nonWord := regexp.MustCompile(`[^a-z0-9\s]+`)
	value = nonWord.ReplaceAllString(value, " ")
	return strings.Join(strings.Fields(value), " ")
}

func cdrusuSearchConversation(page playwright.Page, username string) error {
	selector, err := cdrusuWaitForAnySelector(page, conversationSearchSelectors, 5_000)
	if err != nil {
		return err
	}

	searchBox := page.Locator(selector).First()
	if err := searchBox.Click(); err != nil {
		return err
	}
	if err := searchBox.Fill(""); err != nil {
		// Some search fields are wrapped and do not support Fill cleanly.
	}
	if err := searchBox.Press("Control+A"); err != nil {
		// Ignore if the field does not support this shortcut.
	}
	if err := searchBox.Type(username, playwright.LocatorTypeOptions{Delay: playwright.Float(40)}); err != nil {
		return err
	}
	if err := searchBox.Press("Enter"); err != nil {
		// Not all inbox search fields need Enter.
	}
	return nil
}

func cdrusuReadConversationName(item playwright.Locator) (string, error) {
	for _, selector := range nicknameSelectors {
		loc := item.Locator(selector).First()
		count, err := loc.Count()
		if err != nil || count == 0 {
			continue
		}
		text, err := loc.TextContent()
		if err != nil {
			continue
		}
		if strings.TrimSpace(text) != "" {
			return text, nil
		}
	}
	return "", errors.New("nickname not found")
}

func cdrusuSendMessage(page playwright.Page, message string) error {
	selector, err := cdrusuWaitForAnySelector(page, editorSelectors, 15_000)
	if err != nil {
		return err
	}

	editor := page.Locator(selector).First()
	if err := editor.Click(); err != nil {
		return err
	}
	if err := editor.Fill(""); err != nil {
		// Some TikTok editors are contenteditable and do not support Fill.
	}
	if err := editor.Type(message, playwright.LocatorTypeOptions{Delay: playwright.Float(50)}); err != nil {
		return err
	}
	return editor.Press("Enter")
}

func cdrusuWaitForAnySelector(page playwright.Page, selectors []string, timeoutMs float64) (string, error) {
	for _, selector := range selectors {
		if _, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(timeoutMs),
			State:   playwright.WaitForSelectorStateVisible,
		}); err == nil {
			return selector, nil
		}
	}
	return "", fmt.Errorf("none of the selectors matched: %s", strings.Join(selectors, ", "))
}
