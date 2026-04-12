package cmd

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focus int

const (
	focusIP focus = iota
	focusPort
	focusShell
	focusEncoding
	focusOS
	focusSearch
	focusList
)

var focusOrder = []focus{focusIP, focusPort, focusShell, focusEncoding, focusOS, focusSearch, focusList}

type tuiTheme struct {
	Title       string
	Muted       string
	Selected    string
	Cursor      string
	Label       string
	ActiveLabel string
	Border      string
	Active      string
	Type        string
}

// tuiThemeConfig is the single place to tweak TUI colors.
var tuiThemeConfig = tuiTheme{
	Title:       "81",
	Muted:       "245",
	Selected:    "117",
	Cursor:      "229",
	Label:       "153",
	ActiveLabel: "230",
	Border:      "245",
	Active:      "229",
	Type:        "189",
}

// shellEntry represents a single selectable shell in the flat list.
type shellEntry struct {
	typ    string
	method string
	meta   []string
}

type tuiModel struct {
	focus focus

	// top bar inputs
	ipInput     textinput.Model
	portInput   textinput.Model
	shellInput  textinput.Model
	shells      []string
	shellIdx    int
	customShell string // used when shellIdx points to "custom"
	encodingIdx int
	encodings   []string
	osIdx       int
	osFilters   []string

	// left panel: search + flat shell list
	searchInput textinput.Model
	allEntries  []shellEntry // every type/method combo
	visible     []shellEntry // filtered view
	cursor      int
	selected    shellEntry

	// terminal size
	width  int
	height int

	// scroll state
	scrollOffset  int
	previewScroll int

	// exit state
	confirmed bool
	aborted   bool
}

// --- Model construction ---

func newTUIModel() tuiModel {
	config := readConfigFromFile()

	ipIn := newTextInput("10.10.10.10", 45, 16)
	portIn := newTextInput("9001", 5, 6)
	shellIn := newTextInput("type custom shell", 20, 16)
	searchIn := newTextInput("search...", 30, 20)

	setInputFromConfig(ipIn, config.IPAddress, firstIP())
	setInputFromConfig(portIn, config.Port, DefaultPort)

	allEntries := buildEntries()
	shells, shellIdx, customShell := initShellPreset(config.Shell)
	if customShell != "" {
		shellIn.SetValue(customShell)
	}

	m := tuiModel{
		focus:       focusList,
		ipInput:     *ipIn,
		portInput:   *portIn,
		shellInput:  *shellIn,
		shells:      shells,
		shellIdx:    shellIdx,
		customShell: customShell,
		encodings:   list["encodings"],
		osFilters:   buildOSFilters(allEntries),
		searchInput: *searchIn,
		allEntries:  allEntries,
		width:       80,
		height:      24,
	}

	if len(allEntries) > 0 {
		m.selected = allEntries[0]
	}
	m.rebuildVisible()
	return m
}

// newTextInput creates a styled text input with no prompt.
func newTextInput(placeholder string, charLimit, width int) *textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.CharLimit = charLimit
	ti.Width = width
	return &ti
}

// setInputFromConfig sets a text input's value from config, falling back to a default.
func setInputFromConfig(input *textinput.Model, configVal, fallback string) {
	if configVal != "" {
		input.SetValue(configVal)
	} else if fallback != "" {
		input.SetValue(fallback)
	}
}

func firstIP() string {
	ips := getIP()
	if len(ips) > 0 {
		return ips[0]
	}
	return ""
}

func buildEntries() []shellEntry {
	entries := make([]shellEntry, 0, len(revShells))
	for _, cmd := range revShells {
		entries = append(entries, shellEntry{
			typ:    cmd.Name,
			method: cmd.Method,
			meta:   cmd.Meta,
		})
	}
	return entries
}

