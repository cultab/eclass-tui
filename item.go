package main

import (
	"fmt"
	"io"
	"time"

	"github.com/Huray-hub/eclass-utils/assignment"
	"github.com/Huray-hub/eclass-utils/assignment/config"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lip "github.com/charmbracelet/lipgloss"
)

type item struct {
	assignment assignment.Assignment
	hideReason string
}

func (i *item) ShouldHide(opts config.Options, showHidden bool) bool {
    hide := false
	if i.shouldHideAssignment(opts.ExcludedAssignments) {
		hide = true
	} else if i.shouldHideCourse(opts.ExcludedCourses) {
		hide = true
	} else if i.shouldHideExpired() && (!opts.IncludeExpired || showHidden) {
		// if the deadline has passed and we don't include expired OR we show hidden, hide this item
		hide = true
	}

	return hide == showHidden
}

func (i *item) shouldHideAssignment(excludedAssignments map[string][]string) bool {
	for hidden_ass := range excludedAssignments {
		if hidden_ass == i.assignment.ID {
			i.hideReason = "κρυμένη εργασία"
			return true
		}
	}
	return false
}

func (i *item) shouldHideCourse(excludedCourses map[string]struct{}) bool {
	for hidden_course := range excludedCourses {
		if hidden_course == i.assignment.Course.ID {
			i.hideReason = "εργασία κρυμένου μαθήματος"
			return true
		}
	}

	return false
}

func (i *item) shouldHideExpired() bool {
	if i.isExpired() {
		i.hideReason = "εκπρόθεσμη"
		return true
	}
	return false
}

func (i item) FilterValue() string { // TODO: keybinds to change this func to others
	return i.assignment.Title
}

func (i item) isExpired() bool {
    if i.assignment.Deadline == nil {
        return false
    }
	return i.assignment.Deadline.Before(time.Now())
}

var ( // {{{
	uniwaBlue      = lip.Color("#0b365b")
	hoverBg        = lip.Color("#030F27")
	uniwaLightBlue = lip.Color("#6eaede")
	uniwaOrange    = lip.Color("#e67c17")
	baseStyle      = lip.NewStyle().
			PaddingLeft(4).
			Foreground(lip.Color("#777777")).
			Faint(true)
	titleStyle = baseStyle.Copy().
			PaddingLeft(2).
			UnsetForeground().
			MaxWidth(47).
			Bold(true)
	lateStyle = baseStyle.Copy().
			Foreground(lip.Color("1"))
	doneStyle = baseStyle.Copy().
			Foreground(lip.Color("2"))
	normalStyle = lip.NewStyle().
			BorderStyle(lip.HiddenBorder()).
			BorderLeft(true)
	hoverStyle = lip.NewStyle().
			BorderStyle(lip.ThickBorder()).
			BorderLeft(true).
			BorderForeground(uniwaBlue).
			Foreground(uniwaLightBlue)
) // }}}

type itemDelegate struct{}

func (itemD itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	var title, course, date, hideReason string

	title = titleStyle.Render(i.assignment.Title)
	course = baseStyle.Render(i.assignment.Course.Name)

	if i.assignment.IsSent {
		date = doneStyle.Render("Παραδώθηκε πριν της:")
	} else {
		if i.isExpired() {
			date = lateStyle.Render("Εκπρόθεσμη:         ")
		} else {
			date = baseStyle.Render("Παράδωση εώς:       ")
		}
	}

    if i.assignment.Deadline == nil {
        date += baseStyle.Render("Χωρίς Προθεσμία")
    } else {
        date += baseStyle.Render(i.assignment.Deadline.Format("02/01/2006 15:04:05"))
    }


	if i.hideReason != "" {
		hideReason = baseStyle.Render("Κρυμένη γιατί είναι " + i.hideReason)
	}

	result := lip.JoinVertical(lip.Left, title, date, course)
	result = lip.JoinHorizontal(lip.Top, result, hideReason)

	if index == m.Index() {
		result = hoverStyle.Render(result)
	} else {
		result = normalStyle.Render(result)
	}

	fmt.Fprint(w, result)
}

func (itemD itemDelegate) Height() int {
	return 3
}

func (itemD itemDelegate) Spacing() int {
	return 0
}

func (itemD itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}
