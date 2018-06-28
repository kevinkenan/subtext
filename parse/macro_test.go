package parse

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	// "text/template"
	"github.com/kevinkenan/cobra"
)

func init() {
	cfg := cobra.NewTestingConfig(nil)
	cfg.LogPanicOnly()
	// cfg := cobra.NewTestingConfig([]string{"macro","parse","node","scan"})
	_ = cfg
	// cfg.SetDefault("logalltags", true)
}

func stopFmtUnusedError() {
	fmt.Println("")
}

// ValidateArguments ----------------------------------------------------------
// The initial opening paragraph command is skipped in these tests.

func TestValidateArgs(t *testing.T) {
	var m *Macro
	m = NewMacro("testCmd", "hi", nil, nil)
	testValidateArgs(t, newArgsCheckTestCase(m,
		"bare macro",
		"•testCmd[]", 0,
		"", false))

	opt := Optional{Name: "cThree", Default: ""}
	m = NewMacro("testMacro", "", []string{"aOne", "bTwo"}, []*Optional{&opt})
	testValidateArgs(t, newArgsCheckTestCase(m,
		"no arguments",
		"•testCmd[]", 0,
		"Line 1: command \"testMacro\" is missing 2 arguments: [aOne bTwo]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"an anon argument with 1 missing",
		"•testCmd[{arg}]", 0,
		"Line 1: command \"testMacro\" is missing 1 argument: [bTwo]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"right number of anon args",
		"•testCmd[{arg}{arg}]", 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"empty anon arguments",
		"•testCmd[{}{}]", 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"right number of anon args and one optional",
		"•testCmd[{arg}{arg}{arg}]", 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"one too many anonymous arguments",
		"•testCmd[{arg}{arg}{arg}{arg}]", 0,
		"Line 1: command \"testMacro\" contains 1 unknown argument: [#4]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"two too many anonymous arguments",
		"•testCmd[{arg}{arg}{arg}{arg}{arg}]", 0,
		"Line 1: command \"testMacro\" contains 2 unknown arguments: [#4 #5]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"right named args",
		"•testCmd[aOne={arg} bTwo={arg}]", 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"correct use of named optional args",
		"•testCmd[aOne={arg} bTwo={arg} cThree={arg}]", 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"named missing required argument",
		"•testCmd[aOne={arg}]", 0,
		"Line 1: command \"testMacro\" is missing 1 argument: [bTwo]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"unknown argument",
		"•testCmd[aOne={arg} bTwo={arg} xxx={arg}]", 0,
		"Line 1: command \"testMacro\" contains 1 unknown argument: [xxx]", true))
}

type argsCheckTestCase struct {
	name    string // Test name.
	*Macro         // The macro to be used for this test.
	command string // Command that invokes the macro.
	loc     int    // The index of the command in root.NodeList.
	exp     string // Remember the values are sorted in the tests.
	expErr  bool   // True indicates that we expect an error.
}

func newArgsCheckTestCase(m *Macro, n string, cmd string, loc int, exp string, err bool) *argsCheckTestCase {
	return &argsCheckTestCase{n, m, cmd, loc, exp, err}
}

func testValidateArgs(t *testing.T, test *argsCheckTestCase) {
	root, _ := ParsePlain(test.name, test.command)
	cmdNode := root.NodeList[test.loc] // +1 to skip the opening paragraph command
	args, err := test.Macro.ValidateArgs(cmdNode.(*Cmd))
	switch {
	case err != nil && !test.expErr:
		t.Errorf("%s\n unexpected failure: %s", test.name, err)
		return
	case err != nil && test.expErr:
		if err.Error() != test.exp {
			t.Errorf("%s\n  *result: %s\n*expected: %s", test.name, err, test.exp)
		}
		return
	case err == nil && test.expErr:
		t.Errorf("%s: unexpected success", test.name)
		return
	case err == nil && !test.expErr:
		keys := args.Keys() // keys are in a random changing order.
		sort.Strings(keys)  // sort them so we have something fixed in our tests.
		if strings.Join(keys, " ") != test.exp {
			t.Errorf("%s\n  *result: %s\n*expected: %s", test.name, keys, test.exp)
		}
	}
}
