package tui

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"music-downloader/src/infra/config"
	"music-downloader/src/infra/logs"
)

type tuiState int

const (
	stateLoading tuiState = iota
	stateConfigError
	stateSetupError
	stateRunning
)

type logFilter int

const (
	filterAll logFilter = iota
	filterError
	filterWarn
	filterInfo
)

func (f logFilter) String() string {
	switch f {
	case filterAll:
		return "All"
	case filterError:
		return "Error"
	case filterWarn:
		return "Warn+"
	case filterInfo:
		return "Info+"
	default:
		return "All"
	}
}

func (f logFilter) matches(level string) bool {
	switch f {
	case filterAll:
		return true
	case filterError:
		return level == "Error"
	case filterWarn:
		return level == "Error" || level == "Warn"
	case filterInfo:
		return level == "Error" || level == "Warn" || level == "Info" || level == "Succ"
	default:
		return true
	}
}

const maxLogLines = 2000
const maxDebugLines = 500

type logMsg logs.Entry

type tickMsg time.Time

type configLoadedMsg struct {
	cfg *config.Config
	err error
}

type serverReadyMsg struct {
	srv       *http.Server
	onQuit    func()
	err       error
	outputDir string
}

type downloadStateMsg DownloadState

type model struct {
	state     tuiState
	configErr error
	setupErr  error

	ready      bool
	mainVP     viewport.Model
	debugVP    viewport.Model
	mainLines  []string
	mainLevels []string
	debugLines []string
	showDebug  bool
	server     *http.Server
	onQuit     func()
	addr       string
	setupFn    func(*config.Config) (*http.Server, func(), error)
	width      int
	height     int
	logCount   int
	debugCount int
	startTime  time.Time
	logFile    *os.File

	goroutines int
	heapMB     float64
	sysMB      float64
	numCPU     int
	maxProcs   int

	autoScroll bool
	logFilter  logFilter
	debugSize  int

	downloads        []TrackState
	downloadActive   int
	downloadQueued   int
	downloadComplete int
	downloadFailed   int
	prevBytes        map[string]int64
	prevTime         map[string]time.Time

	outputDir      string
	folderSize     int64
	lastFolderCalc time.Time
	diskTotal      uint64
	diskFree       uint64

	stateCh chan DownloadState
}

var (
	subtle = lipgloss.Color("#5C6370")
	green  = lipgloss.Color("#98C379")
	red    = lipgloss.Color("#E06C75")
	yellow = lipgloss.Color("#E5C07B")
	purple = lipgloss.Color("#C678DD")
	blue   = lipgloss.Color("#61AFEF")
	orange = lipgloss.Color("#D19A66")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple)

	authorStyle = lipgloss.NewStyle().
			Foreground(subtle)

	headerBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(subtle).
			Padding(0, 1)

	statusDot = lipgloss.NewStyle().
			Bold(true).
			Foreground(green).
			Render("● Running")

	addrStyle = lipgloss.NewStyle().
			Foreground(green)

	sectionLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(blue)

	dividerLine = lipgloss.NewStyle().
			Foreground(subtle)

	callerStyle = lipgloss.NewStyle().
			Foreground(subtle)

	logErrorStyle = lipgloss.NewStyle().
			Foreground(red)

	logInfoStyle = lipgloss.NewStyle().
			Foreground(green)

	logSuccessStyle = lipgloss.NewStyle().
			Foreground(green)

	logWarningStyle = lipgloss.NewStyle().
			Foreground(yellow)

	logDebugStyle = lipgloss.NewStyle().
			Foreground(subtle)

	footerStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderForeground(subtle).
			Padding(0, 1)

	keyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple)

	emptyStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).
			Padding(1, 2).
			Width(72)

	errTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(red).
			Align(lipgloss.Center)

	errDetailStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA"))

	stepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(blue)

	stepValueStyle = lipgloss.NewStyle().
			Foreground(green)

	loadingStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true)

	progressBarFull = lipgloss.NewStyle().
			Foreground(green).
			Render("●")

	progressBarEmpty = lipgloss.NewStyle().
				Foreground(subtle).
				Render("○")

	progressLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(green)

	statusStyle = lipgloss.NewStyle().
			Foreground(yellow)

	dlActiveStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	dlQueuedStyle = lipgloss.NewStyle().
			Foreground(yellow)

	dlCompleteStyle = lipgloss.NewStyle().
			Foreground(subtle)

	dlFailedStyle = lipgloss.NewStyle().
			Foreground(red)

	filterTagStyle = lipgloss.NewStyle().
			Background(yellow).
			Foreground(lipgloss.Color("#000")).
			Padding(0, 1).
			Bold(true)

	scrollTagStyle = lipgloss.NewStyle().
			Background(green).
			Foreground(lipgloss.Color("#000")).
			Padding(0, 1).
			Bold(true)

	scrollOffTagStyle = lipgloss.NewStyle().
				Background(subtle).
				Foreground(lipgloss.Color("#000")).
				Padding(0, 1).
				Bold(true)
)

