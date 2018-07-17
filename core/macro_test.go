package core

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
		newPlainTestDoc("•testCmd[]"), 0,
		"", false))

	opt := Optional{Name: "cThree", Default: ""}
	m = NewMacro("testMacro", "", []string{"aOne", "bTwo"}, []*Optional{&opt})
	testValidateArgs(t, newArgsCheckTestCase(m,
		"no arguments",
		newPlainTestDoc("•testCmd[]"), 0,
		"Line 1: command \"testMacro\" is missing 2 arguments: [aOne bTwo]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"an anon argument with 1 missing",
		newPlainTestDoc("•testCmd[{arg}]"), 0,
		"Line 1: command \"testMacro\" is missing 1 argument: [bTwo]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"right number of anon args",
		newPlainTestDoc("•testCmd[{arg}{arg}]"), 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"empty anon arguments",
		newPlainTestDoc("•testCmd[{}{}]"), 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"right number of anon args and one optional",
		newPlainTestDoc("•testCmd[{arg}{arg}{arg}]"), 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"one too many anonymous arguments",
		newPlainTestDoc("•testCmd[{arg}{arg}{arg}{arg}]"), 0,
		"Line 1: command \"testMacro\" contains 1 unknown argument: [#4]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"two too many anonymous arguments",
		newPlainTestDoc("•testCmd[{arg}{arg}{arg}{arg}{arg}]"), 0,
		"Line 1: command \"testMacro\" contains 2 unknown arguments: [#4 #5]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"right named args",
		newPlainTestDoc("•testCmd[aOne={arg} bTwo={arg}]"), 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"correct use of named optional args",
		newPlainTestDoc("•testCmd[aOne={arg} bTwo={arg} cThree={arg}]"), 0,
		"aOne bTwo cThree", false))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"named missing required argument",
		newPlainTestDoc("•testCmd[aOne={arg}]"), 0,
		"Line 1: command \"testMacro\" is missing 1 argument: [bTwo]", true))
	testValidateArgs(t, newArgsCheckTestCase(m,
		"unknown argument",
		newPlainTestDoc("•testCmd[aOne={arg} bTwo={arg} xxx={arg}]"), 0,
		"Line 1: command \"testMacro\" contains 1 unknown argument: [xxx]", true))
}

type argsCheckTestCase struct {
	name    string   // Test name.
	*Macro           // The macro to be used for this test.
	command Document // Command that invokes the macro.
	loc     int      // The index of the command in root.NodeList.
	exp     string   // Remember the values are sorted in the tests.
	expErr  bool     // True indicates that we expect an error.
}

func newArgsCheckTestCase(m *Macro, n string, cmd Document, loc int, exp string, err bool) *argsCheckTestCase {
	return &argsCheckTestCase{n, m, cmd, loc, exp, err}
}

func testValidateArgs(t *testing.T, test *argsCheckTestCase) {
	var err error
	// opt := &Options{Plain: true}
	// opt.Macros = NewMacroMap()

	f := NewFolio()
	// d := NewDoc("testdoc", "testpath")
	// f.Append(d)
	f.Macros[MacroType{"testCmd", ""}] = NewMacro("testCmd", "", nil, nil)
	f.Macros[MacroType{"sys.Z", ""}] = NewMacro("sys.Z", "", nil, nil)
	f.AddMacro(test.Macro)
	err = f.AppendDoc(&test.command)
	if err != nil {
		t.Errorf(err.Error())
	}

	root, err := Parse(f.GetDocs()[0])
	if err != nil {
		t.Errorf("%s:\nParse failed: %s", test.name, err)
		return
	}

	if root == nil {
		t.Errorf("%s:\nRoot is nil", test.name)
		return
	}

	cmdNode := root.NodeList[test.loc]
	args, err := test.Macro.ValidateArgs(cmdNode.(*Cmd), f.GetDocs()[0])
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
