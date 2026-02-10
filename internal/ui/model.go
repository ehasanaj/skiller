package ui

import (
	"fmt"
	"sort"
	"strings"

	"skiller/internal/config"
	"skiller/internal/install"
	"skiller/internal/scan"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type focusPane int

const (
	focusRegistries focusPane = iota
	focusSkills
	focusHarnesses
)

type inputTarget int

const (
	inputNone inputTarget = iota
	inputRegistry
	inputHarness
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmDeleteRegistry
	confirmDeleteHarness
	confirmUninstall
)

type harnessRowKind int

const (
	harnessRowHeader harnessRowKind = iota
	harnessRowSkill
)

type harnessRow struct {
	kind    harnessRowKind
	harness string
	skill   scan.Skill
}

type Model struct {
	width  int
	height int

	focus focusPane

	cfg        *config.Config
	configPath string

	knownHarnesses map[string]struct{}

	registries []string
	harnesses  []string

	registrySkills map[string][]scan.Skill
	harnessSkills  map[string][]scan.Skill
	harnessRows    []harnessRow

	selectedRegistry   int
	selectedSkill      int
	selectedHarnessRow int

	showInput   bool
	inputTarget inputTarget
	input       textinput.Model
	inputPrompt string

	showConfirm    bool
	confirmKind    confirmKind
	confirmMessage string

	pendingPath      string
	pendingHarness   string
	pendingSkillName string

	showConflict bool
	pendingSkill scan.Skill

	statusMessage string
	errorMessage  string
}

func NewModel() (*Model, error) {
	cfg, configPath, err := config.Load()
	if err != nil {
		return nil, err
	}

	input := textinput.New()
	input.Focus()
	input.Prompt = ""
	input.CharLimit = 4096
	input.Width = 80

	m := &Model{
		cfg:            cfg,
		configPath:     configPath,
		focus:          focusRegistries,
		registrySkills: map[string][]scan.Skill{},
		harnessSkills:  map[string][]scan.Skill{},
		input:          input,
	}

	m.refreshSources()
	m.rescan()

	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		return m, nil
	case tea.KeyMsg:
		if m.showInput {
			return m.updateInput(typed)
		}
		if m.showConfirm {
			return m.updateConfirm(typed)
		}
		if m.showConflict {
			return m.updateConflict(typed)
		}
		return m.updateNormal(typed)
	}

	return m, nil
}

func (m *Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.resetInput()
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.input.Value())
		if value == "" {
			m.errorMessage = "Path cannot be empty"
			return m, nil
		}

		var err error
		switch m.inputTarget {
		case inputRegistry:
			err = m.cfg.AddRegistry(value)
			if err == nil {
				m.statusMessage = "Added registry"
			}
		case inputHarness:
			err = m.cfg.AddHarness(value)
			if err == nil {
				m.statusMessage = "Added harness path"
			}
		}

		if err != nil {
			m.errorMessage = err.Error()
			return m, nil
		}

		if err := m.saveConfig(); err != nil {
			m.errorMessage = err.Error()
			return m, nil
		}

		m.resetInput()
		m.refreshSources()
		m.rescan()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		switch m.confirmKind {
		case confirmDeleteRegistry:
			m.cfg.RemoveRegistry(m.pendingPath)
			if err := m.saveConfig(); err != nil {
				m.errorMessage = err.Error()
			} else {
				m.statusMessage = "Removed registry"
				m.refreshSources()
				m.rescan()
			}
		case confirmDeleteHarness:
			m.cfg.RemoveHarness(m.pendingPath)
			if err := m.saveConfig(); err != nil {
				m.errorMessage = err.Error()
			} else {
				m.statusMessage = "Removed harness path"
				m.refreshSources()
				m.rescan()
			}
		case confirmUninstall:
			err := install.UninstallSkill(m.pendingHarness, m.pendingSkillName)
			if err != nil {
				m.errorMessage = err.Error()
			} else {
				m.statusMessage = "Uninstalled skill"
				m.rescan()
			}
		}
		m.resetConfirm()
		return m, nil
	case "n", "esc":
		m.resetConfirm()
		m.statusMessage = "Cancelled"
		return m, nil
	}

	return m, nil
}

