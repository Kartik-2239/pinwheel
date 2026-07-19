package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func (m model) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n\n", titleStyle.Render("LLM Proxy CLI"))
	if m.message != "" {
		fmt.Fprintf(&b, "%s\n\n", errorStyle.Render(m.message))
	}

	switch m.screen {
	case screenMain:
		menu(&b, mainOptionLabels(), m.cursor)
	case screenAPIKeyName:
		fmt.Fprintf(&b, " %s\n\n%s", m.nameInput.View(), helpStyle.Render("enter submits • empty name uses timestamp • q quit"))
	case screenLoadingModels:
		b.WriteString("Loading models from configured providers...")
	case screenModelSelect:
		m.modelList(&b)
	case screenProviderStatus:
		m.providerStatus(&b)
	case screenDone:
		fmt.Fprintf(&b, "%s\n\n", selectedStyle.Render("Done."))
		if m.generatedKey != "" {
			fmt.Fprintf(&b, "New API key:\n\n%s\n\n%s", m.generatedKey, helpStyle.Render("Copy it now. Only the hash was stored."))
		}
		fmt.Fprintf(&b, "\n\n%s", helpStyle.Render("enter main menu • q quit"))
	}
	return b.String()
}

func mainOptionLabels() []string {
	labels := make([]string, len(mainOptions))
	for i, option := range mainOptions {
		labels[i] = option.Label
	}
	return labels
}

func (m model) providerStatus(b *strings.Builder) {
	for _, def := range providerDefs {
		status := errorStyle.Render("missing")
		if m.hasProvider(def.Name) {
			status = selectedStyle.Render("available")
			fmt.Fprintf(b, "  %s (%s)\n", def.Name, status)
		} else {
			fmt.Fprintf(b, "  %s (%s) %s\n", def.Name, status, helpStyle.Render(fmt.Sprintf("export %s", def.EnvKey)))
		}
	}
	fmt.Fprintf(b, "\n%s", helpStyle.Render("Set keys in .env. This CLI only reads them. enter back • q quit"))
}

func (m model) hasProvider(name string) bool {
	for _, provider := range m.providers {
		if provider.Name == name {
			return true
		}
	}
	return false
}

func (m model) modelList(b *strings.Builder) {
	filtered := m.filteredModelIndexes()
	fmt.Fprintf(b, "Choose allowed models:\nFilter: %s\nSelected: %d\n\n", m.modelFilter, len(m.selectedChoices()))
	if len(filtered) == 0 {
		fmt.Fprintf(b, "No models match the filter.\n\n%s", helpStyle.Render("type filter • backspace clear • q quit"))
		return
	}

	start := (m.cursor / m.pageSize) * m.pageSize
	end := min(start+m.pageSize, len(filtered))
	for i := start; i < end; i++ {
		idx := filtered[i]
		choice := m.models[idx]
		b.WriteString(row(i == m.cursor, fmt.Sprintf("%s/%s", choice.Provider, choice.Model), m.selected[idx]))
	}
	pages := max(1, (len(filtered)+m.pageSize-1)/m.pageSize)
	fmt.Fprintf(b, "\n%s", helpStyle.Render(fmt.Sprintf("page %d/%d • type filter • space select • enter create • ←/→ page • q quit", start/m.pageSize+1, pages)))
}

func menu(b *strings.Builder, items []string, cursor int) {
	// b.WriteString(title)
	for i, item := range items {
		b.WriteString(row(i == cursor, item, false))
	}
	fmt.Fprintf(b, "\n%s", helpStyle.Render("↑/↓ move • enter select • q quit"))
}

func row(focused bool, label string, checked bool) string {
	line := "  "
	if focused {
		line = "> "
	}
	if checked {
		line += selectedStyle.Render("[x] ")
	} else if strings.Contains(label, "/") {
		line += "[ ] "
	}
	line += label
	if focused {
		line = selectedStyle.Render(line)
	}
	return line + "\n"
}
