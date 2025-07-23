package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mohammedteir/telegram-video-downloader-bot/internal/utils"
)

// LanguageManager handles loading and retrieving localized strings
type LanguageManager struct {
	languages     map[string]map[string]string
	defaultLang   string
	languagesPath string
	logger        *utils.EnhancedLogger
	mu            sync.RWMutex
}

// NewLanguageManager creates a new language manager
func NewLanguageManager(languagesPath string, defaultLang string, logger *utils.EnhancedLogger) (*LanguageManager, error) {
	manager := &LanguageManager{
		languages:     make(map[string]map[string]string),
		defaultLang:   defaultLang,
		languagesPath: languagesPath,
		logger:        logger,
	}

	// Load all language files
	if err := manager.LoadLanguages(); err != nil {
		return nil, err
	}

	return manager, nil
}

// LoadLanguages loads all language files from the languages directory
func (lm *LanguageManager) LoadLanguages() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Create languages directory if it doesn't exist
	if err := os.MkdirAll(lm.languagesPath, 0755); err != nil {
		return fmt.Errorf("failed to create languages directory: %w", err)
	}

	// Read all files in the languages directory
	files, err := ioutil.ReadDir(lm.languagesPath)
	if err != nil {
		return fmt.Errorf("failed to read languages directory: %w", err)
	}

	// Load each language file
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process JSON files
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Extract language code from filename (e.g., "en.json" -> "en")
		langCode := strings.TrimSuffix(file.Name(), ".json")

		// Load language file
		langPath := filepath.Join(lm.languagesPath, file.Name())
		langData, err := ioutil.ReadFile(langPath)
		if err != nil {
			lm.logger.Error("Failed to read language file %s: %v", langPath, err)
			continue
		}

		// Parse JSON
		var langStrings map[string]string
		if err := json.Unmarshal(langData, &langStrings); err != nil {
			lm.logger.Error("Failed to parse language file %s: %v", langPath, err)
			continue
		}

		// Store language strings
		lm.languages[langCode] = langStrings
		lm.logger.Info("Loaded language file: %s with %d strings", langPath, len(langStrings))
	}

	// Check if default language is loaded
	if _, ok := lm.languages[lm.defaultLang]; !ok {
		// If no languages are loaded, create default language file
		if len(lm.languages) == 0 {
			lm.logger.Warn("No language files found, creating default language file")
			if err := lm.createDefaultLanguageFile(); err != nil {
				return fmt.Errorf("failed to create default language file: %w", err)
			}
		} else {
			// Use first available language as default
			for langCode := range lm.languages {
				lm.defaultLang = langCode
				lm.logger.Warn("Default language %s not found, using %s instead", lm.defaultLang, langCode)
				break
			}
		}
	}

	return nil
}

