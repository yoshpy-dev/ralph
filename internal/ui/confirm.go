package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ConfirmYesMsg is sent when the user confirms the dialog.
type ConfirmYesMsg struct {
	Tag string // identifies which confirmation this responds to
}

// ConfirmNoMsg is sent when the user cancels the dialog.
type ConfirmNoMsg struct {
	Tag string
}

// ConfirmModel is a modal confirmation dialog.
// It captures y/Enter for yes and n/Esc for no.
type ConfirmModel struct {
	Message string
	Tag     string // arbitrary tag to identify the source action
	Visible bool
}

// NewConfirmModel creates a new hidden confirmation dialog.
func NewConfirmModel() ConfirmModel {
	return ConfirmModel{}
}

// Show makes the dialog visible with the given message and tag.
func (m ConfirmModel) Show(message, tag string) ConfirmModel {
	m.Message = message
	m.Tag = tag
	m.Visible = true
	return m
}

// Hide dismisses the dialog.
func (m ConfirmModel) Hide() ConfirmModel {
	m.Visible = false
	m.Message = ""
	m.Tag = ""
	return m
}

// Update handles key input for the confirmation dialog.
// Returns the updated model and any resulting command.
func (m ConfirmModel) Update(msg tea.Msg) (ConfirmModel, tea.Cmd) {
	if !m.Visible {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "y", "Y", "enter":
			tag := m.Tag
			m = m.Hide()
			return m, func() tea.Msg { return ConfirmYesMsg{Tag: tag} }
		case "n", "N", "esc":
			tag := m.Tag
			m = m.Hide()
			return m, func() tea.Msg { return ConfirmNoMsg{Tag: tag} }
		}
	}
	return m, nil
}

// View renders the confirmation dialog overlay.
func (m ConfirmModel) View() string {
	if !m.Visible {
		return ""
	}

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 3).
		Align(lipgloss.Center)

	prompt := fmt.Sprintf("%s\n\n[y/Enter] Yes  [n/Esc] Cancel", m.Message)
	return dialogStyle.Render(prompt)
}
