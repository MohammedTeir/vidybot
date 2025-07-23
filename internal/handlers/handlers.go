package handlers

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/mohammedteir/telegram-video-downloader-bot/internal/config"
	"github.com/mohammedteir/telegram-video-downloader-bot/internal/database"
	"github.com/mohammedteir/telegram-video-downloader-bot/internal/downloader"
	"github.com/mohammedteir/telegram-video-downloader-bot/internal/models"
	"github.com/mohammedteir/telegram-video-downloader-bot/internal/utils"

    "go.mongodb.org/mongo-driver/bson/primitive"

	"gopkg.in/telebot.v3"
)

// BotHandler handles Telegram bot interactions
type BotHandler struct {
	bot           *telebot.Bot
	userRepo      *database.UserRepository
	downloadRepo  *database.DownloadRepository
	redisClient   *database.RedisClient
	config        *config.Config
	logger        *utils.Logger
	downloader    *downloader.VideoDownloader
}


// NewBotHandler creates a new bot handler
func NewBotHandler(
	bot *telebot.Bot,
	userRepo *database.UserRepository,
	redisClient *database.RedisClient,
	config *config.Config,
	logger *utils.Logger,
	dependencyPaths map[string]string,
) *BotHandler {

	// Initialize download repository
	
// Create an adapter for the logger to match the expected EnhancedLogger type

enhancedLoggerConfig := &utils.EnhancedLoggerConfig{
    Enabled:      true,
    Level:        utils.LogLevelInfo,
    Path:         config.Log.Path, // Use Path instead of Directory
    MaxSize:      10,
    MaxBackups:   5,
    MaxAge:       30,
    Compress:     true,
    ConsoleLog:   true,
    JSONFormat:   false,
    CallerInfo:   true,
    StackTraces:  true,
    Development:  false,
    RotationTime: 24,
}


enhancedLogger, err := utils.NewEnhancedLogger(enhancedLoggerConfig)
if err != nil {
    // Handle error - fall back to using the regular logger
    logger.Error("Failed to create enhanced logger: %v", err)
    // You might need a fallback solution here
}

	
mongoClient := userRepo.GetClient() // Access the client directly
downloadRepo := database.NewDownloadRepository(mongoClient, config.MongoDB.Database, enhancedLogger)

	
	// Initialize downloader
	
 videoDownloader := downloader.NewVideoDownloader(config.Download.TempDir, enhancedLogger, 3,dependencyPaths) // 3 is the default max retries

	
	return &BotHandler{
		bot:           bot,
		userRepo:      userRepo,
		downloadRepo:  downloadRepo,
		redisClient:   redisClient,
		config:        config,
		logger:        logger,
		downloader:    videoDownloader,
	}
}

// RegisterHandlers registers all bot command handlers
func (h *BotHandler) RegisterHandlers() {
	// Command handlers
	h.bot.Handle("/start", h.handleStart)
	h.bot.Handle("/help", h.handleHelp)
	h.bot.Handle("/about", h.handleAbout)
	h.bot.Handle("/lang", h.handleLanguage)
	
	// Button handlers
	h.bot.Handle(&telebot.InlineButton{Unique: "set_interface_lang"}, h.handleSetInterfaceLanguage)
	h.bot.Handle(&telebot.InlineButton{Unique: "set_caption_lang"}, h.handleSetCaptionLanguage)
	
	// Language selection buttons
	h.bot.Handle(&telebot.InlineButton{Unique: "lang_ar"}, h.handleLanguageSelection)
	h.bot.Handle(&telebot.InlineButton{Unique: "lang_en"}, h.handleLanguageSelection)
	h.bot.Handle(&telebot.InlineButton{Unique: "lang_de"}, h.handleLanguageSelection)
	h.bot.Handle(&telebot.InlineButton{Unique: "lang_fr"}, h.handleLanguageSelection)
	
	// Handle text messages (for URL processing)
	h.bot.Handle(telebot.OnText, h.handleText)
}

