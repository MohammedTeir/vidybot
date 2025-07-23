# Telegram Video Downloader Bot

A Telegram bot that downloads videos from various platforms and provides them in multiple formats with subtitle support.

## Features

- Downloads videos from YouTube, Twitter, Instagram, and other platforms
- Provides multiple file formats:
  - Best quality video (merged video + audio)
  - Subtitle-embedded video (if captions available)
  - Audio-only file
  - Subtitle file (if available)
- Multi-language support:
  - Arabic
  - English
  - German
  - French
- User preference storage in MongoDB
- Efficient downloading with yt-dlp and aria2c
- Subtitle embedding with FFmpeg

## Requirements

- Go 1.21+
- MongoDB
- Redis
- yt-dlp
- aria2c
- FFmpeg

## Installation

1. Clone the repository:
```bash
git clone https://github.com/mohammedteir/telegram-video-downloader-bot.git
cd telegram-video-downloader-bot
```

2. Install dependencies:
```bash
go mod tidy
```

3. Set up environment variables by creating a `.env` file or using the provided one:
```
TELEGRAM_TOKEN=your_telegram_bot_token
MONGODB_URI=your_mongodb_connection_string
MONGODB_DATABASE=video_downloader
REDIS_URI=your_redis_connection_string
DOWNLOAD_TEMP_DIR=/tmp/video_downloader
```

4. Run the dependency check script to ensure all external dependencies are installed:
```bash
go run scripts/check_dependencies.go
```

## Usage

1. Start the bot:
```bash
go run cmd/main.go
```

2. Open Telegram and search for your bot by username.

3. Start a conversation with the bot by sending the `/start` command.

4. Follow the instructions to set your preferred interface and caption languages.

5. Send a video URL to download it.

## Bot Commands

- `/start` - Start the bot and set up language preferences
- `/help` - Show help information
- `/about` - Show information about the bot
- `/lang` - Change language settings

## Testing

The repository includes several test scripts:

- `scripts/check_dependencies.go` - Check if all required dependencies are installed
- `scripts/test_database.go` - Test database connections and operations
- `scripts/test_downloader.go` - Test the video downloader functionality

To run a test script:
```bash
go run scripts/test_downloader.go https://www.youtube.com/watch?v=example
```

## Project Structure

```
telegram-video-downloader-bot/
├── cmd/
│   └── main.go                  # Main application entry point
├── config/
│   └── config.yaml              # Configuration file
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── database/
│   │   ├── clients.go           # MongoDB and Redis clients
│   │   └── repositories.go      # Data repositories
│   ├── downloader/
│   │   └── downloader.go        # Video download functionality
│   ├── handlers/
│   │   └── handlers.go          # Telegram bot handlers
│   ├── models/
│   │   └── models.go            # Data models
│   └── utils/
│       ├── command.go           # Command execution utilities
│       ├── dependency_checker.go # External dependency checking
│       └── logger.go            # Logging utilities
└── scripts/
    ├── check_dependencies.go    # Dependency checking script
    ├── test_database.go         # Database testing script
    └── test_downloader.go       # Downloader testing script
```

## License

MIT

## Author

Mohammed Teir