// initShellPreset builds the shells list and resolves the initial selection.
// Returns (shells, index, customValue). customValue is non-empty when the
// configured shell doesn't match any preset.
func initShellPreset(configShell string) ([]string, int, string) {
	presets := append([]string(nil), list["shells"]...)
	presets = append(presets, "custom")

	initShell := DefaultShell
	if configShell != "" {
		initShell = configShell
	}

	for i, s := range presets {
		if strings.EqualFold(s, initShell) {
			return presets, i, ""
		}
	}
	// No match — default to "custom" with the configured value.
	return presets, len(presets) - 1, initShell
}

// --- Filtering ---

func (m *tuiModel) rebuildVisible() {
	filter := strings.ToLower(m.searchInput.Value())
	osFilter := m.osFilters[m.osIdx]

	m.visible = nil
	for _, e := range m.allEntries {
		if osFilter != "all" && !containsStr(e.meta, osFilter) {
			continue
		}
		if filter != "" && !matchesFilter(e, filter) {
			continue
		}
		m.visible = append(m.visible, e)
	}
}

func matchesFilter(entry shellEntry, filter string) bool {
	return strings.Contains(strings.ToLower(entry.typ), filter) ||
		strings.Contains(strings.ToLower(entry.method), filter)
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}

// --- Shell value ---

// shellValue returns the effective shell: the preset name, or the custom text input value.
func (m tuiModel) shellValue() string {
	if m.isCustomShell() {
		v := strings.TrimSpace(m.shellInput.Value())
		if v == "" {
			return DefaultShell
		}
		return v
	}
	return m.shells[m.shellIdx]
}

// isCustomShell returns true when the user is on the "custom" shell entry.
func (m tuiModel) isCustomShell() bool {
	return m.shells[m.shellIdx] == "custom"
}

// --- OS filters ---

func buildOSFilters(entries []shellEntry) []string {
	seen := map[string]struct{}{}
	for _, entry := range entries {
		for _, meta := range entry.meta {
			if osName, ok := normalizeOSTag(meta); ok {
				seen[osName] = struct{}{}
			}
		}
	}

	filters := make([]string, 0, len(seen)+1)
	filters = append(filters, "all")

	if len(seen) == 0 {
		return filters
	}

	// Prioritized order, then alphabetical.
	for _, osName := range []string{"linux", "windows", "mac", "android", "apple_ios"} {
		if _, ok := seen[osName]; ok {
			filters = append(filters, osName)
			delete(seen, osName)
		}
	}

	remaining := make([]string, 0, len(seen))
	for osName := range seen {
		remaining = append(remaining, osName)
	}
	sort.Strings(remaining)
	return append(filters, remaining...)
}

func normalizeOSTag(meta string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(meta)) {
	case "linux":
		return "linux", true
	case "windows", "win":
		return "windows", true
	case "mac", "macos", "darwin", "osx":
		return "mac", true
	case "android":
		return "android", true
	case "apple_ios", "ios":
		return "apple_ios", true
	default:
		return "", false
	}
}

// --- Bubble Tea interface ---

func (m tuiModel) Init() tea.Cmd {
	return textinput.Blink
}

// --- Styles ---

var (
	titleStyle           lipgloss.Style
	dimStyle             lipgloss.Style
	selectedStyle        lipgloss.Style
	cursorStyle          lipgloss.Style
	labelStyle           lipgloss.Style
	activeLabelStyle     lipgloss.Style
	previewBorder        lipgloss.Style
	searchBoxStyle       lipgloss.Style
	searchBoxActiveStyle lipgloss.Style
	helpStyle            lipgloss.Style
	typeStyle            lipgloss.Style
)

func init() {
	applyTUITheme(tuiThemeConfig)
}

func applyTUITheme(theme tuiTheme) {
	titleStyle = lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color(theme.Title))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted))

	selectedStyle = lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color(theme.Selected))

	cursorStyle = lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color(theme.Cursor))

	labelStyle = lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color(theme.Label))

	activeLabelStyle = lipgloss.NewStyle().Bold(true).Underline(true).
		Foreground(lipgloss.Color(theme.ActiveLabel))

	previewBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Border)).
		Padding(1, 2)

	searchBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Border)).
		Padding(0, 1)

	searchBoxActiveStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Active)).
		Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Muted))

	typeStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Type))
}
