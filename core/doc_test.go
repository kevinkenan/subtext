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
	// "github.com/kevinkenan/subtext/verbose"
	// "strings"
	"github.com/kevinkenan/cobra"
	// "testing"
)

func init() {
	cfg := cobra.NewTestingConfig(nil)
	cfg.LogPanicOnly()
	// cfg := cobra.NewTestingConfig([]string{"macro","parse","node","scan"})
	_ = cfg
	// cfg.SetDefault("logalltags", true)
}

// Stop fmt complaints --------------------------------------------------------

func stopFmtComplaints() {
	fmt.Println("")
}

// Make Test ------------------------------------------------------------------

// func TestMake(t *testing.T) {
// 	var rƒ func(string) *Render
// 	rƒ = mockDocOne
// 	testMake(t, rƒ, false, "basic bare", "•test{abc}", "Hi abc.")
// 	testMake(t, rƒ, false, "basic anonymous", "•test[{abc}]", "Hi abc.")
// 	testMake(t, rƒ, false, "basic named", "•test[first={abc}]", "Hi abc.")

// 	rƒ = mockDocTwo
// 	testMake(t, rƒ, false, "basic bare with default optional", "•test{abc}", "Hi abcdef.")
// 	testMake(t, rƒ, false, "basic anonymous with default optional", "•test[{abc}]", "Hi abcdef.")
// 	testMake(t, rƒ, false, "basic named with default optional", "•test[first={abc}]", "Hi abcdef.")
// 	testMake(t, rƒ, false, "basic anonymous with optional", "•test[{abc}{xyz}]", "Hi abcxyz.")
// 	testMake(t, rƒ, false, "basic named with optional", "•test[first={abc} second={xyz}]", "Hi abcxyz.")

// 	rƒ = mockDocThree
// 	testMake(t, rƒ, false, "nested macros", "1•a{x •b{y •c{z}}}2", "1<a>x <b>y <c>z</c></b></a>2")
// 	testMake(t, rƒ, false, "paragraph mode off (default initial setting)", "1\n \n2", "1\n \n2")

// 	rƒ = mockDocFour
// 	testMake(t, rƒ, false, "pararagraph mode on", "1\n \n2", "\n1\n\n2\n")
// 	testMake(t, rƒ, false, "initial pararagraphs mode on", "\n\n     \n1\n \n2", "\n1\n\n2\n")
// 	testMake(t, rƒ, false, "whitespace with pararagraph mode on", "1\n  \n2\n   \n\n3", "\n1\n\n2\n\n3\n")
// 	testMake(t, rƒ, false, "final pararagraph", "1\n  \n2\n   \n\n          ", "\n1\n\n2\n")

// 	rƒ = mockDocFive
// 	testMake(t, rƒ, false, "custom pararagraph markers", "1\n\n \n2\n\n3", "<p>1</p>\n<p>2</p>\n<p>3</p>\n")
// 	testMake(t, rƒ, false, "custom final pararagraph simple", "1\n  \n \n \n \n", "<p>1</p>\n")
// 	testMake(t, rƒ, false, "custom final pararagraph", "1\n  \n2\n   \n\n          ", "<p>1</p>\n<p>2</p>\n")
// 	testMake(t, rƒ, false, "custom initial pararagraphs mode on", "\n\n     \n1\n \n2", "<p>1</p>\n<p>2</p>\n")
// 	testMake(t, rƒ, false, "custom pararagraphs with command", "1\n\n•a{2\n\n3}4", "<p>1</p>\n<p>2</p>\n<p>34</p>\n")
// 	testMake(t, rƒ, false, "custom par with vertical command", "1\n\n§pre{\n\n2\n\n3\n\n}4", "<p>1</p>\n<pre>\n\n2\n\n3\n\n</pre>\n<p>4</p>\n")
// 	testMake(t, rƒ, false, "custom par with horizontal command", "1\n\n•pre{2}3", "<p>1</p>\n<p><pre>2</pre>\n3</p>\n")
// }

