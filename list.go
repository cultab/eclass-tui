package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Huray-hub/eclass-utils/assignment"
	"github.com/Huray-hub/eclass-utils/assignment/config"
	"github.com/Huray-hub/eclass-utils/auth"
	"github.com/Huray-hub/eclass-utils/course"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	lip "github.com/charmbracelet/lipgloss"
)

type courseList struct {
	list       list.Model
	cache      []item
	showHidden bool
	keys       keyBinds
	config     config.Config
	testing    bool
	session    http.Client
}

func NewCourseList(conf config.Config, session *http.Client) courseList {
	if conf.Options.ExcludedAssignments == nil { // FIX: should't these be already made?
		conf.Options.ExcludedAssignments = make(map[string][]string)
	}
	if conf.Options.ExcludedCourses == nil {
		conf.Options.ExcludedCourses = make(map[string]struct{})
	}

	m := courseList{
		list:       list.New([]list.Item{}, itemDelegate{}, 0, 0),
		showHidden: false,
		keys:       newKeyBinds(),
		config:     conf,
		session:    *session,
	}

	m.list.Title = "Εργασίες"
	m.list.SetShowStatusBar(true)
	statusTime := time.Second * 2
	m.list.StatusMessageLifetime = statusTime
	m.list.SetStatusBarItemName("Εργασία", "Εργασίες")
	m.list.SetSpinner(spinner.Dot)

	// BUG: this looks like a bubbletea bug, spinner's style is unused in list/list.go
	// m.list.Styles.Spinner = lip.NewStyle().Background(uniwaOrange).Border(lip.DoubleBorder())
	m.list.StartSpinner()
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			m.keys.toggleHidden,
			m.keys.toggleHideCourse,
			m.keys.toggleHideAssignment,
			m.keys.saveConfig,
			m.keys.toggleIncludeExpired,
		}
	}
	return m
}

type keyBinds struct {
	toggleHideAssignment key.Binding
	toggleHideCourse     key.Binding
	toggleHidden         key.Binding
	saveConfig           key.Binding
	toggleIncludeExpired key.Binding
}

func newKeyBinds() keyBinds {
	return keyBinds{
		toggleHideAssignment: key.NewBinding(
			key.WithKeys("c", "ψ"),
			key.WithHelp("c|ψ", "Κρύψε εργασία"),
		),
		toggleHideCourse: key.NewBinding(
			key.WithKeys("x", "θ"),
			key.WithHelp("x|θ", "Κρύψε μάθημα"),
		),
		toggleHidden: key.NewBinding(
			key.WithKeys(tea.KeySpace.String()),
			key.WithHelp("space", "Εμφάνησε κρυμένες εργασίες"),
		),
		saveConfig: key.NewBinding(
			key.WithKeys("s", "σ"),
			key.WithHelp("s|σ", "Αποθήκευση"),
		),
		toggleIncludeExpired: key.NewBinding(
			key.WithKeys("i", "ι"),
			key.WithHelp("i|ι", "Συμπερήληψη εκπρόθεσμων"),
		),
	}
}

func (cl courseList) Init() tea.Cmd {
    // not called unless this is the main model
	return tea.Batch(
		cl.list.StartSpinner(),
		cl.getAssignmentsCmd(),
		updateTitleCmd,
	)
}

var docStyle = lip.NewStyle().Margin(1, 2)

func (cl courseList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, cl.keys.saveConfig):
			// TODO: this

			// err := config.Export(m.config.Options, m.config.Credentials)
			// if err != nil {
			// 	return m, errorCmd(err)
			// }
			cmd := cl.list.NewStatusMessage("Αποθηκέυση επιτυχής! (not really)")
			return cl, cmd
		case key.Matches(msg, cl.keys.toggleHideAssignment):
			i := cl.list.SelectedItem().(item)
			var statusCmd tea.Cmd

			if i.shouldHideAssignment(cl.config.Options.ExcludedAssignments) {
				delete(cl.config.Options.ExcludedAssignments, i.assignment.ID)
				statusCmd = cl.list.NewStatusMessage("Επανέφερες την εργασία " + i.assignment.Title + ".")
			} else {
				cl.config.Options.ExcludedAssignments[i.assignment.ID] = []string{}
				statusCmd = cl.list.NewStatusMessage("Έκρυψες την εργασία " + i.assignment.Title + ".")
			}
			return cl, tea.Batch(updateItemsCmd, statusCmd)
		case key.Matches(msg, cl.keys.toggleHideCourse):
			i := cl.list.SelectedItem().(item)
			var statusCmd tea.Cmd

			if i.shouldHideCourse(cl.config.Options.ExcludedCourses) {
				delete(cl.config.Options.ExcludedCourses, i.assignment.Course.ID)
				statusCmd = cl.list.NewStatusMessage("Επανέφερες τις εργασίες του μαθήματος " + i.assignment.Course.Name + ".")
			} else {
				cl.config.Options.ExcludedCourses[i.assignment.Course.ID] = struct{}{}
				statusCmd = cl.list.NewStatusMessage("Έκρυψες τις εργασίες του μαθήματος " + i.assignment.Course.Name + ".")
			}
			return cl, tea.Batch(updateItemsCmd, statusCmd)
		case key.Matches(msg, cl.keys.toggleHidden):
			cl.showHidden = !cl.showHidden
			return cl, tea.Batch(updateItemsCmd, updateTitleCmd)
		case key.Matches(msg, cl.keys.toggleIncludeExpired):
			cl.config.Options.IncludeExpired = !cl.config.Options.IncludeExpired
			cl.logWhileTesting("%t", cl.config.Options.IncludeExpired)
			return cl, tea.Batch(updateItemsCmd, updateTitleCmd)
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		cl.list.SetSize(msg.Width-h, msg.Height-v)
	case updateItems:
		log.Print("Update list of assignments from cache")
		// remove all items
		for item := range cl.list.Items() {
			cl.list.RemoveItem(item)
		}
		cl.list.SetItems(filterItems(cl.cache, cl.config.Options, cl.showHidden))
		return cl, nil
	case newItems:
		for _, it := range msg {
			cl.cache = append(cl.cache, it.(item))
		}
		log.Print("Loaded assignments")
		cl.list.StopSpinner()
		statusCmd := cl.list.NewStatusMessage("Φόρτωση επιτυχής!")
		return cl, tea.Batch(updateItemsCmd, statusCmd)
	case updateTitle:
		var title string
		if cl.showHidden {
			title += "Κρυμμένες εργασίες"
		} else {
			title += "Εργασίες"
		}
		if cl.config.Options.IncludeExpired {
			title += " | Εμφανίζονται Εκπρόθεσμες"
		}
		cl.list.Title = title
		return cl, nil
	case loginSuccess:
		return cl, tea.Batch(
			cl.list.StartSpinner(),
			cl.getAssignmentsCmd(),
			updateTitleCmd,
		)
	case errorMsg:
		log.Print(msg.err)
		return cl, nil
	}

	var cmd tea.Cmd
	cl.list, cmd = cl.list.Update(msg)

	return cl, cmd
}

