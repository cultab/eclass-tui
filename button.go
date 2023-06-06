package main

import (
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	lip "github.com/charmbracelet/lipgloss"
)

var (
	buttonFocusColor = lip.Color("5")
	buttonStyle      = lip.NewStyle().Border(lip.NormalBorder())
	buttonFocusStyle = buttonStyle.Copy().Foreground(buttonFocusColor).BorderForeground(buttonFocusColor)
)

type button struct {
	style      lip.Style
	focusStyle lip.Style
	text       string
	focus      bool
	Cursor     cursor.Model
}

func NewButton(text string) button {
	c := cursor.New()
	c.SetMode(cursor.CursorHide)
	return button{
		style:      buttonStyle,
		focusStyle: buttonFocusStyle,
		text:       text,
		focus:      false,
		Cursor:     c,
	}
}

func (b *button) Focus() tea.Cmd {
	b.focus = true
	return b.Cursor.Focus()
}

func (b *button) Blur() {
	b.focus = false
	b.Cursor.Blur()
}

func (b button) Init() tea.Cmd {
	return nil
}

func (b button) Update(msg tea.Msg) (button, tea.Cmd) {
	if !b.focus {
		return b, nil
	}
	c, cmd := b.Cursor.Update(msg)
	b.Cursor = c
	return b, cmd
}

func (b button) View() string {
	var str string
	if b.focus {
		str = b.focusStyle.Render(b.text)
	} else {
		str = b.style.Render(b.text)
	}
	return str
}
