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

func TestRenderMacro(t *testing.T) {
	testText := `
•(newmacro){
    name: test
    parameters: ["p"]
    template: w[[ .p ]]d
}

hello •test{orl}
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
	//•(exec){[[ setdata "greeting" "hello" ]]}
	macrodef := `•(exec){[[ setdata "greeting" "hello" ]]}`
	doctext := `•(exec){[[ .Data.greeting ]]}`

	f := NewFolio()
	err = f.loadMacros("macrodef", "", macrodef)
	if err != nil {
		t.Errorf("loadMacros: unexepected error: %s", err)
	}
	// fname := "testmacros"
	// stmdoc := NewDoc(fname, "testpath")
	// stmdoc.Folio = f
	// ParseMacro(fname, macrodef, stmdoc)

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
