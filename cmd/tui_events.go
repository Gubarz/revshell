package cmd

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseMsg:
		return m.updateMouse(msg)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.aborted = true
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusIP, focusPort:
		return m.updateTextInput(msg)
	case focusShell:
		if m.isCustomShell() {
			return m.updateShellCustom(msg)
		}
		return m.updateCycler(msg)
	case focusEncoding, focusOS:
		return m.updateCycler(msg)
	case focusSearch:
		return m.updateSearch(msg)
	case focusList:
		return m.updateList(msg)
	}
	return m, nil
}

// --- Mouse ---

func (m tuiModel) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.moveCursor(m.cursor - 1)
		return m, nil
	case tea.MouseButtonWheelDown:
		m.moveCursor(m.cursor + 1)
		return m, nil
	}

	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		return m.handleLeftClick(msg)
	}
	return m, nil
}

func (m tuiModel) handleLeftClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	const (
		topBarY   = 0
		searchBar = 2
		listStart = 4
	)
	leftWidth, _ := m.panelWidths()

	if msg.Y == topBarY {
		m.focusTopBarAtX(msg.X)
		return m, nil
	}

	if msg.Y == searchBar && msg.X < leftWidth {
		m.setFocus(focusSearch)
		return m, textinput.Blink
	}

	if msg.Y >= listStart && msg.X < leftWidth {
		idx := m.rowToVisibleIndex(msg.Y - listStart)
		if idx >= 0 {
			m.cursor = idx
			m.selected = m.visible[idx]
			m.setFocus(focusList)
		}
	}
	return m, nil
}

func (m *tuiModel) focusTopBarAtX(x int) {
	switch {
	case x < 25:
		m.setFocus(focusIP)
	case x < 38:
		m.setFocus(focusPort)
	case x < 54:
		m.setFocus(focusShell)
	case x < 70:
		m.setFocus(focusEncoding)
	default:
		m.setFocus(focusOS)
	}
}

// --- Text inputs (IP, Port) ---

func (m tuiModel) updateTextInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.setFocus(m.cycleFocus(1))
		return m, nil
	case "shift+tab":
		m.setFocus(m.cycleFocus(-1))
		return m, nil
	case "enter", "esc":
		m.setFocus(focusList)
		return m, nil
	}

	var cmd tea.Cmd
	switch m.focus {
	case focusIP:
		m.ipInput, cmd = m.ipInput.Update(msg)
	case focusPort:
		m.portInput, cmd = m.portInput.Update(msg)
	}
	return m, cmd
}

// --- Custom shell text input ---

func (m tuiModel) updateShellCustom(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.setFocus(m.cycleFocus(1))
		return m, nil
	case "shift+tab":
		m.setFocus(m.cycleFocus(-1))
		return m, nil
	case "enter", "esc":
		m.setFocus(focusList)
		return m, nil
	case "left", "h":
		m.shellIdx = (m.shellIdx - 1 + len(m.shells)) % len(m.shells)
		m.blurAll()
		return m, nil
	}
	var cmd tea.Cmd
	m.shellInput, cmd = m.shellInput.Update(msg)
	m.customShell = m.shellInput.Value()
	return m, cmd
}

// --- Cycler (Shell presets, Encoding, OS) ---

func (m tuiModel) updateCycler(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.setFocus(m.cycleFocus(1))
		return m, nil
	case "shift+tab":
		m.setFocus(m.cycleFocus(-1))
		return m, nil
	case "left", "h":
		m.cycleCurrentIndex(-1)
		return m, nil
	case "right", "l":
		m.cycleCurrentIndex(1)
		return m, nil
	case "enter", "esc":
		m.setFocus(focusList)
		return m, nil
	}
	return m, nil
}

// cycleCurrentIndex adjusts the index for the currently focused cycler
// and runs any side effects (OS → rebuild list, Shell → toggle text input).
func (m *tuiModel) cycleCurrentIndex(delta int) {
	switch m.focus {
	case focusShell:
		m.shellIdx = (m.shellIdx + delta + len(m.shells)) % len(m.shells)
		m.blurAll()
		m.focusCurrent()
	case focusEncoding:
		m.encodingIdx = (m.encodingIdx + delta + len(m.encodings)) % len(m.encodings)
	case focusOS:
		m.osIdx = (m.osIdx + delta + len(m.osFilters)) % len(m.osFilters)
		m.rebuildVisible()
		m.clampCursor()
	}
}

// --- Search ---

func (m tuiModel) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchInput.SetValue("")
		m.setFocus(focusList)
		m.rebuildVisible()
		m.clampCursor()
		return m, nil
	case "enter", "down":
		m.setFocus(focusList)
		return m, nil
	case "tab":
		m.setFocus(m.cycleFocus(1))
		return m, nil
	case "shift+tab":
		m.setFocus(m.cycleFocus(-1))
		return m, nil
	}

	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.rebuildVisible()
	m.clampCursor()
	return m, cmd
}

// --- List navigation ---

func (m tuiModel) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.aborted = true
		return m, tea.Quit
	case "tab":
		m.setFocus(focusIP)
		return m, nil
	case "shift+tab":
		m.setFocus(focusOS)
		return m, nil
	case "/":
		m.setFocus(focusSearch)
		return m, textinput.Blink
	case "j", "down":
		m.moveCursor(m.cursor + 1)
		return m, nil
	case "k", "up":
		m.moveCursor(m.cursor - 1)
		return m, nil
	case "g":
		m.moveCursor(0)
		return m, nil
	case "G":
		m.moveCursor(len(m.visible) - 1)
		return m, nil
	case "ctrl+j", "ctrl+d":
		m.previewScroll++
		return m, nil
	case "ctrl+k", "ctrl+u":
		m.previewScroll = max(0, m.previewScroll-1)
		return m, nil
	case "enter":
		if m.cursor >= 0 && m.cursor < len(m.visible) {
			m.selected = m.visible[m.cursor]
			m.confirmed = true
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

// moveCursor clamps pos to [0, len(visible)-1], then updates selection and scroll.
func (m *tuiModel) moveCursor(pos int) {
	m.cursor = max(0, min(pos, len(m.visible)-1))
	m.selectAtCursor()
	m.adjustScroll()
}