func Run(addr string, setupFn func(*config.Config) (*http.Server, func(), error)) {
	logCh := make(chan logs.Entry, 500)
	logs.SetLogChannel(logCh)

	stateCh := make(chan DownloadState, 100)
	SetDownloadStateChannel(stateCh)

	logFile, _ := os.CreateTemp("", "spotify-dl-*.log")

	p := tea.NewProgram(
		model{
			addr:       addr,
			state:      stateLoading,
			setupFn:    setupFn,
			showDebug:  true,
			startTime:  time.Now(),
			autoScroll: true,
			logFilter:  filterAll,
			debugSize:  12,
			prevBytes:  make(map[string]int64),
			prevTime:   make(map[string]time.Time),
			stateCh:    stateCh,
			logFile:    logFile,
		},
		tea.WithAltScreen(),
	)

	logs.SetTUIMode(true)

	go func() {
		for entry := range logCh {
			p.Send(logMsg(entry))
		}
	}()

	go func() {
		for state := range stateCh {
			p.Send(downloadStateMsg(state))
		}
	}()

	final, _ := p.Run()

	logs.SetTUIMode(false)
	logs.SetLogChannel(nil)
	close(logCh)
	SetDownloadStateChannel(nil)
	close(stateCh)
	if logFile != nil {
		logFile.Close()
		os.Remove(logFile.Name())
	}

	if m, ok := final.(model); ok {
		if m.onQuit != nil {
			m.onQuit()
		}
		if m.server != nil {
			m.server.Close()
		}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.loadConfig, m.tick())
}

func (m model) tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) loadConfig() tea.Msg {
	cfg, err := config.Init()
	return configLoadedMsg{cfg: cfg, err: err}
}

