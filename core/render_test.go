package core

import (
	"testing"
)

func TestRenderText(t *testing.T) {
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = "hello\n\nworld"
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<hello>\n<world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderEcho(t *testing.T) {
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = "•echo{hello\n\nworld}"
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<hello>\n<world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderNestedEcho(t *testing.T) {
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = "•echo{•echo{hello}\n\n•echo{world}}"
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<hello>\n<world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderEchoBlock(t *testing.T) {
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = "A\n\n•Echo{hello\n\nworld}\nB"
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<A>\nhello\n\nworld\n<B>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderNestedMacro(t *testing.T) {
	testText := `
•(newmacro*){
    name: test
    parameters: ["p"]
    template: "•echo{hello} w[[ .p ]]d"
*}

•test{orl}
`
	// testText = fmt.Sprintf(testText, "`")
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = testText
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<hello world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderBlockMacro(t *testing.T) {
	testText := `
•(newmacro){
    name: test
    block: true
    parameters: ["p"]
    template: h[[ .p ]]d
}

A

•test{ello worl}

B
`
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = testText
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<A>\nhello world\n<B>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderTitle(t *testing.T) {
	testText := `>>>
title: hello
---
•(newmacro){
    name: greet
    template: "[[ .Doc.Title ]]"
}

•greet world
`
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = testText
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<hello world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderPlain(t *testing.T) {
	testText := `>>>
title: hello
mode: plain
---
hello world
`
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = testText
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "hello world\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderData(t *testing.T) {
	testText := `
•(exec){[[ setdata "greeting" "hello" ]]}
•(newmacro){
    name: greet
    template: '[[- getdata "greeting" "howdy" -]]'
}

•greet world
`
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = testText
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<hello world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderDataDirect(t *testing.T) {
	testText := `
•(exec){[[ setdata "greeting" "hello" ]]}
•(newmacro){
    name: greet
    template: '[[ index .Data "greeting" ]]'
}

•greet world
`
	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = testText
	f.AppendDoc(d)
	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	exp := "<hello world>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderExecAccess(t *testing.T) {
	var err error
	macrodef := `•(exec){[[ setdata "greeting" "hello" ]]}`
	doctext := `•(exec){[[ .Data.greeting ]]}`

	f := NewFolio()
	err = f.loadMacros("macrodef", "", macrodef)
	if err != nil {
		t.Errorf("loadMacros: unexepected error: %s", err)
	}

	d := NewDoc("testname", "testpath")
	d.Text = doctext
	f.AppendDoc(d)

	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "hello"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderExplicitPar(t *testing.T) {
	var err error
	doctext := `First•paragraph.end[]Second`

	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = doctext
	f.AppendDoc(d)
	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<First>\n<Second>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderExplicitParBlock(t *testing.T) {
	var err error
	doctext := `First•paragraph.end[]•Echo{Second}`

	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = doctext
	f.AppendDoc(d)
	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<First>\nSecond\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}

func TestRenderLiteralCmd(t *testing.T) {
	var err error
	doctext := "•echo{`• hi}"

	f := NewFolio()
	d := NewDoc("testname", "testpath")
	d.Text = doctext
	f.AppendDoc(d)
	out, err := f.MakeDocs()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	exp := "<• hi>\n"
	if out != exp {
		t.Errorf("\nExpected: %q\n     Got: %q", exp, out)
	}
}
