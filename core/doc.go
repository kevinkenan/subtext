package core

import (
	"github.com/kevinkenan/cobra"
)

// Document represents the text being processed.
type Document struct {
	Name       string
	Packages   []string
	Output     string
	Targets    []string
	Metadata   map[string]string
	Text       string
	Root       *Section
	Options    *Options
	Plain      bool // Don't generate paragraphs or aggressively eat whitespace
	ReflowPars bool // if true, remove new lines and collapse whitespace in paragraphs
	macrosIn   []*Macro
}

func NewDoc() *Document {
	d := Document{macrosIn: []*Macro{}}
	return &d
}

func (d *Document) AddMacro(m *Macro) {
	d.macrosIn = append(d.macrosIn, m)
}

func (d *Document) Make() (s string, err error) {
	r := &Render{Document: d, ParagraphMode: !d.Plain}
	s, err = MakeWith(d.Text, r, d.Options)
	return
}

// MakeWith allows arbitrary text to be processed with an existing Render
// context. Most of the time the Document's Make is used (which calls
// MakeWith), but MakeWith itself is useful for handling macros embedded in
// templates.
func MakeWith(t string, r *Render, options *Options) (s string, err error) {
	defer func() { cobra.LogV("finished rendering") }()
	defer func() {
		if e := recover(); e != nil {
			switch e.(type) {
			case RenderError, Error:
				err = e.(error)
			default:
				panic(e)
			}
		}
	}()

	root, macros, err := Parse(r.Name, t, options)
	if err != nil {
		return "", err
	}

	r.macros = macros
	cobra.LogV("rendering (render)")
	return r.render(root), nil
}