func (m *Model) updateConflict(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o":
		m.installWithAction(install.ConflictOverwrite)
		return m, nil
	case "r":
		m.installWithAction(install.ConflictRename)
		return m, nil
	case "s", "n", "esc":
		m.showConflict = false
		m.statusMessage = "Skipped install"
		return m, nil
	}

	return m, nil
}

func (m *Model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab", "right", "l":
		m.focus = (m.focus + 1) % 3
		return m, nil
	case "shift+tab", "left", "h":
		m.focus = (m.focus + 2) % 3
		return m, nil
	case "up", "k":
		m.moveSelection(-1)
		return m, nil
	case "down", "j":
		m.moveSelection(1)
		return m, nil
	case "a":
		m.beginAddPath()
		return m, nil
	case "d":
		m.beginDeletePath()
		return m, nil
	case "i":
		m.beginInstall()
		return m, nil
	case "u":
		m.beginUninstall()
		return m, nil
	case "r":
		m.rescan()
		m.statusMessage = "Rescanned sources"
		m.errorMessage = ""
		return m, nil
	}

	return m, nil
}

func (m *Model) View() string {
	width := m.width
	if width <= 0 {
		width = 120
	}

	height := m.height
	if height <= 0 {
		height = 36
	}

	paneWidth := (width - 4) / 3
	if paneWidth < 30 {
		paneWidth = 30
	}

	header := m.renderHeader(width)
	paneHeight := maxInt(8, height-3)

	left := m.renderRegistriesPane(paneWidth, paneHeight)
	middle := m.renderSkillsPane(paneWidth, paneHeight)
	right := m.renderHarnessPane(paneWidth, paneHeight)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, middle, right)

	footer := m.renderFooter(width)
	status := m.renderStatus(width)
	overlay := m.renderOverlay(width)
	if overlay != "" {
		status = overlay
	}

	parts := []string{header, panes, footer, status}
	frame := strings.Join(parts, "\n")

	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, frame)
}

func (m *Model) renderHeader(width int) string {
	focus := "Registries"
	switch m.focus {
	case focusSkills:
		focus = "Registry Skills"
	case focusHarnesses:
		focus = "Harness Installs"
	}

	text := fmt.Sprintf("skiller | focus: %s", focus)
	return headerStyle.Width(width).Render(truncate(text, width))
}

func (m *Model) renderRegistriesPane(width, height int) string {
	title := paneTitleStyle(m.focus == focusRegistries).Render("Registries")

	lines := []string{title}
	if len(m.registries) == 0 {
		lines = append(lines, mutedStyle.Render("No registries. Press a to add."))
	} else {
		for i, registry := range m.registries {
			line := registry
			if i == m.selectedRegistry {
				line = selectedStyle.Render("> " + line)
			} else {
				line = "  " + line
			}
			lines = append(lines, truncate(line, width-2))
		}
	}

	return paneBoxStyle(width, height, m.focus == focusRegistries).Render(strings.Join(lines, "\n"))
}

func (m *Model) renderSkillsPane(width, height int) string {
	title := paneTitleStyle(m.focus == focusSkills).Render("Registry Skills")
	lines := []string{title}

	registry := m.selectedRegistryPath()
	if registry == "" {
		lines = append(lines, mutedStyle.Render("Select a registry to view skills."))
		return paneBoxStyle(width, height, m.focus == focusSkills).Render(strings.Join(lines, "\n"))
	}

	skills := m.registrySkills[registry]
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("Source: %s", registry)))

	if len(skills) == 0 {
		lines = append(lines, mutedStyle.Render("No SKILL.md folders found."))
	} else {
		for i, skill := range skills {
			line := skill.Name
			if i == m.selectedSkill {
				line = selectedStyle.Render("> " + line)
			} else {
				line = "  " + line
			}
			lines = append(lines, truncate(line, width-2))
		}
	}

	return paneBoxStyle(width, height, m.focus == focusSkills).Render(strings.Join(lines, "\n"))
}

