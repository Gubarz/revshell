package cmd

import "fmt"

func (m *tuiModel) selectAtCursor() {
	if m.cursor >= 0 && m.cursor < len(m.visible) {
		m.selected = m.visible[m.cursor]
		m.previewScroll = 0
	}
}

func (m *tuiModel) clampCursor() {
	m.cursor = max(0, min(m.cursor, len(m.visible)-1))
	m.scrollOffset = 0
	m.selectAtCursor()
	m.adjustScroll()
}

func (m *tuiModel) adjustScroll() {
	listH := m.listHeight()

	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}

	for {
		lastVisible := m.lastVisibleIndex(m.scrollOffset, listH)
		if m.cursor <= lastVisible {
			break
		}
		m.scrollOffset++
		if m.scrollOffset >= len(m.visible) {
			break
		}
	}
}

// lastVisibleIndex returns the last visible[] index that fits on screen
// when rendering starts at startIdx with rowBudget rows available.
func (m tuiModel) lastVisibleIndex(startIdx, rowBudget int) int {
	rows := 0
	lastType := m.precedingType(startIdx)
	last := startIdx

	for i := startIdx; i < len(m.visible); i++ {
		entry := m.visible[i]
		needed := 1
		if entry.typ != lastType {
			needed++ // type header takes a row
		}
		if rows+needed > rowBudget {
			break
		}
		lastType = entry.typ
		rows += needed
		last = i
	}
	return last
}

func (m tuiModel) listHeight() int {
	h := m.panelHeight() - 4 // matches contentH in renderList: (height-3) - 1
	return max(3, h)
}

// rowToVisibleIndex maps a rendered row to the corresponding visible[] index.
func (m tuiModel) rowToVisibleIndex(row int) int {
	lastType := m.precedingType(m.scrollOffset)
	rendered := 0

	for i := m.scrollOffset; i < len(m.visible); i++ {
		entry := m.visible[i]
		if entry.typ != lastType {
			lastType = entry.typ
			if rendered == row {
				return -1 // clicked on a type header
			}
			rendered++
		}
		if rendered == row {
			return i
		}
		rendered++
	}
	return -1
}

// --- Focus management ---

func (m *tuiModel) blurAll() {
	m.ipInput.Blur()
	m.portInput.Blur()
	m.shellInput.Blur()
	m.searchInput.Blur()
}

func (m *tuiModel) focusCurrent() {
	switch m.focus {
	case focusIP:
		m.ipInput.Focus()
	case focusPort:
		m.portInput.Focus()
	case focusShell:
		if m.isCustomShell() {
			m.shellInput.Focus()
		}
	case focusSearch:
		m.searchInput.Focus()
	}
}

func (m *tuiModel) setFocus(next focus) {
	m.blurAll()
	m.focus = next
	m.focusCurrent()
}

// cycleFocus returns the focus that is delta positions away in the focus ring.
func (m tuiModel) cycleFocus(delta int) focus {
	for i, f := range focusOrder {
		if f == m.focus {
			return focusOrder[(i+delta+len(focusOrder))%len(focusOrder)]
		}
	}
	return focusList
}

// --- Preview ---

func (m tuiModel) getPreview() string {
	if m.selected.typ == "" || m.selected.method == "" {
		return "Select a shell type and method..."
	}

	params := CommandParams{
		Name:      m.selected.typ,
		Method:    m.selected.method,
		IPAddress: m.ipInput.Value(),
		Port:      m.portInput.Value(),
		Shell:     m.shellValue(),
		Encoding:  m.encodings[m.encodingIdx],
	}
	cmd := getCommand(params)
	if cmd == "" {
		return "No command template found"
	}
	return setEncoding(params.Encoding, cmd)
}

func (m tuiModel) precedingType(start int) string {
	if start <= 0 || start >= len(m.visible) {
		return ""
	}
	currentType := m.visible[start].typ
	if m.visible[start-1].typ == currentType {
		return currentType
	}
	return ""
}

// --- Layout calculations ---

func (m tuiModel) panelWidths() (int, int) {
	leftWidth := max(20, m.width*2/5-2)
	rightWidth := m.width - leftWidth - 2

	if rightWidth < 20 {
		rightWidth = 20
		leftWidth = max(20, m.width-rightWidth-2)
		rightWidth = m.width - leftWidth - 2
	}
	return leftWidth, max(1, rightWidth)
}

func (m tuiModel) panelHeight() int {
	return max(5, m.height-3)
}

// previewDimensions returns the box and content dimensions for the preview panel.
// Box dimensions are for lipgloss .Width()/.Height() (includes padding, not border).
// Content dimensions are the actual text area inside padding.
func (m tuiModel) previewDimensions(rightWidth, panelHeight int) (boxW, boxH, contentW, contentH int) {
	const borderH, borderV = 2, 2
	const padH, padV = 4, 2

	boxW = max(14, rightWidth-borderH)
	contentW = max(10, boxW-padH)
	boxH = max(5, panelHeight-borderV)
	contentH = max(3, boxH-padV)
	return
}

func (m tuiModel) scrollIndicator() string {
	if len(m.visible) == 0 {
		return ""
	}
	pos := (m.cursor * 100) / len(m.visible)
	return dimStyle.Render(fmt.Sprintf("  %d/%d (%d%%)", m.cursor+1, len(m.visible), pos))
}
