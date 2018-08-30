// Copyright 2018 Kevin Kenan
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy
// of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package core

import (
	"fmt"
	"strings"
	"testing"
)

// Test SelectArguments -------------------------------------------------------
// The initial opening paragraph command is skipped in these tests.

type selectArgumentsTestCase struct {
	name        string
	command     Document
	loc         int // The index of the command in root.NodeList
	reqParams   []string
	optParams   []string
	expSelected []string
	expUnknown  []string
	expMissing  []string
}

var selectArgumentsTestCases = []selectArgumentsTestCase{
	{"no args", newPlainTestDoc("•X[]"), 0, []string{"one"}, nil,
		nil, nil, []string{"one"}},
	{"no args and no parameters", newPlainTestDoc("•X[]"), 0, nil, nil,
		nil, nil, nil},
	{"empty bare arg", newPlainTestDoc("•X{}"), 0, []string{"one"}, nil,
		[]string{"one"}, nil, nil},
	{"empty anonymous arg", newPlainTestDoc("•X[{}]"), 0, []string{"one"}, nil,
		[]string{"one"}, nil, nil},
	{"anonymous arg with missing args", newPlainTestDoc("•X{a}"), 0, []string{"one", "two", "three"}, nil,
		nil, nil, []string{"two", "three"}},
	{"anonymous args with one unknown", newPlainTestDoc("•X[{a}{b}]"), 0, []string{"one"}, nil,
		nil, []string{"#2"}, nil},
	{"anonymous with only optional args", newPlainTestDoc("•X[{abc}{xyz}]"), 0, nil, []string{"one", "two"},
		[]string{"one", "two"}, nil, nil},
	{"anonymous with an optional arg", newPlainTestDoc("•X[{a}{b}]"), 0, []string{"one"}, []string{"two"},
		[]string{"one", "two"}, nil, nil},
	{"anonymous with optional and unknown args", newPlainTestDoc("•X[{a}{b}{c}]"), 0, []string{"one"}, []string{"two"},
		nil, []string{"#3"}, nil},
	{"named args", newPlainTestDoc("•X[one={a}]"), 0, []string{"one"}, nil,
		[]string{"one"}, nil, nil},
	{"named and optional args", newPlainTestDoc("•X[one={a} two={b}]"), 0, []string{"one"}, []string{"two"},
		[]string{"one", "two"}, nil, nil},
	{"named with unknown args", newPlainTestDoc("•X[one={a} two={b}]"), 0, []string{"one"}, nil,
		nil, []string{"two"}, nil},
	{"named with missing args", newPlainTestDoc("•X[one={a}]"), 0, []string{"one", "three"}, nil,
		nil, nil, []string{"three"}},
	{"named with unknown and optional args", newPlainTestDoc("•X[one={a} two={b}]"), 0, []string{"one"}, []string{"three"},
		nil, []string{"two"}, nil},
	{"named with unknown and missing args", newPlainTestDoc("•X[three={a} one={b}]"), 0, nil, []string{"one", "three"},
		[]string{"one", "three"}, nil, nil},
	{"named with no parameters at all", newPlainTestDoc("•X[one={a} two={b}]"), 0, nil, nil,
		nil, []string{"one", "two"}, nil},
}

func testSelectArguments(t *testing.T, test *selectArgumentsTestCase) {
	f := NewFolio()
	f.AddMacro(NewMacro("X", "", nil, nil))
	f.AddMacro(NewMacro("sys.Z", "", nil, nil))
	f.AppendDoc(&test.command)

	root, err := Parse(f.GetDocs()[0])

	if err != nil {
		t.Errorf("%s: Parse failed: %s", test.name, err)
		return
	}

	if root == nil {
		t.Errorf("%s: root is nil", test.name)
		return
	}

	if len(root.NodeList) < test.loc-1 {
		t.Errorf("%s: loc (%d) is beyond the end of root.NodeList", test.name, test.loc)
		return
	}

	cmdNode := root.NodeList[test.loc]
	_, ok := cmdNode.(*Cmd)

	if !ok {
		t.Errorf("%s: node at loc %d is not a Cmd node", test.name, test.loc)
		return
	}

	selected, unknown, missing := cmdNode.(*Cmd).SelectArguments(test.reqParams, test.optParams)
	// fmt.Printf("%s: s: %v ; u: %v ; m: %v\n", test.name, selected, unknown, missing)

	if !checkNodeMapKeys(selected, test.expSelected) || !checkStringSlices(unknown, test.expUnknown) || !checkStringSlices(missing, test.expMissing) {
		t.Errorf("%s\n  *result: %v, %v, %v\n*expected: %v, %v, %v", test.name,
			names(selected), unknown, missing,
			test.expSelected, test.expUnknown, test.expMissing)
		// t.Errorf("%s\n  wrong arguments selected\n  *expected: %v\n       *got: %v", test.name, test.expSelected, names(selected))
	}
}

func TestSelectArguments(t *testing.T) {
	for _, test := range selectArgumentsTestCases {
		testSelectArguments(t, &test)
	}
}

