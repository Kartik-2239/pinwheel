package cli

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"gorm.io/gorm"

	"github.com/Kartik-2239/pinwheel/internal/db"
)

type screen int

const (
	screenMain screen = iota
	screenAPIKeyName
	screenLoadingModels
	screenModelSelect
	screenProviderStatus
	screenDone

	screenAPIKeyLimits
	screenApiKeyCostLimit
	screenApiKeyExpiration

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

	apiKeyName  string
	models      []modelChoice
	modelFilter string
	selected    map[int]bool

	apiKeyLimitSelections map[screen]bool
	apiKeyLimitQueue      []screen
	apiKeyCostInput       textinput.Model
	apiKeyCostLimit       int64
	apiKeyExpiration      time.Time
	generatedKey          string
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

type apiKeyLimitsOption struct {
	Label string
	screen
}

var apiKeyLimitsOptions = []apiKeyLimitsOption{
	// {Label: "Set request limit per 5-hour window"},
	{Label: "Set total cost budget", screen: screenApiKeyCostLimit},
	{Label: "Set expiration date", screen: screenApiKeyExpiration},
}

type expirationOption struct {
	Label string
	Hours int
}

var expirationOptions = []expirationOption{
	{Label: "3 hours", Hours: 3},
	{Label: "1 day", Hours: 24},
	{Label: "7 days", Hours: 168},
	{Label: "30 days", Hours: 720},
	{Label: "Never", Hours: 0},
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

	m := model{db: database, providers: loadProviders(), screen: screenMain, pageSize: 10, selected: map[int]bool{}, nameInput: newNameInput(), apiKeyCostInput: newCostInput(), apiKeyLimitSelections: map[screen]bool{}}
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

func newCostInput() textinput.Model {
	input := textinput.New()
	input.Prompt = " "
	input.Placeholder = "Cost limit in dollars"
	input.CharLimit = 18
	input.Width = 24
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
	case screenAPIKeyLimits:
		return m.updateAPIKeyLimits(msg)
	case screenApiKeyCostLimit:
		return m.updateApiKeyCostInput(msg)
	case screenApiKeyExpiration:
		return m.updateApiKeyExpirationInput(msg)

	case screenProviderStatus:
		if msg.String() == "enter" {
			m.screen, m.cursor = screenMain, 0
		}
	case screenDone:
		if msg.String() == "enter" {
			m.screen, m.cursor, m.message, m.generatedKey = screenMain, 0, "", ""
			m.nameInput = newNameInput()
			m = m.resetAPIKeyLimitState()
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
			m = m.resetAPIKeyLimitState()
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
		m.screen, m.cursor = screenAPIKeyLimits, 0
		return m, nil
	default:
		if msg.Type == tea.KeyRunes {
			m.modelFilter += string(msg.Runes)
			m.cursor, m.message = 0, ""
		}
	}
	return m, nil
}

func (m model) updateAPIKeyLimits(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.cursor = wrap(m.cursor-1, len(apiKeyLimitsOptions))
	case "down", "j":
		m.cursor = wrap(m.cursor+1, len(apiKeyLimitsOptions))
	case " ", "space":
		if m.apiKeyLimitSelections == nil {
			m.apiKeyLimitSelections = map[screen]bool{}
		}
		selectedScreen := apiKeyLimitsOptions[m.cursor].screen
		if m.apiKeyLimitSelections[selectedScreen] {
			delete(m.apiKeyLimitSelections, selectedScreen)
		} else {
			m.apiKeyLimitSelections[selectedScreen] = true
		}
	case "enter":
		m.apiKeyLimitQueue = m.selectedAPIKeyLimitScreens()
		return m.advanceAPIKeyLimitQueue()
	}
	return m, nil
}

func (m model) updateApiKeyCostInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		value := strings.TrimSpace(m.apiKeyCostInput.Value())
		costLimit, err := strconv.ParseInt(value, 10, 64)
		if value == "" || err != nil || costLimit < 0 {
			m.message = "Enter a valid whole-number cost limit."
			return m, nil
		}
		m.apiKeyCostLimit = costLimit * 1e6 // Convert dollars to micros
		m.apiKeyCostInput = newCostInput()
		m.message = ""
		return m.advanceAPIKeyLimitQueue()
	case "backspace", "ctrl+h", "delete", "left", "right", "home", "end":
	default:
		if msg.Type == tea.KeyRunes {
			for _, r := range msg.Runes {
				if r < '0' || r > '9' {
					return m, nil
				}
			}
		}
	}

	var cmd tea.Cmd
	m.apiKeyCostInput, cmd = m.apiKeyCostInput.Update(msg)
	return m, cmd
}

func (m model) updateApiKeyExpirationInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.cursor = wrap(m.cursor-1, len(expirationOptions))
	case "down", "j":
		m.cursor = wrap(m.cursor+1, len(expirationOptions))
	case "enter":
		selectedOption := expirationOptions[m.cursor]
		if selectedOption.Hours == 0 {
			m.apiKeyExpiration = time.Time{} // No expiration
		} else {
			m.apiKeyExpiration = time.Now().Add(time.Duration(selectedOption.Hours) * time.Hour)
		}
		m.message = ""
		return m.advanceAPIKeyLimitQueue()
	}
	return m, nil
}

func (m model) selectedAPIKeyLimitScreens() []screen {
	queue := make([]screen, 0, len(apiKeyLimitsOptions))
	for _, option := range apiKeyLimitsOptions {
		if m.apiKeyLimitSelections[option.screen] {
			queue = append(queue, option.screen)
		}
	}
	return queue
}

func (m model) advanceAPIKeyLimitQueue() (tea.Model, tea.Cmd) {
	if len(m.apiKeyLimitQueue) == 0 {
		return m.createAPIKey()
	}

	m.screen, m.cursor = m.apiKeyLimitQueue[0], 0
	m.apiKeyLimitQueue = m.apiKeyLimitQueue[1:]
	if m.screen == screenApiKeyCostLimit {
		m.apiKeyCostInput.Focus()
	}
	return m, nil
}

func (m model) resetAPIKeyLimitState() model {
	m.apiKeyLimitSelections = map[screen]bool{}
	m.apiKeyLimitQueue = nil
	m.apiKeyCostInput = newCostInput()
	m.apiKeyCostLimit = 0
	m.apiKeyExpiration = time.Time{}
	return m
}

func (m model) startModelSelect(name string) (model, tea.Cmd) {
	if name == "" {
		name = "user-" + time.Now().Format("20060102150405")
	}
	m.apiKeyName, m.cursor, m.screen = name, 0, screenLoadingModels
	m.nameInput = newNameInput()
	m = m.resetAPIKeyLimitState()
	return m, fetchModelsCmd(m.providers)
}

func (m model) createAPIKey() (tea.Model, tea.Cmd) {
	choices := m.selectedChoices()
	if len(choices) == 0 {
		m.message = "Select at least one model."
		return m, nil
	}
	var costLimit *int64
	if m.apiKeyLimitSelections[screenApiKeyCostLimit] {
		costLimit = &m.apiKeyCostLimit
	}

	var expiration *time.Time
	if m.apiKeyLimitSelections[screenApiKeyExpiration] && !m.apiKeyExpiration.IsZero() {
		expiration = &m.apiKeyExpiration
	}

	key, err := createAPIKey(m.db, m.apiKeyName, choices, costLimit, expiration)
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