// createDefaultLanguageFile creates a default language file with English strings
func (lm *LanguageManager) createDefaultLanguageFile() error {
	// Default English strings
	defaultStrings := map[string]string{
		// Welcome messages
		"welcome_new":      "Welcome to the Video Downloader Bot! Please select your preferred language:",
		"welcome_back":     "Welcome back! Send a video link to download it.",
		
		// Help messages
		"help_title":       "Video Downloader Bot Help",
		"help_usage":       "How to use:",
		"help_usage_1":     "1. Simply send a video link from YouTube, Twitter, Instagram, etc.",
		"help_usage_2":     "2. The bot will download and send you:",
		"help_usage_2_1":   "   - Best quality video",
		"help_usage_2_2":   "   - Subtitle-embedded video (if captions available)",
		"help_usage_2_3":   "   - Audio-only file",
		"help_usage_2_4":   "   - Subtitle file (if available)",
		"help_commands":    "Commands:",
		"help_cmd_start":   "/start - Start the bot",
		"help_cmd_help":    "/help - Show this help message",
		"help_cmd_lang":    "/lang - Change language settings",
		"help_cmd_about":   "/about - About this bot",
		"help_lang":        "Language Settings:",
		"help_lang_desc":   "You can change your interface language and preferred caption language using the /lang command.",
		
		// About message
		"about":            "This bot downloads and sends: best video, best audio, and subtitles in your preferred language. It also embeds captions into a video version if available. Developed by MohammedTeir.",
		
		// Language settings
		"lang_select":      "Please select what you want to change:",
		"lang_interface":   "Interface Language",
		"lang_caption":     "Caption Language",
		"lang_choose_interface": "Choose Interface Language:",
		"lang_choose_caption":   "Choose Caption Language:",
		"lang_updated_interface": "Interface language changed to English!",
		"lang_updated_caption":   "Caption language updated!",
		
		// Download messages
		"invalid_url":      "Please send a valid video URL.",
		"processing":       "Processing your video. This may take a while...",
		"download_error":   "Failed to download video. Please try again later.",
		"download_completed": "Download completed! Sending files...",
		"video_with_subs":  "Video with embedded subtitles",
		"all_files_sent":   "All files sent! Send another video link to download more.",
		
		// Error messages
		"error_general":    "An error occurred. Please try again later.",
		"error_rate_limit": "You've reached the rate limit. Please try again later.",
		
		// Button labels
		"btn_ar":           "العربية 🇸🇦",
		"btn_en":           "English 🇬🇧",
		"btn_de":           "Deutsch 🇩🇪",
		"btn_fr":           "Français 🇫🇷",
	}

	// Create default language file
	langPath := filepath.Join(lm.languagesPath, fmt.Sprintf("%s.json", lm.defaultLang))
	langData, err := json.MarshalIndent(defaultStrings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal default language strings: %w", err)
	}

	if err := ioutil.WriteFile(langPath, langData, 0644); err != nil {
		return fmt.Errorf("failed to write default language file: %w", err)
	}

	// Load default language
	lm.languages[lm.defaultLang] = defaultStrings
	lm.logger.Info("Created default language file: %s", langPath)

	return nil
}

// GetString returns a localized string for the given key and language
func (lm *LanguageManager) GetString(langCode string, key string) string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Check if language exists
	langStrings, ok := lm.languages[langCode]
	if !ok {
		// Fallback to default language
		langStrings, ok = lm.languages[lm.defaultLang]
		if !ok {
			// If default language doesn't exist, return the key
			return key
		}
	}

	// Check if key exists in language
	value, ok := langStrings[key]
	if !ok {
		// Check if key exists in default language
		if langCode != lm.defaultLang {
			defaultStrings, ok := lm.languages[lm.defaultLang]
			if ok {
				value, ok = defaultStrings[key]
				if ok {
					return value
				}
			}
		}
		// If key doesn't exist in any language, return the key
		return key
	}

	return value
}

// GetDefaultLanguage returns the default language code
func (lm *LanguageManager) GetDefaultLanguage() string {
	return lm.defaultLang
}

// GetAvailableLanguages returns a list of available language codes
func (lm *LanguageManager) GetAvailableLanguages() []string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	languages := make([]string, 0, len(lm.languages))
	for langCode := range lm.languages {
		languages = append(languages, langCode)
	}

	return languages
}

// AddOrUpdateString adds or updates a string in a language file
func (lm *LanguageManager) AddOrUpdateString(langCode string, key string, value string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Check if language exists
	langStrings, ok := lm.languages[langCode]
	if !ok {
		// Create new language
		langStrings = make(map[string]string)
		lm.languages[langCode] = langStrings
	}

	// Add or update string
	langStrings[key] = value

	// Save language file
	langPath := filepath.Join(lm.languagesPath, fmt.Sprintf("%s.json", langCode))
	langData, err := json.MarshalIndent(langStrings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal language strings: %w", err)
	}

	if err := ioutil.WriteFile(langPath, langData, 0644); err != nil {
		return fmt.Errorf("failed to write language file: %w", err)
	}

	return nil
}

// ReloadLanguages reloads all language files
func (lm *LanguageManager) ReloadLanguages() error {
	return lm.LoadLanguages()
}