func (m model) setupServer(cfg *config.Config) tea.Msg {
	logs.SetDebug(cfg.Debug)
	logs.Info("Config loaded, starting server...")

	srv, onQuit, err := m.setupFn(cfg)
	if err != nil {
		return serverReadyMsg{err: err}
	}
	if srv == nil {
		return serverReadyMsg{err: fmt.Errorf("server setup returned nil")}
	}

	return serverReadyMsg{srv: srv, onQuit: onQuit, outputDir: cfg.OutputDir}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case configLoadedMsg:
		if msg.err != nil {
			m.state = stateConfigError
			m.configErr = msg.err
			return m, nil
		}
		return m, func() tea.Msg {
			return m.setupServer(msg.cfg)
		}

	case serverReadyMsg:
		if msg.err != nil {
			m.state = stateSetupError
			m.setupErr = msg.err
			return m, nil
		}
		m.state = stateRunning
		m.server = msg.srv
		m.onQuit = msg.onQuit
		m.outputDir = msg.outputDir

		if m.ready {
			m.mainVP = viewport.New(m.width, m.mainViewHeight())
			m.mainVP.YPosition = headerHeight() + 1
			m.mainVP.KeyMap = viewport.DefaultKeyMap()
			m.debugVP = viewport.New(m.width, m.debugViewHeight())
			if len(m.mainLines) > 0 {
				m.mainVP.SetContent(renderLines(m.mainLines))
				m.mainVP.GotoBottom()
			}
			if len(m.debugLines) > 0 {
				m.debugVP.SetContent(renderLines(m.debugLines))
				m.debugVP.GotoBottom()
			}
		}

		go func() {
			if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logs.Error("Server error: %v", err)
			}
		}()

		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		if m.state == stateRunning {
			m.mainVP = viewport.New(m.width, m.mainViewHeight())
			m.mainVP.YPosition = headerHeight() + 1
			m.mainVP.KeyMap = viewport.DefaultKeyMap()

			m.debugVP = viewport.New(m.width, m.debugViewHeight())

			if len(m.mainLines) > 0 {
				m.mainVP.SetContent(renderLines(m.mainLines))
				m.mainVP.GotoBottom()
			}
			if len(m.debugLines) > 0 {
				m.debugVP.SetContent(renderLines(m.debugLines))
				m.debugVP.GotoBottom()
			}
		}

	case tickMsg:
		m.collectStats()
		return m, m.tick()

	case logMsg:
		m.writeLog(logs.Entry(msg))

		if m.state != stateRunning && msg.Level != "Debug" {
			m.mainLines = append(m.mainLines, formatEntry(logs.Entry(msg)))
			m.mainLevels = append(m.mainLevels, msg.Level)
			m.logCount++
		}

		if m.state == stateRunning {
			line := formatEntry(logs.Entry(msg))
			if msg.Level == "Debug" {
				m.debugLines = append(m.debugLines, line)
				m.debugCount++
				if len(m.debugLines) > maxDebugLines {
					m.debugLines = m.debugLines[len(m.debugLines)-maxDebugLines:]
				}
				if m.ready {
					m.debugVP.SetContent(renderLines(m.debugLines))
					if m.autoScroll {
						m.debugVP.GotoBottom()
					}
				}
			} else {
				m.mainLines = append(m.mainLines, line)
				m.mainLevels = append(m.mainLevels, msg.Level)
				m.logCount++
				if len(m.mainLines) > maxLogLines {
					excess := len(m.mainLines) - maxLogLines
					m.mainLines = m.mainLines[excess:]
					m.mainLevels = m.mainLevels[excess:]
				}
				if m.ready {
					m.mainVP.SetContent(m.filteredLogContent())
					if m.autoScroll {
						m.mainVP.GotoBottom()
					}
				}
			}
		}

	case downloadStateMsg:
		s := DownloadState(msg)
		m.downloads = s.Tracks
		m.downloadActive = s.Active
		m.downloadQueued = s.Queued
		m.downloadComplete = s.Complete
		m.downloadFailed = s.Failed

		for _, t := range s.Tracks {
			if t.DownloadedBytes > 0 {
				if prev, ok := m.prevBytes[t.ID]; ok && prev > 0 {
					delta := t.DownloadedBytes - prev
					if delta > 0 {
						m.prevTime[t.ID] = time.Now()
					}
				} else {
					m.prevTime[t.ID] = time.Now()
				}
				m.prevBytes[t.ID] = t.DownloadedBytes
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "d":
			m.showDebug = !m.showDebug
			if m.ready && m.state == stateRunning {
				m.mainVP.Height = m.mainViewHeight()
			}
		case "s":
			m.autoScroll = !m.autoScroll
		case "f":
			m.logFilter = (m.logFilter + 1) % 4
			if m.ready && m.state == stateRunning {
				m.mainVP.SetContent(m.filteredLogContent())
				if m.autoScroll {
					m.mainVP.GotoBottom()
				}
			}
		case "+", "=":
			if m.debugSize < 20 {
				m.debugSize++
				if m.ready && m.state == stateRunning && m.showDebug {
					m.mainVP.Height = m.mainViewHeight()
				}
			}
		case "-", "_":
			if m.debugSize > 3 {
				m.debugSize--
				if m.ready && m.state == stateRunning && m.showDebug {
					m.mainVP.Height = m.mainViewHeight()
				}
			}
		}
	}

	if m.ready && m.state == stateRunning {
		m.mainVP, cmd = m.mainVP.Update(msg)
		var cmd2 tea.Cmd
		m.debugVP, cmd2 = m.debugVP.Update(msg)
		cmd = tea.Batch(cmd, cmd2)
	}

	return m, cmd
}

func (m model) View() string {
	switch m.state {
	case stateLoading:
		return m.loadingView()
	case stateConfigError:
		return m.configErrorView()
	case stateSetupError:
		return m.setupErrorView()
	case stateRunning:
		return m.runningView()
	default:
		return "unknown state"
	}
}