func filterItems(items []item, opts config.Options, showHidden bool) []list.Item {
	var new_items []list.Item
	for _, item := range items {
		if item.ShouldHide(opts, showHidden) {
			new_items = append(new_items, item)
		}
	}

	return new_items
}

func (cl courseList) View() string {
	return docStyle.Render(cl.list.View())
}

type updateTitle struct{}
type writeConfig struct{}
type updateItems struct{}
type newItems []list.Item
type errorMsg struct{ err error }

func updateTitleCmd() tea.Msg { return updateTitle{} }
func writeConfigCmd() tea.Msg { return writeConfig{} }
func updateItemsCmd() tea.Msg { return updateItems{} }
func errorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		return errorMsg{err}
	}
}

func (e errorMsg) Error() string { return e.err.Error() }

func getAssignments(service assignment.Service) tea.Msg {
	a, err := service.FetchAssignments(context.Background())
	if err != nil {
		log.Println(err)
	}

	var items = make([]list.Item, len(a))

	for i, ass := range a {
		items[i] = item{
			assignment: ass,
		}
	}

	return newItems(items)
}

func (cl courseList) getAssignmentsCmd() tea.Cmd {
	if cl.testing {
		return mockGetAssignments
	}
	return func() tea.Msg {
		return getAllAssignments(cl.session, cl.config.Credentials, cl.config.Options.BaseDomain)
	}
}

func getAllAssignments(session http.Client, creds auth.Credentials, domain string) tea.Msg {
	conf := config.Config{
		Credentials: creds,

		Options: config.Options{
			PlainText:           false,
			IncludeExpired:      true,
			ExportICS:           false,
			ExcludedAssignments: make(map[string][]string),
			Options: course.Options{
				ExcludedCourses: make(map[string]struct{}),
				BaseDomain:      domain,
			},
		},
	}

	ser, err := assignment.NewService(context.Background(), conf, &session)
	if err != nil {
		log.Fatal(err)
	}

	a, err := ser.FetchAssignments(context.Background())
	if err != nil {
		return errorMsg{err}
	}
	var items = make([]list.Item, len(a))

	for i, ass := range a {
		items[i] = item{
			assignment: ass,
		}
	}

	return newItems(items)
}

func (cl courseList) logWhileTesting(format string, a ...any) {
	if cl.testing {
		fmt.Printf(format, a...)
	}
}

func mockGetAssignments() tea.Msg {
	a := []assignment.Assignment{ // {{{
		{
			ID: "A1",
			Course: &course.Course{
				ID:   "CS101",
				Name: "Name 1",
				URL:  "https://some.random.url",
			},
			Title:    "Course #1",
			Deadline: time.Now(),
			IsSent:   true,
		},
		{
			ID: "A2",
			Course: &course.Course{
				ID:   "CS302",
				Name: "Name 2",
				URL:  "https://some.random.url",
			},
			Title:    "Course #2",
			Deadline: time.Now(),
			IsSent:   false,
		},
		{
			ID: "A3",
			Course: &course.Course{
				ID:   "CS404",
				Name: "Name 0",
				URL:  "https://some.random.url",
			},
			Title:    "Course #3",
			Deadline: time.Now().Add(time.Hour),
			IsSent:   false,
		},
	} // }}}

	var items = make([]list.Item, len(a))

	for i, ass := range a {
		items[i] = item{
			assignment: ass,
		}
	}

	return newItems(items)
}
