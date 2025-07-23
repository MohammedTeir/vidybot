package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/joho/godotenv"
    "github.com/mohammedteir/telegram-video-downloader-bot/internal/config"
    "github.com/mohammedteir/telegram-video-downloader-bot/internal/database"
    "github.com/mohammedteir/telegram-video-downloader-bot/internal/downloader"
    "github.com/mohammedteir/telegram-video-downloader-bot/internal/handlers"
    "github.com/mohammedteir/telegram-video-downloader-bot/internal/utils"

    "gopkg.in/telebot.v3"
)

func main() {
    // Load .env file if it exists
    _ = godotenv.Load()

    // Load configuration
    cfg, err := config.LoadConfig()
    if err != nil {
        fmt.Printf("Error loading configuration: %v\n", err)
        os.Exit(1)
    }

    // Initialize logger
    logger, err := utils.NewLogger(cfg.Log.Enabled, cfg.Log.Path)
    if err != nil {
        fmt.Printf("Error initializing logger: %v\n", err)
        os.Exit(1)
    }
    defer logger.Close()

    // Initialize enhanced logger for components that require it
    enhancedLoggerConfig := &utils.EnhancedLoggerConfig{
    Enabled:      true,
    Level:        utils.LogLevelInfo,
    Path:         cfg.Log.Path,
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
        logger.Error("Failed to create enhanced logger: %v", err)
        fmt.Printf("Failed to create enhanced logger: %v\n", err)
        fmt.Fprintf(os.Stderr, "[ERROR] Failed to create enhanced logger: %v\n", err)
        os.Exit(1)
    }

    logger.Info("Starting Telegram Video Downloader Bot")
    
    
 // ✅ Step: Check and install external dependencies (yt-dlp, aria2c, ffmpeg)
depChecker := utils.NewDependencyChecker()

// Check dependencies and get their paths
results, err := depChecker.CheckDependencies()
if err != nil {
    logger.Error("🔍 Dependency check failed: %v", err)
    logger.Info("📦 Attempting to install missing dependencies...")

    // Try to install missing dependencies
    if installErr := depChecker.InstallDependencies(); installErr != nil {
        logger.Error("❌ Failed to install dependencies: %v", installErr)
        fmt.Printf("❌ Failed to install dependencies: %v\n", installErr)
        os.Exit(1)
    }

    // Re-check after installation to update dependencyPaths
    results, err = depChecker.CheckDependencies()
    if err != nil {
        logger.Error("❌ Dependency check failed after installation: %v", err)
        fmt.Printf("❌ Dependency check failed after installation: %v\n", err)
        os.Exit(1)
    }
}

// ✅ Log final status of each dependency
for dep, installed := range results {
    if installed {
        logger.Info("✅ Dependency '%s' is installed", dep)
    } else {
        logger.Warn("⚠️ Dependency '%s' is still missing after attempted installation", dep)
        fmt.Printf("⚠️ Dependency '%s' is still missing!\n", dep)
    }
}
  
    // Ensure download directory exists
    if err := os.MkdirAll(cfg.Download.TempDir, 0755); err != nil {
        logger.Error("Failed to create download directory: %v", err)
        fmt.Printf("Failed to create download directory: %v\n", err)
        os.Exit(1)
    }

    // Initialize MongoDB connection
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

mongoClient, err := database.NewMongoClient(ctx, cfg.MongoDB.URI)
if err != nil {
    logger.Error("Failed to connect to MongoDB: %v", err)
    fmt.Printf("Failed to connect to MongoDB: %v\n", err)
    os.Exit(1)
}

// Graceful MongoDB disconnect with new context
defer func() {
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer shutdownCancel()
    if err := mongoClient.Disconnect(shutdownCtx); err != nil {
        logger.Error("Error disconnecting MongoDB: %v", err)
    }
}()

    // Initialize Redis connection
    redisClient, err := database.NewRedisClient(ctx, cfg.Redis.URI)
    if err != nil {
        logger.Error("Failed to connect to Redis: %v", err)
        fmt.Printf("Failed to connect to Redis: %v\n", err)
        os.Exit(1)
    }
    defer redisClient.Close()

    // Initialize repositories
    userRepo := database.NewUserRepository(mongoClient, cfg.MongoDB.Database, enhancedLogger)
    
    // Initialize downloader, passing the dependency paths
    videoDownloader := downloader.NewVideoDownloader(cfg.Download.TempDir, enhancedLogger, 3, depChecker.GetDependencyPaths()) // Use getter method here

    // Initialize Telegram bot
    bot, err := telebot.NewBot(telebot.Settings{
        Token:  cfg.Telegram.Token,
        Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
    })
    if err != nil {
        logger.Error("Failed to create Telegram bot: %v", err)
        fmt.Printf("Failed to create Telegram bot: %v\n", err)
        os.Exit(1)
    }

    // Initialize handlers
    // NEW: Pass depChecker.GetDependencyPaths() to NewBotHandler
    handler := handlers.NewBotHandler(bot, userRepo, redisClient, cfg, logger, depChecker.GetDependencyPaths())
    handler.RegisterHandlers()

    // Start the bot
    logger.Info("Bot started successfully")
    fmt.Println("Bot started successfully")

    // Start the bot in a separate goroutine
    go bot.Start()

    // Start cleanup goroutine
    go func() {
        for {
            // Clean up old downloads every hour
            time.Sleep(1 * time.Hour)
            if err := videoDownloader.CleanupDownloads(24 * time.Hour); err != nil {
                logger.Error("Failed to clean up old downloads: %v", err)
            }
        }
    }()

    // Wait for termination signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    logger.Info("Shutting down bot...")
    fmt.Println("Shutting down bot...")
    defer bot.Stop()
}