func (m model) loadingView() string {
	if !m.ready {
		return "Initializing..."
	}
	content := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		loadingStyle.Render("🎵 Loading configuration..."),
	)
	return content
}

func (m model) configErrorView() string {
	if !m.ready {
		return "Loading..."
	}

	errMsg := m.configErr.Error()

	panelContent := fmt.Sprintf(
		"%s %s\n\n%s\n\n"+
			"%s\n  • Go to https://developer.spotify.com/dashboard\n"+
			"  • Create a Web Application\n"+
			"  • Add Redirect URI:\n\n    %s\n\n"+
			"  • Copy your Client ID and Client Secret\n    into config.json\n\n"+
			"%s",
		titleStyle.Render("🎵 Spotify Music Downloader"),
		authorStyle.Render("by THEBOSS9345"),
		errTitleStyle.Render("✗ "+errMsg),
		stepStyle.Render("Setup Steps"),
		stepValueStyle.Render("http://"+m.addr+"/api/auth"),
		footerStyle.Render("Press q or Ctrl+C to quit"),
	)

	panel := panelStyle.Render(panelContent)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		panel,
	)
}

func (m model) setupErrorView() string {
	if !m.ready {
		return "Loading..."
	}

	lines := strings.Join(m.mainLines, "\n")

	if lines == "" {
		lines = m.setupErr.Error()
	}

	content := fmt.Sprintf(
		"%s %s\n\n%s\n\n%s\n\n%s",
		titleStyle.Render("🎵 Spotify Music Downloader"),
		authorStyle.Render("by THEBOSS9345"),
		errTitleStyle.Render("✗ Setup Error"),
		lines,
		footerStyle.Render("Press q or Ctrl+C to quit"),
	)

	panel := panelStyle.Render(content)

	var b strings.Builder
	b.WriteString(lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		panel,
	))

	return b.String()
}

func (m model) runningView() string {
	if !m.ready {
		return "Starting..."
	}

	var b strings.Builder

	b.WriteString(m.headerView())
	b.WriteString("\n")

	b.WriteString(m.downloadsView())
	b.WriteString("\n")

	logLabel := fmt.Sprintf("Logs %s SCROLL:%s", m.filterTag(), m.scrollTag())
	b.WriteString(m.sectionLabel(logLabel))
	b.WriteString("\n")
	b.WriteString(m.mainVP.View())
	b.WriteString("\n")

	if m.showDebug && len(m.debugLines) > 0 {
		b.WriteString(m.divider())
		b.WriteString("\n")
		b.WriteString(m.sectionLabel(fmt.Sprintf("Debug  %d", m.debugCount)))
		b.WriteString("\n")
		b.WriteString(m.debugVP.View())
		b.WriteString("\n")
	}

	b.WriteString(m.footerView())

	return b.String()
}

func (m model) filterTag() string {
	return filterTagStyle.Render(" " + m.logFilter.String() + " ")
}

func (m model) scrollTag() string {
	if m.autoScroll {
		return scrollTagStyle.Render(" ON ")
	}
	return scrollOffTagStyle.Render(" OFF")
}

