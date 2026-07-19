package cli

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"gorm.io/gorm"

	"github.com/Kartik-2239/openai-proxy/internal/db"
)

type screen int

const (
	screenMain screen = iota
	screenAPIKeyName
	screenLoadingModels
	screenModelSelect
	screenProviderStatus
	screenDone

	screenEditKeys
)

type model struct {
	db        *gorm.DB
	providers []providerDef
	screen    screen
	cursor    int
	pageSize  int
	nameInput textinput.Model
	message   string

	apiKeyName   string
	models       []modelChoice
	modelFilter  string
	selected     map[int]bool
	generatedKey string
}

type modelsMsg struct {
	models []modelChoice
	err    error
}

type mainOption struct {
	Label            string
	Screen           screen
	NeedsProviderKey bool
}

var mainOptions = []mainOption{
	{Label: "Create new API key", Screen: screenAPIKeyName, NeedsProviderKey: true},
	{Label: "Available provider keys", Screen: screenProviderStatus},
	{Label: "Edit keys", Screen: screenEditKeys},
}

func Run() error {
	dbPath := os.Getenv("PROXY_DB_PATH")
	if dbPath == "" {
		dbPath = "proxy.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return err
	}

	m := model{db: database, providers: loadProviders(), screen: screenMain, pageSize: 10, selected: map[int]bool{}, nameInput: newNameInput()}
	if len(m.providers) == 0 {
		var s strings.Builder
		for i, provider := range providerDefs {
			if i == len(providerDefs)-1 {
				s.WriteString("or ")
				s.WriteString(provider.EnvKey)
			} else {
				s.WriteString(provider.EnvKey)
				s.WriteString(", ")
			}
		}
		m.message = "No provider env keys found. Set " + s.String() + "."
	}

	_, err = tea.NewProgram(m).Run()
	return err
}

func (m model) Init() tea.Cmd { return textinput.Blink }

func newNameInput() textinput.Model {
	input := textinput.New()
	input.Prompt = " "
	input.Placeholder = "Name for api key (optional)"
	input.CharLimit = 64
	input.Width = 32
	input.Focus()
	return input
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		return m.updateKey(msg)
	case modelsMsg:
		if msg.err != nil {
			m.screen = screenMain
			m.message = msg.err.Error()
			return m, nil
		}
		m.models, m.cursor, m.modelFilter, m.selected = msg.models, 0, "", map[int]bool{}
		m.screen = screenModelSelect
	}
	return m, nil
}

func (m model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenMain:
		return m.updateMain(msg)
	case screenAPIKeyName:
		return m.updateTextInput(msg, m.startModelSelect)
	case screenModelSelect:
		return m.updateModelSelect(msg)
	case screenProviderStatus:
		if msg.String() == "enter" {
			m.screen, m.cursor = screenMain, 0
		}
	case screenDone:
		if msg.String() == "enter" {
			m.screen, m.cursor, m.message, m.generatedKey = screenMain, 0, "", ""
			m.nameInput = newNameInput()
		}
	}
	return m, nil
}

func (m model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.cursor = wrap(m.cursor-1, len(mainOptions))
	case "down", "j":
		m.cursor = wrap(m.cursor+1, len(mainOptions))
	case "enter":
		m.message = ""
		option := mainOptions[m.cursor]
		if option.NeedsProviderKey && len(m.providers) == 0 {
			m.message = "No provider env keys found. Add one in .env, then restart this CLI."
			return m, nil
		}
		m.screen, m.cursor = option.Screen, 0
		if m.screen == screenAPIKeyName {
			m.nameInput = newNameInput()
		}
	}
	return m, nil
}

func (m model) updateTextInput(msg tea.KeyMsg, submit func(string) (model, tea.Cmd)) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		return submit(strings.TrimSpace(m.nameInput.Value()))
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m model) updateModelSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredModelIndexes()
	switch msg.String() {
	case "up", "k":
		m.cursor = wrap(m.cursor-1, len(filtered))
	case "down", "j":
		m.cursor = wrap(m.cursor+1, len(filtered))
	case "pgup", "left":
		m.cursor = max(0, m.cursor-m.pageSize)
	case "pgdown", "right":
		m.cursor = min(max(0, len(filtered)-1), m.cursor+m.pageSize)
	case "backspace", "ctrl+h":
		if m.modelFilter != "" {
			m.modelFilter, m.cursor = m.modelFilter[:len(m.modelFilter)-1], 0
		}
	case " ":
		if len(filtered) > 0 {
			idx := filtered[m.cursor]
			m.selected[idx] = !m.selected[idx]
		}
	case "enter":
		return m.createAPIKey()
	default:
		if msg.Type == tea.KeyRunes {
			m.modelFilter += string(msg.Runes)
			m.cursor, m.message = 0, ""
		}
	}
	return m, nil
}

func (m model) startModelSelect(name string) (model, tea.Cmd) {
	if name == "" {
		name = "user-" + time.Now().Format("20060102150405")
	}
	m.apiKeyName, m.cursor, m.screen = name, 0, screenLoadingModels
	m.nameInput = newNameInput()
	return m, fetchModelsCmd(m.providers)
}

func (m model) createAPIKey() (tea.Model, tea.Cmd) {
	choices := m.selectedChoices()
	if len(choices) == 0 {
		m.message = "Select at least one model."
		return m, nil
	}
	key, err := createAPIKey(m.db, m.apiKeyName, choices)
	if err != nil {
		m.message = err.Error()
		return m, nil
	}
	m.generatedKey, m.message, m.screen = key, "", screenDone
	return m, nil
}

func (m model) selectedChoices() []modelChoice {
	choices := make([]modelChoice, 0, len(m.selected))
	for idx, selected := range m.selected {
		if selected {
			choices = append(choices, m.models[idx])
		}
	}
	return choices
}

func (m model) filteredModelIndexes() []int {
	filter := strings.ToLower(strings.TrimSpace(m.modelFilter))
	indexes := make([]int, 0, len(m.models))
	for i, model := range m.models {
		if filter == "" || strings.Contains(strings.ToLower(model.Provider+"/"+model.Model), filter) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}
