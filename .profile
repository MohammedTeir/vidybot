#!/bin/bash

# Check if pip3 is available
if command -v pip3 >/dev/null 2>&1; then
    echo "Installing yt-dlp..."
    pip3 install --upgrade yt-dlp
else
    echo "pip3 not found. Skipping yt-dlp install."
fi

# Set and log the download directory
export DOWNLOAD_TEMP_DIR="/tmp/video_downloader"
echo "Download directory: $DOWNLOAD_TEMP_DIR"

# Create the download directory (safe regardless)
mkdir -p "$DOWNLOAD_TEMP_DIR"

# Set execution permission on your Go binary
chmod +x bin/vidybot