func (m *Model) renderHarnessPane(width, height int) string {
	title := paneTitleStyle(m.focus == focusHarnesses).Render("Harness Installs")
	lines := []string{title}

	if len(m.harnessRows) == 0 {
		lines = append(lines, mutedStyle.Render("No harness paths. Press a to add."))
		return paneBoxStyle(width, height, m.focus == focusHarnesses).Render(strings.Join(lines, "\n"))
	}

	for i, row := range m.harnessRows {
		var line string
		if row.kind == harnessRowHeader {
			line = fmt.Sprintf("[%s]", row.harness)
		} else {
			line = "  - " + row.skill.Name
		}

		if i == m.selectedHarnessRow {
			line = selectedStyle.Render("> " + line)
		} else {
			line = "  " + line
		}

		lines = append(lines, truncate(line, width-2))
	}

	return paneBoxStyle(width, height, m.focus == focusHarnesses).Render(strings.Join(lines, "\n"))
}

func (m *Model) renderFooter(width int) string {
	text := "Nav: arrows/hjkl | pane: h/l/tab | a add path | d delete path | i install | u uninstall | r rescan | q quit"
	return helpStyle.Width(width).Render(truncate(text, width))
}

func (m *Model) renderStatus(width int) string {
	if m.errorMessage != "" {
		return errorStyle.Width(width).Render("Error: " + m.errorMessage)
	}
	if m.statusMessage != "" {
		return statusStyle.Width(width).Render(m.statusMessage)
	}
	return statusStyle.Width(width).Render("")
}

func (m *Model) renderOverlay(width int) string {
	switch {
	case m.showInput:
		prompt := fmt.Sprintf("%s: %s", m.inputPrompt, m.input.View())
		return overlayStyle.Width(width).Render(prompt + "  [enter save, esc cancel]")
	case m.showConfirm:
		return overlayStyle.Width(width).Render(m.confirmMessage + "  [y/n]")
	case m.showConflict:
		message := "Skill already exists in target harness. [o] overwrite  [r] rename  [s] skip"
		return overlayStyle.Width(width).Render(message)
	default:
		return ""
	}
}

func (m *Model) beginAddPath() {
	m.errorMessage = ""
	m.statusMessage = ""
	switch m.focus {
	case focusRegistries:
		m.inputPrompt = "Add registry path"
		m.inputTarget = inputRegistry
	case focusHarnesses:
		m.inputPrompt = "Add harness path"
		m.inputTarget = inputHarness
	default:
		m.statusMessage = "Switch to Registries or Harness Installs pane to add paths"
		return
	}
	m.input.SetValue("")
	m.showInput = true
}

func (m *Model) beginDeletePath() {
	m.errorMessage = ""
	m.statusMessage = ""

	switch m.focus {
	case focusRegistries:
		registry := m.selectedRegistryPath()
		if registry == "" {
			m.statusMessage = "No registry selected"
			return
		}
		m.showConfirm = true
		m.confirmKind = confirmDeleteRegistry
		m.pendingPath = registry
		m.confirmMessage = fmt.Sprintf("Delete registry %s?", registry)
	case focusHarnesses:
		harness := m.selectedHarnessPath()
		if harness == "" {
			m.statusMessage = "No harness selected"
			return
		}
		if !m.cfg.IsCustomHarness(harness) {
			m.statusMessage = "Auto-detected harness paths cannot be removed"
			return
		}
		m.showConfirm = true
		m.confirmKind = confirmDeleteHarness
		m.pendingPath = harness
		m.confirmMessage = fmt.Sprintf("Delete harness path %s?", harness)
	default:
		m.statusMessage = "Switch to Registries or Harness Installs pane to delete paths"
	}
}

