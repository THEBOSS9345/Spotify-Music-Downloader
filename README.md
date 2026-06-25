# Spotify Music Downloader

Download your Spotify playlists as MP3s with metadata and album art embedded. Log in with Spotify, browse your library or search, and it downloads matching audio from YouTube.

Works on **Linux** and **Windows**.

## Quick Start

```bash
# 1. Clone and enter the project
git clone https://github.com/your-username/music-downloader
cd music-downloader

# 2. Build the frontend (required before Go build)
cd frontend
npm install
npm run build
cd ..

# 3. Build the Go binary
go build -o music-downloader

# 4. Run it
./music-downloader          # Linux/macOS
music-downloader.exe        # Windows
```

Open http://127.0.0.1:50811.

<details>
<summary><b>Setup: Spotify API credentials</b></summary>

1. Create a [Spotify app](https://developer.spotify.com/dashboard)
2. Add this Redirect URI:
   ```
   http://127.0.0.1:50811/api/auth
   ```
3. Run the app once to generate `config.json`, then fill in your credentials:

   ```json
   {
     "spotify": {
       "clientId": "YOUR_CLIENT_ID",
       "clientSecret": "YOUR_CLIENT_SECRET"
     },
     "outputDir": "downloads",
     "maxConcurrentDownloads": 3
   }
   ```
</details>

<details>
<summary><b>Prerequisites</b></summary>

- [Go](https://go.dev/dl/) 1.23+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) on PATH
- [ffmpeg](https://ffmpeg.org/) on PATH
</details>

<details>
<summary><b>Development (Frontend)</b></summary>

The frontend is a React app in `frontend/`. To rebuild after changes:

```bash
cd frontend
npm install   # first time only
npm run build
```

Then rebuild the Go binary to embed the updated frontend.
</details>

## Disclaimer

This project is for educational purposes only. If you are from Spotify's legal team and want me to take this project down, please contact me and I will remove it immediately. I am not responsible for any damages, legal issues, or lawsuits arising from the use of this software. Use at your own risk.

## License

MIT