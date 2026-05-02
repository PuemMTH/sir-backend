package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type state int

const (
	stateInit state = iota
	stateWaiting
	stateFetching
	stateDone
	stateError
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	infoStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	failStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F57")).Bold(true)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	bodyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	rowStyle    = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(0, 1).
			MarginBottom(1)
)

type model struct {
	state        state
	err          error
	spinner      spinner.Model
	accessToken  string
	refreshToken string
	results      []result
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{state: stateInit, spinner: s}
}

type dataFetchedMsg struct {
	results []result
}
type errMsg struct{ err error }

func (e errMsg) Error() string { return e.err.Error() }

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
		if m.state == stateInit && msg.String() == "enter" {
			m.state = stateWaiting
			return m, tea.Batch(m.spinner.Tick, startAuthFlow())
		}
	case authSuccessMsg:
		m.state = stateFetching
		m.accessToken = msg.accessToken
		m.refreshToken = msg.refreshToken
		return m, fetchData(msg.accessToken, msg.refreshToken)
	case dataFetchedMsg:
		m.state = stateDone
		m.results = msg.results
		return m, tea.Quit
	case errMsg:
		m.state = stateError
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	s := "\n" + titleStyle.Render("sir-backend API Demo") + "\n\n"

	switch m.state {
	case stateInit:
		s += "Press " + lipgloss.NewStyle().Bold(true).Render("Enter") + " to open your browser and authenticate.\n"
		s += "Press 'q' or 'ctrl+c' to quit.\n"

	case stateWaiting:
		s += m.spinner.View() + " Waiting for browser authorization...\n"

	case stateFetching:
		s += m.spinner.View() + " Running API calls...\n"

	case stateDone:
		s += infoStyle.Render("✔ All API calls complete") + "\n\n"

		// Split results into two columns.
		mid := (len(m.results) + 1) / 2
		left := renderColumn(m.results[:mid])
		right := renderColumn(m.results[mid:])
		s += lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right) + "\n"

	case stateError:
		s += errorStyle.Render(fmt.Sprintf("✖ Error: %v", m.err)) + "\n"
	}

	return s + "\n"
}

func renderColumn(results []result) string {
	var rows []string
	for _, r := range results {
		icon := okStyle.Render("✔")
		if r.status == 0 || r.status >= 400 {
			icon = failStyle.Render("✖")
		}

		header := fmt.Sprintf("%s  %s  %s",
			icon,
			statusStyle.Render(statusText(r.status)),
			labelStyle.Render(r.label),
		)

		// Truncate body to keep boxes compact.
		body := r.body
		lines := strings.Split(body, "\n")
		if len(lines) > 6 {
			lines = append(lines[:6], "  ...")
		}
		body = strings.Join(lines, "\n")

		content := header + "\n" + bodyStyle.Render(body)
		rows = append(rows, rowStyle.Render(content))
	}
	return strings.Join(rows, "\n")
}

func statusText(code int) string {
	if code == 0 {
		return "ERR"
	}
	text := http.StatusText(code)
	if text == "" {
		return fmt.Sprintf("%d", code)
	}
	return fmt.Sprintf("%d %s", code, text)
}