func (m *Model) beginInstall() {
	m.errorMessage = ""
	m.statusMessage = ""

	skill, ok := m.selectedRegistrySkill()
	if !ok {
		m.statusMessage = "No skill selected"
		return
	}

	harness := m.selectedHarnessPath()
	if harness == "" {
		m.statusMessage = "No harness selected"
		return
	}

	result, err := install.InstallSkill(skill.Path, harness, install.ConflictSkip)
	if err != nil {
		m.errorMessage = err.Error()
		return
	}

	if result.Conflict {
		m.pendingSkill = skill
		m.pendingHarness = harness
		m.showConflict = true
		return
	}

	m.statusMessage = fmt.Sprintf("Installed %s", result.Name)
	m.rescan()
}

func (m *Model) beginUninstall() {
	m.errorMessage = ""
	m.statusMessage = ""

	row, ok := m.selectedHarnessRowValue()
	if !ok || row.kind != harnessRowSkill {
		m.statusMessage = "Select an installed skill in Harness Installs pane"
		return
	}

	m.pendingHarness = row.harness
	m.pendingSkillName = row.skill.Name
	m.showConfirm = true
	m.confirmKind = confirmUninstall
	m.confirmMessage = fmt.Sprintf("Uninstall %s from %s?", row.skill.Name, row.harness)
}

func (m *Model) installWithAction(action install.ConflictAction) {
	result, err := install.InstallSkill(m.pendingSkill.Path, m.pendingHarness, action)
	if err != nil {
		m.errorMessage = err.Error()
		m.showConflict = false
		return
	}

	m.showConflict = false
	if !result.Installed {
		m.statusMessage = "Skipped install"
		return
	}

	if result.Renamed {
		m.statusMessage = fmt.Sprintf("Installed as %s", result.Name)
	} else {
		m.statusMessage = fmt.Sprintf("Installed %s", result.Name)
	}

	m.rescan()
}

func (m *Model) saveConfig() error {
	return m.cfg.Save(m.configPath)
}

func (m *Model) refreshSources() {
	m.registries = append([]string(nil), m.cfg.Registries...)

	detected := config.DetectKnownHarnesses()
	m.harnesses = config.MergeUnique(m.cfg.Harnesses, detected)

	m.knownHarnesses = map[string]struct{}{}
	for _, known := range detected {
		m.knownHarnesses[known] = struct{}{}
	}

	sort.Strings(m.registries)
	sort.Strings(m.harnesses)

	if m.selectedRegistry >= len(m.registries) {
		m.selectedRegistry = maxInt(0, len(m.registries)-1)
	}
}

func (m *Model) rescan() {
	m.registrySkills = map[string][]scan.Skill{}
	m.harnessSkills = map[string][]scan.Skill{}

	for _, registry := range m.registries {
		skills, err := scan.ScanRegistry(registry)
		if err != nil {
			m.errorMessage = err.Error()
			continue
		}
		m.registrySkills[registry] = skills
	}

	for _, harness := range m.harnesses {
		skills, err := scan.ScanHarness(harness)
		if err != nil {
			m.errorMessage = err.Error()
			continue
		}
		m.harnessSkills[harness] = skills
	}

	m.rebuildHarnessRows()
	m.clampSelections()
}

func (m *Model) rebuildHarnessRows() {
	rows := make([]harnessRow, 0)
	for _, harness := range m.harnesses {
		rows = append(rows, harnessRow{kind: harnessRowHeader, harness: harness})
		for _, skill := range m.harnessSkills[harness] {
			rows = append(rows, harnessRow{kind: harnessRowSkill, harness: harness, skill: skill})
		}
	}
	m.harnessRows = rows
}