func (m model) downloadsView() string {
	label := fmt.Sprintf("Downloads  %d active, %d queued", m.downloadActive, m.downloadQueued)
	var b strings.Builder
	b.WriteString(m.sectionLabel(label))
	b.WriteString("\n")

	shown := 0
	for _, t := range m.downloads {
		if shown >= 4 {
			break
		}
		if t.Status != "searching" && t.Status != "downloading" && t.Status != "converting" {
			continue
		}

		barWidth := m.width / 4
		if barWidth < 6 {
			barWidth = 6
		}
		if barWidth > 30 {
			barWidth = 30
		}

		bar := m.progressBar(t.Progress, barWidth)
		pctStr := fmt.Sprintf(" %d%%", t.Progress)
		speedStr := ""
		if prev, ok := m.prevBytes[t.ID]; ok && prev > 0 {
			speedStr = " " + formatSpeed(t.DownloadedBytes-prev)
		}

		title := t.Title
		artist := t.Artist

		for {
			bar = m.progressBar(t.Progress, barWidth)
			displayTitle := title
			displayArtist := artist
			if len(t.Title) > len(title) {
				displayTitle += "…"
			}
			if len(t.Artist) > len(artist) {
				displayArtist += "…"
			}
			content := fmt.Sprintf("  %s  %s - %s%s%s", bar, displayTitle, displayArtist, pctStr, speedStr)
			if lipgloss.Width(content) <= m.width {
				title = displayTitle
				artist = displayArtist
				break
			}
			if barWidth > 6 {
				barWidth--
			} else if len(artist) > 3 {
				artist = artist[:len(artist)-1]
			} else if len(title) > 3 {
				title = title[:len(title)-1]
			} else {
				break
			}
		}

		line := fmt.Sprintf("  %s  %s - %s%s%s",
			bar,
			dlActiveStyle.Render(title),
			statusStyle.Render(artist),
			progressLabel.Render(pctStr),
			speedStr,
		)

		b.WriteString(line)
		b.WriteString("\n")
		shown++
	}

	if shown == 0 {
		b.WriteString(emptyStyle.Render("  Idle — no active downloads"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m model) progressBar(pct int, maxWidth int) string {
	if maxWidth < 10 {
		maxWidth = 10
	}
	barWidth := maxWidth - 2
	filled := barWidth * pct / 100
	if filled > barWidth {
		filled = barWidth
	}
	var b strings.Builder
	b.WriteString("[")
	for i := 0; i < barWidth; i++ {
		if i < filled {
			b.WriteString("=")
		} else if i == filled {
			b.WriteString(">")
		} else {
			b.WriteString(" ")
		}
	}
	b.WriteString("]")
	return b.String()
}

func (m model) headerView() string {
	title := titleStyle.Render("🎵 Spotify Music Downloader")
	author := authorStyle.Render("by THEBOSS9345")
	status := statusDot
	addr := addrStyle.Render("http://" + m.addr)

	left := title + " " + author + "  " + status + "  " + addr
	w := lipgloss.Width(left)
	if w > m.width {
		left = left[:m.width]
	} else {
		left += strings.Repeat(" ", m.width-w)
	}

	return headerBorder.Render(left)
}

func (m model) sectionLabel(text string) string {
	w := m.width - lipgloss.Width(text) - 2
	if w < 3 {
		w = 3
	}
	rule := strings.Repeat("─", w-2)
	return " " + sectionLabel.Render(text) + " " + dividerLine.Render(rule)
}

func (m model) divider() string {
	return dividerLine.Render(strings.Repeat("─", m.width))
}

func (m model) footerView() string {
	uptime := formatDuration(time.Since(m.startTime))

	keys := fmt.Sprintf("%s  %s  %s  %s  %s  %s",
		keyStyle.Render("q")+" quit",
		keyStyle.Render("d")+" debug",
		keyStyle.Render("f")+" filter",
		keyStyle.Render("s")+" scroll",
		keyStyle.Render("+/-")+" resize",
		keyStyle.Render("↑↓")+" scroll",
	)

	var dlStat string
	if m.downloadComplete > 0 || m.downloadFailed > 0 {
		dlStat = fmt.Sprintf("saved: %d failed: %d ", m.downloadComplete, m.downloadFailed)
	} else {
		dlStat = "DL: —"
	}

	info := fmt.Sprintf("logs:%d  debug:%d  %s  uptime: %s",
		m.logCount, m.debugCount, dlStat, uptime)

	stats := fmt.Sprintf("goroutines:%d  mem:%.1fMB  %s  cores:%d",
		m.goroutines, m.heapMB, m.diskStr(), m.numCPU)
	statsStyle := lipgloss.NewStyle().Foreground(subtle)

	gap := m.width - lipgloss.Width(keys) - lipgloss.Width(info) - 4
	if gap < 1 {
		gap = 1
	}
	line1 := keys + strings.Repeat(" ", gap) + info

	pad := m.width - lipgloss.Width(stats) - 5
	if pad < 1 {
		pad = 1
	}
	line2 := strings.Repeat(" ", pad) + statsStyle.Render(stats)

	return footerStyle.Render(line1 + "\n" + line2)
}

func (m model) diskStr() string {
	dir := m.outputDir
	if dir == "" {
		dir = "downloads"
	}
	folderStr := formatBytes(m.folderSize)
	freeStr := formatBytes(int64(m.diskFree))
	return fmt.Sprintf("disk: %s/%s", folderStr, freeStr)
}

func formatBytes(b int64) string {
	switch {
	case b > 1024*1024*1024:
		return fmt.Sprintf("%.1fGB", float64(b)/(1024*1024*1024))
	case b > 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(b)/(1024*1024))
	case b > 1024:
		return fmt.Sprintf("%.0fKB", float64(b)/1024)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatSpeed(bytesPerSec int64) string {
	if bytesPerSec < 0 {
		return ""
	}
	switch {
	case bytesPerSec > 1e9:
		return fmt.Sprintf("%.1fGB/s", float64(bytesPerSec)/1e9)
	case bytesPerSec > 1e6:
		return fmt.Sprintf("%.1fMB/s", float64(bytesPerSec)/1e6)
	case bytesPerSec > 1e3:
		return fmt.Sprintf("%.0fKB/s", float64(bytesPerSec)/1e3)
	default:
		return fmt.Sprintf("%dB/s", bytesPerSec)
	}
}

func (m *model) collectStats() {
	m.goroutines = runtime.NumGoroutine()
	m.numCPU = runtime.NumCPU()
	m.maxProcs = runtime.GOMAXPROCS(0)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	m.heapMB = float64(mem.Alloc) / 1024 / 1024
	m.sysMB = float64(mem.Sys) / 1024 / 1024

	m.collectDiskUsage()
}

func (m *model) collectDiskUsage() {
	dir := m.outputDir
	if dir == "" {
		dir = "downloads"
	}

	if time.Since(m.lastFolderCalc) > 30*time.Second {
		m.lastFolderCalc = time.Now()
		m.folderSize = 0
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				m.folderSize += info.Size()
			}
			return nil
		})
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	root := filepath.VolumeName(abs) + "\\"

	total, free, ok := diskUsage(root)
	if ok {
		m.diskTotal = total
		m.diskFree = free
	}
}

func (m model) mainViewHeight() int {
	active := m.downloadActive
	if active > 4 {
		active = 4
	}
	downloadH := active + 2

	h := m.height - headerHeight() - footerHeight() - 4 - downloadH
	if m.showDebug && len(m.debugLines) > 0 {
		h -= m.debugViewHeight() + 3
	}
	if h < 3 {
		h = 3
	}
	return h
}

func (m model) debugViewHeight() int {
	h := m.debugSize
	if h < 3 {
		h = 3
	}
	if h > m.height/2 {
		h = m.height / 2
	}
	return h
}

func (m model) filteredLogContent() string {
	var filtered []string
	for i, line := range m.mainLines {
		level := "Info"
		if i < len(m.mainLevels) {
			level = m.mainLevels[i]
		}
		if m.logFilter.matches(level) {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) == 0 {
		return emptyStyle.Render("  No matching log entries")
	}
	return renderLines(filtered)
}

func (m *model) writeLog(e logs.Entry) {
	if m.logFile == nil {
		return
	}
	line := fmt.Sprintf("%s [%s] %s\n", e.Time.Format("1/2 3:04PM"), e.Caller, e.Message)
	m.logFile.WriteString(line)
}

func extractLevel(line string) string { return "" }

func headerHeight() int {
	return 2
}

func footerHeight() int {
	return 1
}

func formatEntry(e logs.Entry) string {
	var msgStyle lipgloss.Style
	switch e.Level {
	case "Error":
		msgStyle = logErrorStyle
	case "Info":
		msgStyle = logInfoStyle
	case "Succ":
		msgStyle = logSuccessStyle
	case "Warn":
		msgStyle = logWarningStyle
	case "Debug":
		msgStyle = logDebugStyle
	default:
		msgStyle = logInfoStyle
	}

	ts := e.Time.Format("1/2 3:04PM")
	tsStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("#3E4452")).Render(ts)

	if e.Level == "Debug" {
		return fmt.Sprintf("  %s %s %s", tsStyled, callerStyle.Render(e.Caller), logDebugStyle.Render(e.Message))
	}
	return fmt.Sprintf("  %s %s %s", tsStyled, callerStyle.Render(e.Caller), msgStyle.Render(e.Message))
}

func renderLines(lines []string) string {
	return strings.Join(lines, "\n")
}