// handleStart handles the /start command
func (h *BotHandler) handleStart(c telebot.Context) error {
	chatID := c.Chat().ID
	h.logger.Info("Received /start command from chat ID: %d", chatID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Check if user exists
	user, err := h.userRepo.FindUserByChatID(ctx, chatID)
	if err != nil {
		h.logger.Error("Error finding user: %v", err)
		return c.Send("An error occurred. Please try again later.")
	}
	
	if user == nil {
		// New user
		user = models.NewUser(chatID)
		user, err = h.userRepo.CreateUser(ctx, user)
		if err != nil {
			h.logger.Error("Error creating user: %v", err)
			return c.Send("An error occurred. Please try again later.")
		}
		
		// Send welcome message with language selection
		return h.sendWelcomeMessage(c)
	}
	
	// Returning user
	var welcomeBack string
	switch user.InterfaceLanguage {
	case "ar":
		welcomeBack = "مرحبًا بعودتك! أرسل رابط فيديو لتنزيله."
	case "de":
		welcomeBack = "Willkommen zurück! Sende einen Video-Link zum Herunterladen."
	case "fr":
		welcomeBack = "Bon retour! Envoyez un lien vidéo pour le télécharger."
	default: // English
		welcomeBack = "Welcome back! Send a video link to download it."
	}
	
	return c.Send(welcomeBack)
}

// sendWelcomeMessage sends the welcome message with language selection
func (h *BotHandler) sendWelcomeMessage(c telebot.Context) error {
	welcomeMsg := "Welcome to the Video Downloader Bot! Please select your preferred language:"
	
	// Create language selection buttons
	var buttons [][]telebot.InlineButton
	
	// Add language buttons
	langRow := []telebot.InlineButton{
		{Text: "العربية 🇸🇦", Unique: "lang_ar", Data: "interface"},
		{Text: "English 🇬🇧", Unique: "lang_en", Data: "interface"},
	}
	
	langRow2 := []telebot.InlineButton{
		{Text: "Deutsch 🇩🇪", Unique: "lang_de", Data: "interface"},
		{Text: "Français 🇫🇷", Unique: "lang_fr", Data: "interface"},
	}
	
	buttons = append(buttons, langRow, langRow2)
	
	return c.Send(welcomeMsg, &telebot.ReplyMarkup{
		InlineKeyboard: buttons,
	})
}

// handleHelp handles the /help command
func (h *BotHandler) handleHelp(c telebot.Context) error {
	chatID := c.Chat().ID
	h.logger.Info("Received /help command from chat ID: %d", chatID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get user language preference
	user, err := h.userRepo.FindUserByChatID(ctx, chatID)
	if err != nil {
		h.logger.Error("Error finding user: %v", err)
		return c.Send("An error occurred. Please try again later.")
	}
	
	var helpText string
	if user == nil || user.InterfaceLanguage == "en" {
		helpText = `*Video Downloader Bot Help*

*How to use:*
1. Simply send a video link from YouTube, Twitter, Instagram, etc.
2. The bot will download and send you:
   - Best quality video
   - Subtitle-embedded video (if captions available)
   - Audio-only file
   - Subtitle file (if available)

*Commands:*
/start - Start the bot
/help - Show this help message
/lang - Change language settings
/about - About this bot

*Language Settings:*
You can change your interface language and preferred caption language using the /lang command.`
	} else if user.InterfaceLanguage == "ar" {
		helpText = `*مساعدة بوت تنزيل الفيديو*

*كيفية الاستخدام:*
1. ما عليك سوى إرسال رابط فيديو من يوتيوب أو تويتر أو انستغرام، إلخ.
2. سيقوم البوت بتنزيل وإرسال:
   - فيديو بأفضل جودة
   - فيديو مع ترجمة مدمجة (إذا كانت الترجمة متوفرة)
   - ملف صوتي فقط
   - ملف الترجمة (إذا كان متوفرًا)

*الأوامر:*
/start - بدء البوت
/help - عرض رسالة المساعدة هذه
/lang - تغيير إعدادات اللغة
/about - حول هذا البوت

*إعدادات اللغة:*
يمكنك تغيير لغة الواجهة ولغة الترجمة المفضلة باستخدام الأمر /lang`
	} else if user.InterfaceLanguage == "de" {
		helpText = `*Video Downloader Bot Hilfe*

*Verwendung:*
1. Senden Sie einfach einen Video-Link von YouTube, Twitter, Instagram usw.
2. Der Bot lädt herunter und sendet Ihnen:
   - Video in bester Qualität
   - Video mit eingebetteten Untertiteln (falls verfügbar)
   - Nur-Audio-Datei
   - Untertiteldatei (falls verfügbar)

*Befehle:*
/start - Bot starten
/help - Diese Hilfemeldung anzeigen
/lang - Spracheinstellungen ändern
/about - Über diesen Bot

*Spracheinstellungen:*
Sie können Ihre Oberflächensprache und bevorzugte Untertitelsprache mit dem Befehl /lang ändern.`
	} else if user.InterfaceLanguage == "fr" {
		helpText = `*Aide du Bot de Téléchargement Vidéo*

*Comment utiliser:*
1. Envoyez simplement un lien vidéo de YouTube, Twitter, Instagram, etc.
2. Le bot téléchargera et vous enverra:
   - Vidéo de meilleure qualité
   - Vidéo avec sous-titres intégrés (si disponibles)
   - Fichier audio uniquement
   - Fichier de sous-titres (si disponible)

*Commandes:*
/start - Démarrer le bot
/help - Afficher ce message d'aide
/lang - Modifier les paramètres de langue
/about - À propos de ce bot

*Paramètres de langue:*
Vous pouvez modifier votre langue d'interface et votre langue de sous-titres préférée à l'aide de la commande /lang`
	}
	
	return c.Send(helpText, &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
}

// handleAbout handles the /about command
func (h *BotHandler) handleAbout(c telebot.Context) error {
	chatID := c.Chat().ID
	h.logger.Info("Received /about command from chat ID: %d", chatID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get user language preference
	user, err := h.userRepo.FindUserByChatID(ctx, chatID)
	if err != nil {
		h.logger.Error("Error finding user: %v", err)
		return c.Send("An error occurred. Please try again later.")
	}
	
	var aboutText string
	if user == nil || user.InterfaceLanguage == "en" {
		aboutText = "This bot downloads and sends: best video, best audio, and subtitles in your preferred language. It also embeds captions into a video version if available. Developed by MohammedTeir."
	} else if user.InterfaceLanguage == "ar" {
		aboutText = "يقوم هذا البوت بتنزيل وإرسال: أفضل فيديو، وأفضل صوت، وترجمات بلغتك المفضلة. كما يدمج الترجمات في نسخة الفيديو إذا كانت متوفرة. تم تطويره بواسطة محمد طير."
	} else if user.InterfaceLanguage == "de" {
		aboutText = "Dieser Bot lädt herunter und sendet: bestes Video, besten Audio und Untertitel in Ihrer bevorzugten Sprache. Er bettet auch Untertitel in eine Videoversion ein, falls verfügbar. Entwickelt von MohammedTeir."
	} else if user.InterfaceLanguage == "fr" {
		aboutText = "Ce bot télécharge et envoie: la meilleure vidéo, le meilleur audio et les sous-titres dans votre langue préférée. Il intègre également les sous-titres dans une version vidéo si disponible. Développé par MohammedTeir."
	}
	
	return c.Send(aboutText)
}

// handleLanguage handles the /lang command
func (h *BotHandler) handleLanguage(c telebot.Context) error {
	chatID := c.Chat().ID
	h.logger.Info("Received /lang command from chat ID: %d", chatID)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get user language preference
	user, err := h.userRepo.FindUserByChatID(ctx, chatID)
	if err != nil {
		h.logger.Error("Error finding user: %v", err)
		return c.Send("An error occurred. Please try again later.")
	}
	
	var langText string
	if user == nil || user.InterfaceLanguage == "en" {
		langText = "Please select what you want to change:"
	} else if user.InterfaceLanguage == "ar" {
		langText = "الرجاء تحديد ما تريد تغييره:"
	} else if user.InterfaceLanguage == "de" {
		langText = "Bitte wählen Sie aus, was Sie ändern möchten:"
	} else if user.InterfaceLanguage == "fr" {
		langText = "Veuillez sélectionner ce que vous souhaitez modifier:"
	}
	
	// Create language selection buttons
	var buttons [][]telebot.InlineButton
	
	// Add language type buttons
	var interfaceBtn, captionBtn telebot.InlineButton
	
	if user == nil || user.InterfaceLanguage == "en" {
		interfaceBtn = telebot.InlineButton{Text: "Interface Language", Unique: "set_interface_lang"}
		captionBtn = telebot.InlineButton{Text: "Caption Language", Unique: "set_caption_lang"}
	} else if user.InterfaceLanguage == "ar" {
		interfaceBtn = telebot.InlineButton{Text: "لغة الواجهة", Unique: "set_interface_lang"}
		captionBtn = telebot.InlineButton{Text: "لغة الترجمة", Unique: "set_caption_lang"}
	} else if user.InterfaceLanguage == "de" {
		interfaceBtn = telebot.InlineButton{Text: "Oberflächensprache", Unique: "set_interface_lang"}
		captionBtn = telebot.InlineButton{Text: "Untertitelsprache", Unique: "set_caption_lang"}
	} else if user.InterfaceLanguage == "fr" {
		interfaceBtn = telebot.InlineButton{Text: "Langue d'interface", Unique: "set_interface_lang"}
		captionBtn = telebot.InlineButton{Text: "Langue des sous-titres", Unique: "set_caption_lang"}
	}
	
	buttons = append(buttons, []telebot.InlineButton{interfaceBtn})
	buttons = append(buttons, []telebot.InlineButton{captionBtn})
	
	return c.Send(langText, &telebot.ReplyMarkup{
		InlineKeyboard: buttons,
	})
}

// handleSetInterfaceLanguage handles the interface language selection button
func (h *BotHandler) handleSetInterfaceLanguage(c telebot.Context) error {
	chatID := c.Chat().ID
	h.logger.Info("User %d is setting interface language", chatID)
	
	// Create language selection buttons
	var buttons [][]telebot.InlineButton
	
	// Add language buttons
	langRow := []telebot.InlineButton{
		{Text: "العربية 🇸🇦", Unique: "lang_ar", Data: "interface"},
		{Text: "English 🇬🇧", Unique: "lang_en", Data: "interface"},
	}
	
	langRow2 := []telebot.InlineButton{
		{Text: "Deutsch 🇩🇪", Unique: "lang_de", Data: "interface"},
		{Text: "Français 🇫🇷", Unique: "lang_fr", Data: "interface"},
	}
	
	buttons = append(buttons, langRow, langRow2)
	
	return c.Edit("Choose Interface Language:", &telebot.ReplyMarkup{
		InlineKeyboard: buttons,
	})
}

// handleSetCaptionLanguage handles the caption language selection button
func (h *BotHandler) handleSetCaptionLanguage(c telebot.Context) error {
	chatID := c.Chat().ID
	h.logger.Info("User %d is setting caption language", chatID)
	
	// Create language selection buttons
	var buttons [][]telebot.InlineButton
	
	// Add language buttons
	langRow := []telebot.InlineButton{
		{Text: "العربية 🇸🇦", Unique: "lang_ar", Data: "caption"},
		{Text: "English 🇬🇧", Unique: "lang_en", Data: "caption"},
	}
	
	langRow2 := []telebot.InlineButton{
		{Text: "Deutsch 🇩🇪", Unique: "lang_de", Data: "caption"},
		{Text: "Français 🇫🇷", Unique: "lang_fr", Data: "caption"},
	}
	
	buttons = append(buttons, langRow, langRow2)
	
	return c.Edit("Choose Caption Language:", &telebot.ReplyMarkup{
		InlineKeyboard: buttons,
	})
}

// handleLanguageSelection handles language selection buttons
func (h *BotHandler) handleLanguageSelection(c telebot.Context) error {
	chatID := c.Chat().ID
	data := c.Data()
	
	// Extract language code from button unique identifier
	langCode := c.Callback().Unique[5:] // Remove "lang_" prefix
	
	h.logger.Info("User %d selected language %s for %s", chatID, langCode, data)
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var successMsg string
	
	if data == "interface" {
		// Update interface language
		err := h.userRepo.UpdateUserInterfaceLanguage(ctx, chatID, langCode)
		if err != nil {
			h.logger.Error("Error updating interface language: %v", err)
			return c.Respond(&telebot.CallbackResponse{
				Text: "Error updating language",
			})
		}
		
		// Set success message based on selected language
		switch langCode {
		case "ar":
			successMsg = "تم تغيير لغة الواجهة إلى العربية!"
		case "de":
			successMsg = "Oberflächensprache auf Deutsch geändert!"
		case "fr":
			successMsg = "Langue d'interface changée en français!"
		default:
			successMsg = "Interface language changed to English!"
		}
	} else {
		// Update caption language
		err := h.userRepo.UpdateUserCaptionLanguage(ctx, chatID, langCode)
		if err != nil {
			h.logger.Error("Error updating caption language: %v", err)
			return c.Respond(&telebot.CallbackResponse{
				Text: "Error updating language",
			})
		}
		
		// Get user's interface language for the success message
		user, err := h.userRepo.FindUserByChatID(ctx, chatID)
		if err != nil {
			h.logger.Error("Error finding user: %v", err)
			successMsg = "Caption language updated!"
		} else if user == nil {
			successMsg = "Caption language updated!"
		} else {
			// Set success message based on interface language
			switch user.InterfaceLanguage {
			case "ar":
				successMsg = "تم تغيير لغة الترجمة!"
			case "de":
				successMsg = "Untertitelsprache geändert!"
			case "fr":
				successMsg = "Langue des sous-titres modifiée!"
			default:
				successMsg = "Caption language updated!"
			}
		}
	}
	
	// Respond to callback
	c.Respond(&telebot.CallbackResponse{
		Text: successMsg,
	})
	
	// Edit message to show success
	return c.Edit(successMsg)
}

// handleText handles text messages (for URL processing)
func (h *BotHandler) handleText(c telebot.Context) error {
	chatID := c.Chat().ID
	text := c.Text()
	
	h.logger.Info("Received text from chat ID %d: %s", chatID, text)
	
	// Check if text is a URL
	if !isValidURL(text) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Get user language preference
		user, err := h.userRepo.FindUserByChatID(ctx, chatID)
		if err != nil {
			h.logger.Error("Error finding user: %v", err)
			return c.Send("Please send a valid video URL.")
		}
		
		var invalidURLMsg string
		if user == nil || user.InterfaceLanguage == "en" {
			invalidURLMsg = "Please send a valid video URL."
		} else if user.InterfaceLanguage == "ar" {
			invalidURLMsg = "الرجاء إرسال رابط فيديو صالح."
		} else if user.InterfaceLanguage == "de" {
			invalidURLMsg = "Bitte senden Sie eine gültige Video-URL."
		} else if user.InterfaceLanguage == "fr" {
			invalidURLMsg = "Veuillez envoyer une URL vidéo valide."
		}
		
		return c.Send(invalidURLMsg)
	}
	
	// URL is valid, send processing message
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// Get user language preference
	user, err := h.userRepo.FindUserByChatID(ctx, chatID)
	if err != nil {
		h.logger.Error("Error finding user: %v", err)
		return c.Send("Processing your video. This may take a while...")
	}
	
	var processingMsg string
	if user == nil || user.InterfaceLanguage == "en" {
		processingMsg = "Processing your video. This may take a while..."
	} else if user.InterfaceLanguage == "ar" {
		processingMsg = "جاري معالجة الفيديو الخاص بك. قد يستغرق هذا بعض الوقت..."
	} else if user.InterfaceLanguage == "de" {
		processingMsg = "Ihr Video wird verarbeitet. Dies kann eine Weile dauern..."
	} else if user.InterfaceLanguage == "fr" {
		processingMsg = "Traitement de votre vidéo en cours. Cela peut prendre un moment..."
	}
	
	// Send processing message
	statusMsg, err := h.bot.Send(c.Chat(), processingMsg)
	if err != nil {
		h.logger.Error("Error sending processing message: %v", err)
	}
	
	// Create download request
	downloadRequest := models.NewDownloadRequest(chatID, text)
	downloadRequest, err = h.downloadRepo.CreateDownloadRequest(ctx, downloadRequest)
	if err != nil {
		h.logger.Error("Error creating download request: %v", err)
		return c.Send("An error occurred. Please try again later.")
	}
	
	// Get caption language
	captionLang := "en" // Default to English
	if user != nil {
		captionLang = user.CaptionLanguage
	}
	
	// Process download in a goroutine
	go func() {
		h.processDownload(downloadRequest.ID, chatID, text, captionLang, statusMsg)
	}()
	
	return nil
}

// sendThumbnail sends the thumbnail to the user if it exists
func (h *BotHandler) sendThumbnail(chatID int64, thumbnailPath string, user *models.User) {
    if thumbnailPath == "" || !fileExists(thumbnailPath) {
        h.logger.Debug("No thumbnail to send or file doesn't exist")
        return
    }

    chat := &telebot.Chat{ID: chatID}
    
    // Create caption based on user's language
    var caption string
    if user == nil || user.InterfaceLanguage == "en" {
        caption = "Video thumbnail"
    } else if user.InterfaceLanguage == "ar" {
        caption = "صورة مصغرة للفيديو"
    } else if user.InterfaceLanguage == "de" {
        caption = "Video-Vorschaubild"
    } else if user.InterfaceLanguage == "fr" {
        caption = "Miniature de la vidéo"
    }

    // Send as photo
    photo := &telebot.Photo{
        File:    telebot.FromDisk(thumbnailPath),
        Caption: caption,
    }
    
    _, err := h.bot.Send(chat, photo)
    if err != nil {
        h.logger.Error("Error sending thumbnail: %v", err)
    }
}

// sendAudioFile sends the downloaded audio file to the user with a descriptive name
func (h *BotHandler) sendAudioFile(chat *telebot.Chat, audioPath string, user *models.User) {
    if audioPath == "" || !fileExists(audioPath) {
        h.logger.Debug("No audio file to send or file doesn't exist")
        return
    }

    // Create file name based on user's language
    var fileName string
    if user == nil || user.InterfaceLanguage == "en" {
        fileName = "Audio Track.mp3"
    } else if user.InterfaceLanguage == "ar" {
        fileName = "المقطع الصوتي.mp3"
    } else if user.InterfaceLanguage == "de" {
        fileName = "Audiospur.mp3"
    } else if user.InterfaceLanguage == "fr" {
        fileName = "Piste Audio.mp3"
    }

    audio := &telebot.Audio{
        File:     telebot.FromDisk(audioPath),
        FileName: fileName,
    }
    
    _, err := h.bot.Send(chat, audio)
    if err != nil {
        h.logger.Error("Error sending audio file: %v", err)
    }
}

// sendSubtitleFile sends the downloaded subtitle file to the user with a descriptive name
func (h *BotHandler) sendSubtitleFile(chat *telebot.Chat, subtitlePath string, user *models.User) {
    if subtitlePath == "" || !fileExists(subtitlePath) {
        h.logger.Debug("No subtitle file to send or file doesn't exist")
        return
    }

    // Get file extension
    ext := filepath.Ext(subtitlePath)
    if ext == "" {
        ext = ".srt" // default to .srt if no extension found
    }

    // Create file name based on user's language
    var fileName string
    if user == nil || user.InterfaceLanguage == "en" {
        fileName = "Subtitles" + ext
    } else if user.InterfaceLanguage == "ar" {
        fileName = "الترجمة" + ext
    } else if user.InterfaceLanguage == "de" {
        fileName = "Untertitel" + ext
    } else if user.InterfaceLanguage == "fr" {
        fileName = "Sous-titres" + ext
    }

    doc := &telebot.Document{
        File:     telebot.FromDisk(subtitlePath),
        FileName: fileName,
    }
    
    _, err := h.bot.Send(chat, doc)
    if err != nil {
        h.logger.Error("Error sending subtitle file: %v", err)
    }
}


// sendPrimaryVideo sends the main video file to the user
func (h *BotHandler) sendPrimaryVideo(chat *telebot.Chat, videoPath string, user *models.User) {
    if videoPath == "" || !fileExists(videoPath) {
        h.logger.Debug("No primary video to send or file doesn't exist")
        return
    }

    // Create file name based on user's language
    var fileName string
    if user == nil || user.InterfaceLanguage == "en" {
        fileName = "Video.mp4"
    } else if user.InterfaceLanguage == "ar" {
        fileName = "الفيديو.mp4"
    } else if user.InterfaceLanguage == "de" {
        fileName = "Video.mp4"
    } else if user.InterfaceLanguage == "fr" {
        fileName = "Vidéo.mp4"
    }

    video := &telebot.Video{
        File:     telebot.FromDisk(videoPath),
        FileName: fileName,
    }
    
    _, err := h.bot.Send(chat, video)
    if err != nil {
        h.logger.Error("Error sending primary video: %v", err)
    }
}

// sendVideoWithSubtitles sends the video with embedded subtitles to the user
func (h *BotHandler) sendVideoWithSubtitles(chat *telebot.Chat, videoPath string, user *models.User) {
    if videoPath == "" || !fileExists(videoPath) {
        h.logger.Debug("No subtitled video to send or file doesn't exist")
        return
    }

    // Create caption and file name based on user's language
    var captionText, fileName string
    if user == nil || user.InterfaceLanguage == "en" {
        captionText = "Video with embedded subtitles"
        fileName = "Video (With Subtitles).mp4"
    } else if user.InterfaceLanguage == "ar" {
        captionText = "فيديو مع ترجمة مدمجة"
        fileName = "الفيديو (مع ترجمة).mp4"
    } else if user.InterfaceLanguage == "de" {
        captionText = "Video mit eingebetteten Untertiteln"
        fileName = "Video (mit Untertiteln).mp4"
    } else if user.InterfaceLanguage == "fr" {
        captionText = "Vidéo avec sous-titres intégrés"
        fileName = "Vidéo (avec sous-titres).mp4"
    }

    video := &telebot.Video{
        File:     telebot.FromDisk(videoPath),
        Caption:  captionText,
        FileName: fileName,
    }
    
    _, err := h.bot.Send(chat, video)
    if err != nil {
        h.logger.Error("Error sending video with subtitles: %v", err)
    }
}

// processDownload handles the video download process
func (h *BotHandler) processDownload(requestID interface{}, chatID int64, url string, captionLang string, statusMsg *telebot.Message) {
	ctx := context.Background()
	
	// Update request status to processing
	h.downloadRepo.UpdateDownloadRequestStatus(ctx, requestID.(primitive.ObjectID), "processing")
	
	// Download video
	result, err := h.downloader.Download(ctx, url, captionLang)
	if err != nil {
		h.logger.Error("Error downloading video: %v", err)
		
		// Update request status to failed
		h.downloadRepo.UpdateDownloadRequestStatus(ctx, requestID.(primitive.ObjectID), "failed")
		
		// Get user language preference
		user, _ := h.userRepo.FindUserByChatID(ctx, chatID)
		
		var errorMsg string
		if user == nil || user.InterfaceLanguage == "en" {
			errorMsg = "Failed to download video. Please try again later."
		} else if user.InterfaceLanguage == "ar" {
			errorMsg = "فشل تنزيل الفيديو. الرجاء المحاولة مرة أخرى لاحقًا."
		} else if user.InterfaceLanguage == "de" {
			errorMsg = "Video konnte nicht heruntergeladen werden. Bitte versuchen Sie es später erneut."
		} else if user.InterfaceLanguage == "fr" {
			errorMsg = "Échec du téléchargement de la vidéo. Veuillez réessayer plus tard."
		}
		
		// Send error message
		h.bot.Edit(statusMsg, errorMsg)
		return
	}
	
	// Update request status to completed
	h.downloadRepo.UpdateDownloadRequestStatus(ctx, requestID.(primitive.ObjectID), "completed")
	
	// Create download result
	downloadResult := &models.DownloadResult{
		RequestID:       requestID.(primitive.ObjectID),
		ChatID:          chatID,
		VideoPath:       result.VideoPath,
		VideoWithSubPath: result.VideoWithSubPath,
		AudioPath:       result.AudioPath,
		SubtitlePath:    result.SubtitlePath,
		HasSubtitle:     result.HasSubtitle,
		CreatedAt:       time.Now(),
	}
	
	_, err = h.downloadRepo.CreateDownloadResult(ctx, downloadResult)
	if err != nil {
		h.logger.Error("Error creating download result: %v", err)
	}
	
	// Get user language preference
	user, _ := h.userRepo.FindUserByChatID(ctx, chatID)
	
	var completedMsg string
	if user == nil || user.InterfaceLanguage == "en" {
		completedMsg = "Download completed! Sending files..."
	} else if user.InterfaceLanguage == "ar" {
		completedMsg = "اكتمل التنزيل! جاري إرسال الملفات..."
	} else if user.InterfaceLanguage == "de" {
		completedMsg = "Download abgeschlossen! Dateien werden gesendet..."
	} else if user.InterfaceLanguage == "fr" {
		completedMsg = "Téléchargement terminé! Envoi des fichiers..."
	}
	
	// Update status message
	h.bot.Edit(statusMsg, completedMsg)
	
	// Send files to user
	chat := &telebot.Chat{ID: chatID}
	
	// Send thumbnail if available
   if result.ThumbnailPath != "" {
    h.sendThumbnail(chatID, result.ThumbnailPath, user)
    }

     // Send primary video if available
    h.sendPrimaryVideo(chat, result.VideoPath, user)

    // Send video with subtitles if available
     h.sendVideoWithSubtitles(chat, result.VideoWithSubPath, user)
	
    // Send audio file if available
      h.sendAudioFile(chat, result.AudioPath, user)

    // Send subtitle file if available
      h.sendSubtitleFile(chat, result.SubtitlePath, user)
	
	// Send completion message
	var doneMsg string
	if user == nil || user.InterfaceLanguage == "en" {
		doneMsg = "All files sent! Send another video link to download more."
	} else if user.InterfaceLanguage == "ar" {
		doneMsg = "تم إرسال جميع الملفات! أرسل رابط فيديو آخر للتنزيل مرة أخرى."
	} else if user.InterfaceLanguage == "de" {
		doneMsg = "Alle Dateien gesendet! Senden Sie einen weiteren Video-Link, um mehr herunterzuladen."
	} else if user.InterfaceLanguage == "fr" {
		doneMsg = "Tous les fichiers envoyés! Envoyez un autre lien vidéo pour télécharger plus."
	}
	
	h.bot.Send(chat, doneMsg)
	
	// Schedule cleanup of download files (after 1 hour)
	go func() {
		time.Sleep(1 * time.Hour)
		
		// Clean up download directory
		if result.VideoPath != "" {
			os.Remove(result.VideoPath)
		}
		if result.VideoWithSubPath != "" {
			os.Remove(result.VideoWithSubPath)
		}
		if result.AudioPath != "" {
			os.Remove(result.AudioPath)
		}
		if result.SubtitlePath != "" {
			os.Remove(result.SubtitlePath)
		}
		
		// Remove parent directory
		if result.VideoPath != "" {
			os.RemoveAll(filepath.Dir(result.VideoPath))
		}
	}()
}

// isValidURL checks if a string is a valid URL
func isValidURL(text string) bool {
	// This is a simple check, you might want to use a more robust URL validation
	return len(text) > 8 && (text[:7] == "http://" || text[:8] == "https://")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