// func testMake(t *testing.T, rƒ func(string) *Render, expErr bool, name, command, exp string) {
// 	testMakeFull(t, rƒ, expErr, name, command, exp, false)
// }

// func testMakeV(t *testing.T, rƒ func(string) *Render, expErr bool, name, command, exp string) {
// 	testMakeFull(t, rƒ, expErr, name, command, exp, false)
// }

// func testMakeFull(t *testing.T, rƒ func(string) *Render, expErr bool, name, command, exp string, verb bool) {
// 	r := rƒ("") // mockDoc(command)
// 	// fmt.Printf("%s: %v\n", name, r.macros["paragraph.begin"].String())
// 	s, err := MakeWith(command, r, r.macrosIn)
// 	// fmt.Printf("%q\n", s)
// 	switch {
// 	case err != nil && !expErr:
// 		t.Errorf("%s\n  unexpected failure: %s", name, err)
// 	case err != nil && expErr:
// 		if err.Error() != exp {
// 			t.Errorf("%s\n  *expected: %s\n       *got: %s", name, exp, err)
// 		}
// 	case err == nil && expErr:
// 		t.Errorf("%s: unexpected success", name)
// 	case err == nil && !expErr:
// 		if s != exp {
// 			t.Errorf("%s\n  *expected: %q\n       *got: %q", name, exp, s)
// 		}
// 	}
// }

// func mockDocOne(input string) *Render {
// 	var m *parse.Macro
// 	// d := Document{macros: make(map[string]*parse.Macro)}
// 	d := NewDoc()
// 	m = parse.NewMacro("test", "Hi {{.first}}.", []string{"first"}, nil)
// 	d.macrosIn = append(d.macrosIn, m)
// 	d.Text = input
// 	r := &Render{Document: d}
// 	return r
// }

// func mockDocTwo(input string) *Render {
// 	var m *parse.Macro
// 	// d := Document{macros: make(map[string]*parse.Macro)}
// 	d := NewDoc()
// 	opt := parse.Optional{Name: "second", Default: "def"}
// 	m = parse.NewMacro("test", "Hi {{.first}}{{.second}}.", []string{"first"}, []*parse.Optional{&opt})
// 	d.macrosIn = append(d.macrosIn, m)
// 	d.Text = input
// 	r := &Render{Document: d}
// 	return r
// }

// func mockDocThree(input string) *Render {
// 	var m *parse.Macro
// 	// opt := parse.Optional{Name: "second", Default: "def"}
// 	// d := Document{macros: make(map[string]*parse.Macro)}
// 	d := NewDoc()
// 	m = parse.NewMacro("a", "<a> {{- .first -}} </a>", []string{"first"}, nil)
// 	d.macrosIn = append(d.macrosIn, m)
// 	m = parse.NewMacro("b", "<b>{{.first}}</b>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macrosIn = append(d.macrosIn, m)
// 	m = parse.NewMacro("c", "<c>{{.first}}</c>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macrosIn = append(d.macrosIn, m)
// 	d.Text = input
// 	r := &Render{Document: d}
// 	return r
// }

// func mockDocFour(input string) *Render {
// 	d := NewDoc()
// 	d.Text = input
// 	r := &Render{Document: d}
// 	r.ParagraphMode = true
// 	return r
// }

// func mockDocFive(input string) *Render {
// 	d := NewDoc()
// 	d.AddMacro(parse.NewMacro("paragraph.begin", "<p>", []string{"orig"}, nil))
// 	d.AddMacro(parse.NewMacro("paragraph.end", "</p>\n", []string{"orig"}, nil))
// 	d.AddMacro(parse.NewMacro("a", "{{.body}}", []string{"body"}, nil))
// 	d.AddMacro(parse.NewMacro("pre", "<pre>{{.body}}</pre>\n", []string{"body"}, nil))
// 	d.Text = input
// 	r := &Render{Document: d}
// 	r.ParagraphMode = true
// 	return r
// }
