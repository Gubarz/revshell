package cmd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

func (m tuiModel) View() string {
	if m.width < 40 {
		return "Terminal too narrow"
	}

	var b strings.Builder
	b.WriteString(m.renderTopBar())
	b.WriteString("\n")

	leftWidth, rightWidth := m.panelWidths()
	panelHeight := m.panelHeight()

	leftPanel := lipgloss.NewStyle().
		Width(leftWidth).
		Height(panelHeight).
		Render(m.renderList(leftWidth, panelHeight))

	boxW, boxH, contentW, contentH := m.previewDimensions(rightWidth, panelHeight)
	rightPanel := previewBorder.
		Width(boxW).
		Height(boxH).
		Render(m.renderPreview(contentW, contentH))

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel))
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	// Hard-clamp to terminal height so the top bar never scrolls off.
	return clampLines(b.String(), m.height)
}

// --- Top bar ---

func (m tuiModel) renderTopBar() string {
	label := func(f focus, text string) string {
		if m.focus == f {
			return activeLabelStyle.Render(text)
		}
		return labelStyle.Render(text)
	}

	var shellDisplay string
	if m.isCustomShell() {
		shellDisplay = m.shellInput.View()
	} else {
		shellDisplay = m.renderCyclerValue(m.focus == focusShell, m.shells[m.shellIdx])
	}

	sep := dimStyle.Render(" | ")
	return strings.Join([]string{
		fmt.Sprintf(" %s %s", label(focusIP, "IP:"), m.ipInput.View()),
		fmt.Sprintf("%s %s", label(focusPort, "Port:"), m.portInput.View()),
		fmt.Sprintf("%s %s", label(focusShell, "Shell:"), shellDisplay),
		fmt.Sprintf("%s %s", label(focusEncoding, "Enc:"), m.renderCyclerValue(m.focus == focusEncoding, m.encodings[m.encodingIdx])),
		fmt.Sprintf("%s %s", label(focusOS, "OS:"), m.renderCyclerValue(m.focus == focusOS, m.osFilters[m.osIdx])),
	}, sep)
}

func (m tuiModel) renderCyclerValue(active bool, value string) string {
	value = truncate(value, 12)
	if active {
		return activeLabelStyle.Render(fmt.Sprintf("[ %s ]", value))
	}
	return selectedStyle.Render(value)
}

// --- Left panel: list ---

func (m tuiModel) renderList(width, height int) string {
	var b strings.Builder

	// Search box.
	searchContent := fmt.Sprintf("/ %s", m.searchInput.View())
	innerWidth := max(10, width-4)
	if m.focus == focusSearch {
		b.WriteString(searchBoxActiveStyle.Width(innerWidth).Render(searchContent))
	} else {
		b.WriteString(searchBoxStyle.Width(innerWidth).Render(searchContent))
	}
	b.WriteString("\n")

	// Content area (list entries).
	contentH := max(1, height-3-1) // -3 for search box chrome, -1 to reserve indicator line
	lastType := m.precedingType(m.scrollOffset)
	rendered := 0

	for i := m.scrollOffset; i < len(m.visible) && rendered < contentH; i++ {
		entry := m.visible[i]
		isCursor := i == m.cursor
		isSelected := entry.typ == m.selected.typ && entry.method == m.selected.method

		// Type header when entering a new group.
		if entry.typ != lastType {
			lastType = entry.typ
			if rendered >= contentH-1 {
				break
			}
			b.WriteString(m.renderTypeHeader(entry.typ, isCursor, width))
			b.WriteString("\n")
			rendered++
			if rendered >= contentH {
				break
			}
		}

		b.WriteString(m.renderMethodLine(entry.method, isCursor, isSelected, width))
		b.WriteString("\n")
		rendered++
	}

	// Pad to fixed height so help bar never jumps.
	for rendered < contentH {
		b.WriteString("\n")
		rendered++
	}
	b.WriteString(m.scrollIndicator())

	return b.String()
}

func (m tuiModel) renderTypeHeader(typ string, isCursor bool, width int) string {
	header := truncate("  "+typ, width)
	if isCursor {
		return cursorStyle.Render(header)
	}
	return typeStyle.Render(header)
}

func (m tuiModel) renderMethodLine(method string, isCursor, isSelected bool, width int) string {
	prefix := "    "
	if isSelected {
		prefix = "  ▶ "
	}
	line := truncate(prefix+method, width)

	switch {
	case isSelected && isCursor:
		return cursorStyle.Render(line)
	case isSelected:
		return selectedStyle.Render(line)
	case isCursor:
		return cursorStyle.Render(line)
	default:
		return dimStyle.Render(line)
	}
}

