package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Huray-hub/eclass-utils/assignment/config"
	tea "github.com/charmbracelet/bubbletea"
)

type ActiveWindow int

const (
	Login ActiveWindow = iota + 1
	List
)

func (s ActiveWindow) String() string {
	switch s {
	case 0:
		return "Uninitialized"
	case Login:
		return "Login"
	case List:
		return "List"
	default:
		return "Undefined"
	}
}

type window struct {
	login    form
	list     courseList
	active   ActiveWindow
	conf     config.Config
	session  *http.Client
	lastSize tea.WindowSizeMsg
}

func NewWindow(win ActiveWindow, conf config.Config, session *http.Client, err error) window {
	w := window{
		active:  win,
		conf:    conf,
		session: session,
	}

	if w.active == Login {
		w.login = NewForm(conf)
	}
	w.list = NewCourseList()

	return w
}

// used to set config and session of courseList
type loginSuccess struct {
	conf    config.Config
	session *http.Client
}

func (w window) Init() tea.Cmd {
	if w.active == List {
		return func() tea.Msg {
			return loginSuccess{
				conf:    w.conf,
				session: w.session,
			}
		}
	}
	return nil
}
func (w window) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case loginSuccess:
		w.active = List
		cmds = append(cmds, func() tea.Msg {
			return w.lastSize
		})
	case tea.WindowSizeMsg:
		// keep track of last WindowSizeMsg, to pass onto switched to window
		w.lastSize = msg
	case tea.KeyMsg:
		switch key := msg.String(); key {
		case "ctrl+c":
			log.Println("quit")
			return w, tea.Quit
		}
	}

	switch w.active {
	case Login:
		l, cmd := w.login.Update(msg)
		w.login = l.(form)
		cmds = append(cmds, cmd)
	case List:
		l, cmd := w.list.Update(msg)
		w.list = l.(courseList)
		cmds = append(cmds, cmd)
	default:
		log.Panic(fmt.Errorf("window state %s", w.active))
		return w, nil // unreachable ?
	}
	return w, tea.Batch(cmds...)
}

func (w window) View() string {
	switch w.active {
	case Login:
		return w.login.View()
	case List:
		return w.list.View()
	default:
		log.Panic(fmt.Errorf("window state %s", w.active))
		return "" // unreachable
	}
}
