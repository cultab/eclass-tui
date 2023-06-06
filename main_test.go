package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/Huray-hub/eclass-utils/assignment/config"
	"github.com/Huray-hub/eclass-utils/auth"
	"github.com/Huray-hub/eclass-utils/course"
	tea "github.com/charmbracelet/bubbletea"
)

func test_panic(t *testing.T, expectPanic bool, f func()) {
	defer func() {
		panic_reason := recover()
		test_panicked := panic_reason != nil
		if expectPanic {
			if test_panicked {
				t.Logf("Panicked as expected with: %v", panic_reason)
			} else {
				t.Fatalf("Expected panic, but did not panic!")
			}
		} else if test_panicked {
			t.Fatalf("Unexpected panic with: %v", panic_reason)

		}
	}()

	f()
}

var baseConfig = config.Config{
	Credentials: auth.Credentials{
		Username: "asdf", // just to appease eclass-utils
		Password: "asdf",
	},
	Options: config.Options{
		PlainText:           false,
		IncludeExpired:      true,
		ExportICS:           false,
		ExcludedAssignments: map[string][]string{},
		Options: course.Options{
			BaseDomain:      "eclass.eclass.gr",
			ExcludedCourses: map[string]struct{}{},
		},
	},
}

type input_test struct {
	input       string
	expectPanic bool
	inject      []tea.Msg
}

// create a new test case, injecting multiple msgs
func NewInputTest(input string, expectPanic bool, msgs ...tea.Msg) input_test {
	return input_test{input, expectPanic, msgs}
}

func Test_different_inputs(t *testing.T) {

	conf := baseConfig

	tests := []input_test{
		NewInputTest("q", false),
		NewInputTest("xxq", false),
		NewInputTest("ccq", false, mockGetAssignments),
		NewInputTest("iiq", false, mockGetAssignments),
		NewInputTest("jkhlq", false, mockGetAssignments),
		NewInputTest("cxkixcjk ixckj q", false, mockGetAssignments),
		NewInputTest("b", true, mockGetAssignments, errorMsg{fmt.Errorf("forced error")}),
	}

	for i, test := range tests {
		t.Logf("Subtest #%d with input %s", i, test.input)

		test_panic(t, test.expectPanic, func() {
			err := error(nil)
			state := List

			w := NewWindow(state, conf, http.DefaultClient, err)

			in := strings.NewReader(test.input)

			p := tea.NewProgram(w, tea.WithoutRenderer(), tea.WithInput(in), tea.WithoutCatchPanics())

			// inject msg
			go func() {
				p.Send(test.inject)
			}()

			if _, err := p.Run(); err != nil {
				log.Panicf("Alas, there's been an error: %v", err)
			}
		})
	}
}

type config_test struct {
	config      config.Config
	expectPanic bool
}

func Test_bad_configs(t *testing.T) {

	other := baseConfig
	other.Options.BaseDomain = "nope"

	tests := []config_test{
		{baseConfig, false},
		{other, true},
	}

	for i, test := range tests {
		t.Logf("Subtest #%d", i)
		test_panic(t, test.expectPanic, func() {
			err := error(nil)
			state := List

			w := NewWindow(state, test.config, http.DefaultClient, err)

			in := strings.NewReader("              q")

			p := tea.NewProgram(w, tea.WithoutRenderer(), tea.WithInput(in), tea.WithoutCatchPanics())

			if _, err := p.Run(); err != nil {
				t.Logf("Alas, there's been an error: %v", err)
			}

		})
	}
}
