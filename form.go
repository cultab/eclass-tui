package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/Huray-hub/eclass-utils/assignment/config"
	"github.com/Huray-hub/eclass-utils/auth"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lip "github.com/charmbracelet/lipgloss"
)

type form struct {
	fields          []textinput.Model
	style           lip.Style
	selectedInput   int
	conf            config.Config
	loginFailed     int
	loginFailReason string
}

const (
	Username = iota
	Password
	Domain
)

var defaultFormStyle = lip.NewStyle().
	Align(lip.Center, lip.Center)

type invalid struct {
	why string
}

func (err invalid) Error() string {
	return err.why
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (f form) validate() bool {
	errs := make([]error, 3)
	errs[Username] = validateUsername(f.fields[Username].Value())
	errs[Password] = validatePassword(f.fields[Password].Value())
	errs[Domain] = validateDomain(f.fields[Domain].Value())

	for _, v := range errs {
		if v != nil {
			return false
		}
	}
	return true
}

func validateDomain(domain string) error {
	err := validateLength(domain, "Domain")
	if err != nil {
		return err
	}

	prefix := "eclass."

	// if the domain so far does not have the prefix
	// where prefix is truncated to the size of the domain
	if !strings.HasPrefix(domain, prefix[0:min(len(domain), len(prefix))]) {
		return invalid{
			why: "Domain must start with: " + prefix + "",
		}
	}

	return nil
}

func validateUsername(name string) error {
	return validateLength(name, "Username")
}

func validatePassword(pass string) error {
	return validateLength(pass, "Password")
}

func validateLength(in string, what string) error {
	if len(in) == 0 {
		return invalid{
			why: what + " must be non empty!",
		}
	}
	return nil
}

func NewForm(conf config.Config) form {
	username := textinput.New()
	username.Prompt = "Username: "
	username.SetValue(conf.Credentials.Username)
	username.Validate = validateUsername
	username.Placeholder = " Όνομα χρήστη (Username)" // BUG: without space before 'Ό', garbage is printed

	password := textinput.New()
	password.Prompt = "Password: "
	username.SetValue(conf.Credentials.Password)
	password.Validate = validatePassword
	password.EchoCharacter = '*'
	password.EchoMode = textinput.EchoPassword
	password.Placeholder = " Συνθηματικό (Password)" // BUG: without space before 'Σ', garbage is printed

	domain := textinput.New()
	domain.Validate = validateDomain
	domain.SetValue(conf.Options.BaseDomain)
	domain.Prompt = "Domain:   "
	domain.Placeholder = " eclass.<domain>.<xyz>"

	return form{
		fields: []textinput.Model{
			Username: username,
			Password: password,
			Domain:   domain,
		},
		style:         defaultFormStyle,
		selectedInput: 0,
		conf:          conf,
	}
}

func (f form) Init() tea.Cmd {
	return nil
}

func (f form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		switch key {
		case "tab":
			if f.selectedInput != Domain {
				f.selectedInput++
			}
		case "shift+tab":
			if f.selectedInput != Username {
				f.selectedInput--
			}
		case "enter":
			if f.selectedInput != Domain {
				f.selectedInput++
			} else if f.validate() {
				log.Println("Attempting login..")
				return f, f.loginCmd
			}
		}
	case tea.WindowSizeMsg:
		f.style.Height(msg.Height)
		f.style.Width(msg.Width)
	case loginSuccess:
		log.Println("Login success!")
	case loginFail:
		f.loginFailed++
		f.loginFailReason = msg.reason
		log.Println(msg.Err)
	}

	cmd = f.updateFields(msg)
	return f, cmd
}

type loginFail struct {
	Err    error
	reason string
}

func (l loginFail) Error() string {
	return l.Err.Error()
}

type startSpinner struct{}

func startSpinnerCmd() tea.Msg {
	return startSpinner{}
}

func (f *form) loginCmd() tea.Msg {

	// update config
	f.conf.Credentials = auth.Credentials{
		Username: f.fields[Username].Value(),
		Password: f.fields[Password].Value(),
	}
	f.conf.Options.BaseDomain = f.fields[Domain].Value()

	client, err := auth.Session(
		context.Background(),
		"https://"+f.fields[Domain].Value(),
		f.conf.Credentials,
		nil,
	)

	if err != nil {
        var errInvalidDomain auth.ErrInvalidDomain
        var errUrlError *url.Error
        log.Printf("%T", err)
		switch {
		case errors.Is(err, auth.ErrNoCredentials):
			return loginFail{fmt.Errorf("login failed: %w", err), "Δεν δώθηκε password ή username"}
		case errors.Is(err, auth.ErrInvalidCredentials): log.Print("invalid creds")
			return loginFail{fmt.Errorf("login failed: %w", err), "Λάθος username ή password"}
        case errors.As(err, &errUrlError):
            return loginFail{fmt.Errorf("login failed: %w", err), "Λάθος domain"}
		case errors.Is(err, &errInvalidDomain):
			return loginFail{fmt.Errorf("login failed: %w", err), "Λάθος domain"}
		}
	}

	if client == nil {
		return loginFail{fmt.Errorf("login failed client: %v", err), "??unexpected login failure"}
	}

	return loginSuccess{
		conf:    f.conf,
		session: client,
	}
}

func (f *form) updateFields(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(f.fields)+1)

	for i, field := range f.fields {
		if i == f.selectedInput {
			field.Focus()
			field.SetValue(field.Value()) // HACK: yes
		} else {
			field.Blur()
		}
		f.fields[i], cmds[i] = field.Update(msg)
	}

	return tea.Batch(cmds...)
}

var (
	selectedInputStyle = lip.NewStyle().Foreground(lip.Color("5"))
	warningStyle       = lip.NewStyle().Foreground(lip.Color("1"))
	inputStyle         = lip.NewStyle()
	boxStyle           = lip.NewStyle().BorderStyle(lip.RoundedBorder()).Width(64)
)

func (f form) View() string {
	var inputsBoxes []string

	var warnings []string
	for i, field := range f.fields {
		if i == f.selectedInput {
			inputsBoxes = append(inputsBoxes, selectedInputStyle.Render(field.View()))
		} else {
			inputsBoxes = append(inputsBoxes, inputStyle.Render(field.View()))
		}
		if field.Err != nil {
			warnings = append(warnings, warningStyle.Render(field.Err.Error()))
		}
	}

	var loginFailed_msg string
	if f.loginFailed == 1 {
		loginFailed_msg = "Failed to login: " + f.loginFailReason
	} else if f.loginFailed > 1 {
		loginFailed_msg = fmt.Sprintf("Failed to login(%d): %s", f.loginFailed-1, f.loginFailReason)

	}
	warnings = append(warnings, warningStyle.Render(loginFailed_msg))

	str := lip.JoinVertical(lip.Left,
		boxStyle.Render(
			lip.JoinVertical(lip.Left, inputsBoxes...),
		),
		boxStyle.Render(
			lip.JoinVertical(lip.Left, warnings...),
		),
	)

	return f.style.Render(str)
}