func (m *Model) clampSelections() {
	if m.selectedRegistry >= len(m.registries) {
		m.selectedRegistry = maxInt(0, len(m.registries)-1)
	}

	skills := m.skillsForSelectedRegistry()
	if m.selectedSkill >= len(skills) {
		m.selectedSkill = maxInt(0, len(skills)-1)
	}

	if m.selectedHarnessRow >= len(m.harnessRows) {
		m.selectedHarnessRow = maxInt(0, len(m.harnessRows)-1)
	}
	if m.selectedHarnessRow < 0 {
		m.selectedHarnessRow = 0
	}
}

func (m *Model) moveSelection(delta int) {
	switch m.focus {
	case focusRegistries:
		if len(m.registries) == 0 {
			return
		}
		previous := m.selectedRegistry
		m.selectedRegistry = clamp(m.selectedRegistry+delta, 0, len(m.registries)-1)
		if m.selectedRegistry != previous {
			m.selectedSkill = 0
		}
	case focusSkills:
		skills := m.skillsForSelectedRegistry()
		if len(skills) == 0 {
			return
		}
		m.selectedSkill = clamp(m.selectedSkill+delta, 0, len(skills)-1)
	case focusHarnesses:
		if len(m.harnessRows) == 0 {
			return
		}
		m.selectedHarnessRow = clamp(m.selectedHarnessRow+delta, 0, len(m.harnessRows)-1)
	}
}

func (m *Model) selectedRegistryPath() string {
	if len(m.registries) == 0 {
		return ""
	}
	if m.selectedRegistry < 0 || m.selectedRegistry >= len(m.registries) {
		return ""
	}
	return m.registries[m.selectedRegistry]
}

func (m *Model) skillsForSelectedRegistry() []scan.Skill {
	registry := m.selectedRegistryPath()
	if registry == "" {
		return []scan.Skill{}
	}
	return m.registrySkills[registry]
}

func (m *Model) selectedRegistrySkill() (scan.Skill, bool) {
	skills := m.skillsForSelectedRegistry()
	if len(skills) == 0 {
		return scan.Skill{}, false
	}
	if m.selectedSkill < 0 || m.selectedSkill >= len(skills) {
		return scan.Skill{}, false
	}
	return skills[m.selectedSkill], true
}

func (m *Model) selectedHarnessRowValue() (harnessRow, bool) {
	if len(m.harnessRows) == 0 {
		return harnessRow{}, false
	}
	if m.selectedHarnessRow < 0 || m.selectedHarnessRow >= len(m.harnessRows) {
		return harnessRow{}, false
	}
	return m.harnessRows[m.selectedHarnessRow], true
}

func (m *Model) selectedHarnessPath() string {
	row, ok := m.selectedHarnessRowValue()
	if !ok {
		return ""
	}
	return row.harness
}

func (m *Model) resetInput() {
	m.showInput = false
	m.inputTarget = inputNone
	m.inputPrompt = ""
	m.input.SetValue("")
}

func (m *Model) resetConfirm() {
	m.showConfirm = false
	m.confirmKind = confirmNone
	m.confirmMessage = ""
	m.pendingPath = ""
	m.pendingSkillName = ""
	m.pendingHarness = ""
}

func clamp(value, min, max int) int {
	if max < min {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncate(input string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	runes := []rune(input)
	if len(runes) <= maxWidth {
		return input
	}
	if maxWidth <= 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

var (
	paneBorder = lipgloss.RoundedBorder()

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1)

	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("231")).Background(lipgloss.Color("31"))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("249"))

	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	overlayStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("24")).Padding(0, 1)
)

func paneTitleStyle(active bool) lipgloss.Style {
	style := lipgloss.NewStyle().Bold(true)
	if active {
		return style.Foreground(lipgloss.Color("230")).Background(lipgloss.Color("25")).Padding(0, 1)
	}
	return style.Foreground(lipgloss.Color("250"))
}

func paneBoxStyle(width, height int, active bool) lipgloss.Style {
	style := lipgloss.NewStyle().Border(paneBorder).Padding(0, 1).Width(width).Height(height)
	if active {
		return style.BorderForeground(lipgloss.Color("45"))
	}
	return style.BorderForeground(lipgloss.Color("238"))
}
