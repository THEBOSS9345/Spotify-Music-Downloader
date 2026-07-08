package cli

import (
	"fmt"

	"spotscoop/src/infra/config"
)

func Startup(cfg *config.Config, addr string) {

	content := fmt.Sprintf(
		"%s\n\n"+
			"%s\n"+
			"%s %s\n"+
			"%s %s\n"+
			"%s %d\n"+
			"%s %t\n\n"+
			"%s\n"+
			"Listening for requests...\n\n"+
			"%s",

		Title.Render("🎵 Spotify Music Downloader\nby THEBOSS9345"),

		Header.Render("Server"),

		Label.Render("Address:"),
		Value.Render("http://"+addr),

		Label.Render("Downloads:"),
		Value.Render(cfg.OutputDir),

		Label.Render("Threads:"),
		cfg.MaxDownloadThreads,

		Label.Render("Debug:"),
		cfg.Debug,

		Header.Render("Status"),

		Footer.Render("Press Ctrl+C to stop."),
	)

	fmt.Println(Panel.Render(content))
}

func ConfigError(err error, addr string) {

	content := fmt.Sprintf(
		"%s\n\n"+
			"%s\n\n"+
			"%s\n\n"+
			"%s\n"+
			" 1. Go to https://developer.spotify.com/dashboard\n"+
			" 2. Create a Web Application\n"+
			" 3. Add Redirect URI:\n\n"+
			"    %s\n\n"+
			" 4. Copy your Client ID and Client Secret\n"+
			"    into config.json",

		Title.Render("🎵 Spotify Music Downloader\nby THEBOSS9345"),

		Error.Render("Configuration Error"),

		err.Error(),

		Header.Render("Setup"),

		Value.Render("http://"+addr+"/api/auth"),
	)

	fmt.Println(Panel.Render(content))
}
