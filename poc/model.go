package main

import (
	"fmt"

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
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4")).Padding(0, 1)
	infoStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true)
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("63"))
	dataStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("228"))
	notesStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("120"))
	usersStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("81"))
)

type model struct {
	state       state
	err         error
	spinner     spinner.Model
	accessToken string
	meData      string
	usersData   string
	notesData   string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{state: stateInit, spinner: s}
}

type authSuccessMsg string
type dataFetchedMsg struct {
	me    string
	users string
	notes string
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
		m.accessToken = string(msg)
		return m, fetchData(string(msg))
	case dataFetchedMsg:
		m.state = stateDone
		m.meData = msg.me
		m.usersData = msg.users
		m.notesData = msg.notes
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
	s := "\n" + titleStyle.Render("OAuth 2.0 TUI Client") + "\n\n"

	switch m.state {
	case stateInit:
		s += "Press " + lipgloss.NewStyle().Bold(true).Render("Enter") + " to open your browser and authenticate.\n"
		s += "Press 'q' or 'ctrl+c' to quit.\n"
	case stateWaiting:
		s += m.spinner.View() + " Waiting for browser authorization...\n"
	case stateFetching:
		s += m.spinner.View() + " Exchanging code & fetching data...\n"
	case stateDone:
		s += infoStyle.Render("✔ Authentication Successful!") + "\n\n"

		meBox    := boxStyle.Render(fmt.Sprintf("GET /api/me\n\n%s", dataStyle.Render(m.meData)))
		usersBox := boxStyle.Render(fmt.Sprintf("GET /api/admin/users\n\n%s", usersStyle.Render(m.usersData)))
		notesBox := boxStyle.Render(fmt.Sprintf("GET /api/notes\n\n%s", notesStyle.Render(m.notesData)))

		s += lipgloss.JoinHorizontal(lipgloss.Top, meBox, "  ", usersBox, "  ", notesBox) + "\n"
	case stateError:
		s += errorStyle.Render(fmt.Sprintf("✖ Error: %v", m.err)) + "\n"
	}

	return s + "\n"
}