// --- Right panel: preview ---

type previewLine struct {
	content    string
	isOriginal bool
}

func (m tuiModel) renderPreview(width, height int) string {
	preview := sanitizePreviewText(m.getPreview())

	header := titleStyle.Render(fmt.Sprintf("%s / %s", m.selected.typ, m.selected.method))
	if m.selected.typ == "" {
		header = dimStyle.Render("no selection")
	}

	lineNumWidth := 4
	contentWidth := max(1, width-lineNumWidth)

	maxLines := max(1, height-3)

	allLines := splitPreview(preview, contentWidth)
	totalLines := 0
	for _, l := range allLines {
		if l.isOriginal {
			totalLines++
		}
	}

	start, end := clampScrollWindow(len(allLines), maxLines, m.previewScroll)
	visibleLines := allLines[start:end]

	trueLineNum := 0
	for _, l := range allLines[:start] {
		if l.isOriginal {
			trueLineNum++
		}
	}

	var b strings.Builder
	for _, pl := range visibleLines {
		if pl.isOriginal {
			trueLineNum++
			numStr := fmt.Sprintf("%d", trueLineNum)
			padding := lineNumWidth - len(numStr)
			if padding < 1 {
				padding = 1
			}
			b.WriteString(numStr)
			b.WriteString(strings.Repeat(" ", padding))
		} else {
			b.WriteString(strings.Repeat(" ", lineNumWidth))
		}
		b.WriteString(pl.content)
		b.WriteString("\n")
	}

	scrollInfo := scrollInfoText(totalLines, maxLines, end)
	return header + scrollInfo + "\n\n" + b.String()
}

// --- Help bar ---

func (m tuiModel) renderHelp() string {
	var parts []string
	switch m.focus {
	case focusList:
		parts = []string{"tab:fields", "j/k:navigate", "enter:select", "/:search", "ctrl+j/k:scroll preview", "q:quit"}
	case focusSearch:
		parts = []string{"type to filter", "enter/down:back to list", "esc:clear", "tab:fields"}
	case focusEncoding, focusOS:
		parts = []string{"h/l:change", "tab:next field", "enter:back to list"}
	case focusShell:
		if m.isCustomShell() {
			parts = []string{"type custom", "h:prev preset", "tab:next field", "enter:back to list"}
		} else {
			parts = []string{"h/l:change shell", "tab:next field", "enter:back to list"}
		}
	default:
		parts = []string{"type to edit", "tab:next field", "enter:back to list"}
	}

	return helpStyle.Render("  " + strings.Join(parts, "  │  "))
}

// --- Utility functions ---

// splitPreview splits text by newlines, expands tabs, and wraps long lines.
func splitPreview(text string, maxWidth int) []previewLine {
	raw := strings.Split(text, "\n")
	lines := make([]previewLine, 0, len(raw))
	for _, line := range raw {
		line = strings.ReplaceAll(line, "\t", "    ")
		wrapped := wrapLine(line, maxWidth)
		lines = append(lines, wrapped...)
	}
	return lines
}

// wrapLine wraps a line to maxWidth, returning previewLine structs.
func wrapLine(line string, maxWidth int) []previewLine {
	if maxWidth <= 0 {
		return []previewLine{{content: "", isOriginal: true}}
	}
	runes := []rune(line)
	if len(runes) <= maxWidth {
		return []previewLine{{content: line, isOriginal: true}}
	}
	var result []previewLine
	first := true
	for len(runes) > maxWidth {
		result = append(result, previewLine{content: string(runes[:maxWidth]), isOriginal: first})
		first = false
		runes = runes[maxWidth:]
	}
	if len(runes) > 0 {
		result = append(result, previewLine{content: string(runes), isOriginal: first})
	}
	return result
}

// clampScrollWindow returns the [start, end) range for a scroll window.
func clampScrollWindow(totalLines, maxLines, offset int) (start, end int) {
	start = min(max(0, offset), max(0, totalLines-maxLines))
	end = min(start+maxLines, totalLines)
	return
}

// scrollInfoText returns the scroll indicator string for the preview header.
func scrollInfoText(totalLines, maxLines, end int) string {
	if totalLines <= maxLines {
		return ""
	}
	remaining := totalLines - end
	if remaining > 0 {
		return dimStyle.Render(fmt.Sprintf(" \u25bc %d more lines", remaining))
	}
	return dimStyle.Render(" \u25b2 end")
}

// truncate cuts a string to maxWidth runes, appending an ellipsis if truncated.
func truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return string(runes[:maxWidth-1]) + "…"
}

// clampLines hard-truncates rendered output to at most maxLines lines.
func clampLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

func sanitizePreviewText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