func names(nodes NodeMap) (names []string) {
	names = []string{}
	for k := range nodes {
		names = append(names, k)
	}
	return
}

// checkNodeMapKeys returns true if all the keys specified in the []string
// argument are in the given NodeMap.
func checkNodeMapKeys(nodes NodeMap, keys []string) bool {
	if len(nodes) != len(keys) {
		return false
	}
	for _, k := range keys {
		_, ok := nodes[k]
		// fmt.Println("here: " + k)
		if !ok {
			return false
		}
	}
	return true
}

func checkStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if isSubset(a, b) {
		// We know len(a) == len(b), so subset implies equality.
		return true
	}
	return false
}

// isSubset returns true if all members of a are in b.
func isSubset(a, b []string) bool {
	for _, av := range a {
		found := false
		for _, bv := range b {
			if av == bv {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Test Parsing ---------------------------------------------------------------
// The initial opening paragraph command is skipped in these tests.

type parseTestCase struct {
	name string
	// input    string
	doc      Document
	expOut   string
	expErr   bool
	expNodes []string
}

func newPlainTestDoc(s string) Document {
	return Document{
		Name:        "<stdin>",
		Path:        "<stdin>",
		Text:        s,
		Plain:       true,
		Initialized: true,
	}
}

var parseTestCases = []parseTestCase{
	{"empty", newPlainTestDoc(""), "", false, []string{}},
	{"basic", newPlainTestDoc("test"), "test", false, []string{"Text Node"}},
	{"contains three literals", newPlainTestDoc("test ``, `• and `◊."), "test `, • and ◊.", false, []string{
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node"}},
	{"basic with linebreak", newPlainTestDoc("test\nline two"), "test\nline two", false, []string{
		"Text Node",
		"Text Node",
		"Text Node"}},
	{"bare command", newPlainTestDoc("1 •X 3"), "1 •X[] 3", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"empty full command", newPlainTestDoc("1 •X[] 3"), "1 •X[] 3", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"simple empty command", newPlainTestDoc("1 •X{} 3"), "1 •X[{}] 3", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"two anonymous command args", newPlainTestDoc("1 •X[{a}{b}] 3"), "1 •X[{a}{b}] 3", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"flags", newPlainTestDoc("1 •X[<34 \n5=6>]7"), "1 •X[<34,5=6>]7", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"anonymous command on different lines", newPlainTestDoc("1 •X[{a}\n{•X{c}}] 4"), "1 •X[{a}{•X[{c}]}] 4", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"anonymous command with line breaks", newPlainTestDoc("1 •X[{a\nb} {c\nd}] 4"), "1 •X[{a\nb}{c\nd}] 4", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"two named args on different lines", newPlainTestDoc("1 •X[x={a}\ny={b}]3"), "1 •X[x={a}y={b}]3", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"complex context with named args", newPlainTestDoc("1 •X[1={a}2={b •X{c}}] 4"), "1 •X[1={a}2={b •X[{c}]}] 4", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"context with comment", newPlainTestDoc("•X[◊\n{a}\n] 4"), "•X[{a}] 4", false, []string{
		"Cmd Node",
		"Text Node"}},
	{"line breaks", newPlainTestDoc("\n\n1\n\n2\n\n3\n"), "\n\n1\n\n2\n\n3\n", false, []string{
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node",
		"Text Node"}},
	// {"line breaks with parscan flag on", "¶+\n\n1\n\n2\n\n3\n", "•sys.paragraph.begin[<>{}]1•sys.paragraph.end[<>{\n\n}]•sys.paragraph.begin[<>{}]2•sys.paragraph.end[<>{\n\n}]•sys.paragraph.begin[<>{}]3•sys.paragraph.end[<>{\n}]", false, []string{
	// 	"Text Node",
	// 	}},

	// SysCmd tests
	{"SysCmd", newPlainTestDoc("test•(Z)now"), "test•sys.Z[]now", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"SysCmd advanced", newPlainTestDoc("test•(Z){that}now"), "test•sys.Z[{that}]now", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"SysCmd with linebreak", newPlainTestDoc("test•(Z){\nthat}now"), "test•sys.Z[{\nthat}]now", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},
	{"SysCmd", newPlainTestDoc("test•(Z)%  \n now"), "test•sys.Z[]now", false, []string{
		"Text Node",
		"Cmd Node",
		"Text Node"}},

	// Error tests
	// {"basic kv command", newPlainTestDoc("1•"), "", 0, true, false},
}

func newParTestDoc(s string) Document {
	return Document{
		Name:        "testdoc",
		Path:        "testpath",
		Text:        s,
		Plain:       false,
		Initialized: true,
	}
}

var parParseTestCases = []parseTestCase{
	// 	// Paragaph tests
	{"basic paragraph", newParTestDoc("one\n\ntwo"),
		"•paragraph.begin[]one•paragraph.end[]•paragraph.begin[]two•paragraph.end[]", false, []string{
			"Cmd Node",
			"Text Node",
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node"}},
	{"simple command with paragraph", newParTestDoc("1•X{A\n\nB}3"),
		"•paragraph.begin[]1•X[{A•paragraph.end[]•paragraph.begin[]B}]3•paragraph.end[]", false, []string{
			"Cmd Node",
			"Text Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node"}},
	{"simple command with spaces", newParTestDoc("•X{}    \n\nA"),
		"•paragraph.begin[]•X[{}]    •paragraph.end[]•paragraph.begin[]A•paragraph.end[]", false, []string{
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node"}},
	{"new macro", newParTestDoc(
		`•(newmacro){
name: section
block: true
parameters: ["text"]
template: "((.text))"
}•section{a}`),
		"•section[{a}]", false, []string{
			"Cmd Node"}},
}

func newFlowTestDoc(s string) Document {
	return Document{
		Name:        "testdoc",
		Path:        "testpath",
		Text:        s,
		Reflow:      true,
		Initialized: true,
	}
}

var flowParseTestCases = []parseTestCase{
	{"flow text", newFlowTestDoc("    A    \nB"),
		"•paragraph.begin[]A B•paragraph.end[]", false, []string{
			"Cmd Node",
			"Text Node",
			"Text Node",
			"Text Node",
			"Cmd Node"}},
	{"flow simple command arg with spaces", newFlowTestDoc("•X{arg}    \n\nA"),
		"•paragraph.begin[]•X[{arg}]•paragraph.end[]•paragraph.begin[]A•paragraph.end[]", false, []string{
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node"}},
	{"flow simple command with spaces", newFlowTestDoc("•X{}    \n\nA"),
		"•paragraph.begin[]•X[{}]•paragraph.end[]•paragraph.begin[]A•paragraph.end[]", false, []string{
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node",
			"Cmd Node",
			"Text Node",
			"Cmd Node"}},
}

func TestPlainParse(t *testing.T) {
	tnum := -1
	var start, end = 0, len(parseTestCases)
	if tnum > 0 {
		start = tnum
		end = tnum + 1
	}
	// opts := &Options{Plain: true}
	// d := NewDoc("testdoc", "testpath")
	// d.Plain = true
	testParse(t, parseTestCases, start, end)
}

func TestParse(t *testing.T) {
	tnum := -1
	var start, end = 0, len(parParseTestCases)
	if tnum > 0 {
		start = tnum
		end = tnum + 1
	}
	// opts := &Options{Plain: false}
	// d := NewDoc("testdoc", "testpath")
	testParse(t, parParseTestCases, start, end)
}

func TestFlowParse(t *testing.T) {
	tnum := -1
	var start, end = 0, len(flowParseTestCases)
	if tnum >= 0 {
		start = tnum
		end = tnum + 1
	}
	// opts := &Options{Plain: false, Reflow: true}
	// d := NewDoc("testdoc", "testpath")
	// d.Reflow = true
	testParse(t, flowParseTestCases, start, end)
}

func testParse(t *testing.T, tests []parseTestCase, start, end int) {
	for tc, test := range tests[start:end] {
		var (
			result *Section
			err    error
		)

		f := NewFolio()
		f.AddMacro(NewMacro("X", "", nil, nil))
		f.AddMacro(NewMacro("sys.Z", "", nil, nil))
		f.AppendDoc(&test.doc)

		result, err = Parse(f.GetDocs()[0])
		if err != nil {
			// fmt.Printf("%s: %q -> error: %q\n", test.name, test.input, err)
			t.Errorf("%s: %s\n", test.name, err)
			return
		}
		nodes := getNodes(result)
		eqNodes := nodeListTypeEqual(nodes, test.expNodes)
		if test.expErr && err == nil {
			t.Errorf("%s\n  *result: unexpected success", test.name)
		}
		if !test.expErr && err != nil {
			t.Errorf("%s\n  *result: unexpected error: %s", test.name, err)
		}
		if err == nil {
			s := result.NodeList.String()
			if s != test.expOut {
				t.Errorf("[%d] %s\n  *result:   %q\n  *expected: %q\n", tc, test.name, s, test.expOut)
			}
			if !eqNodes {
				t.Errorf("[%d] %s\n  *result:   %s\n  *expected: %s", tc, test.name, joinList(nodes), joinList(test.expNodes))
			}
		}
	}
}

func joinList(l []string) string {
	w := new(strings.Builder)
	for _, s := range l {
		w.WriteString(fmt.Sprintf("\n      %s", s))
	}
	return w.String()
}

func getNodes(root *Section) []string {
	nodes := []string{}
	c := make(chan string)
	go root.WalkS(c)
	// fmt.Printf(":: %q\n", root.String())
	// fmt.Printf("> Root Section Node: contains %d nodes\n", len(root.NodeList))
	for s := range c {
		nodes = append(nodes, s)
	}
	return nodes
}

func nodeListTypeEqual(n1, n2 []string) bool {
	if len(n1) != len(n2) {
		return false
	}
	for k := range n1 {
		if !nodeTypeEqual(n1[k], n2[k]) {
			return false
		}
	}
	return true
}

func nodeTypeEqual(node, ntype string) bool {
	return strings.HasPrefix(node, ntype+":")
}
