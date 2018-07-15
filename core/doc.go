package core

import (
	"strings"
	"time"

	"github.com/kevinkenan/cobra"
)

// Folio is a collection of documents.
type Folio struct {
	Documents map[DocFile]Document
	Data      map[string]interface{}
	Macros    MacroMap
}

func NewFolio() *Folio {
	return &Folio{
		Documents: make(map[DocFile]Document),
		Data:      make(map[string]interface{}),
		Macros:    NewMacroMap(),
	}
}

// Append adds the document to the folio.
func (f *Folio) Append(d *Document) {
	if d.Name == "" || d.Path == "" {
		panic("missing Name or Path when appending doc")
	}
	d.Folio = f
	f.Documents[DocFile{FileName: d.Name, FilePath: d.Path}] = *d
}

func (f *Folio) GetMacro(name, format string) *Macro {
	return f.Macros.GetMacro(name, format)
}

// AddMacro adds a single Macro to the map.
func (f *Folio) AddMacro(m *Macro) {
	f.Macros.AddMacro(m)
}

// AddMacros merges the MacroMap passed as an argument into Folio's MacroMap.
func (f *Folio) AddMacros(mm MacroMap) {
	f.Macros.AddMacros(mm)
}

func (f *Folio) Make() (s string, err error) {
	ds := []string{}
	// w := new(strings.Builder)

	for _, d := range f.Documents {
		r := &Render{Doc: &d}
		var made string

		made, err = MakeWith(r)
		if err != nil {
			return
		}
		ds = append(ds, made)
	}

	s = strings.Join(ds, "\n")
	return
}

// GetDocs returns the Folio's documents in a slice.
func (f *Folio) GetDocs() (docs []*Document) {
	for _, d := range f.Documents {
		docs = append(docs, &d)
	}
	return
}

// DocFile represents the location of a text file to be processed.
type DocFile struct {
	FileName string
	FilePath string
}

// Document represents a file of text to be processed. The fields are mostly
// populated from the file's metadata.
type Document struct {
	Folio        *Folio            // The folio that contains this document
	Name         string            // Name of the file
	Path         string            // The file system path to the file
	Title        string            // The title of the document
	OutputName   string            // The name of the output file
	Date         time.Time         // The date of the document
	Ignore       bool              // If true, this file is not included in the output
	Rendered     bool              // True when the document has been rendered and output
	Packages     []string          //
	Output       string            // The rendered output
	Targets      []string          //
	Metadata     map[string]string //
	Text         string            // The raw text of the file
	contentBegin int               // The index in Text where the config ends and the content begins
	Root         *Section          // The root node of the parsed content
	Plain        bool              // Don't generate paragraphs or aggressively eat whitespace
	Reflow       bool              // if true, remove new lines and collapse whitespace in paragraphs
	Format       string            // The format (html, latex, etc.) is used to select the right macro
}

// NewDoc creates a new Document and initializes the macrosIn field.
func NewDoc(name, path string) *Document {
	d := Document{Name: name, Path: path}
	return &d
}

// Make renders the document.
func (d *Document) Make() (s string, err error) {
	r := &Render{Doc: d}
	// s, err = MakeWith(d.Text, r, d.Options)
	s, err = MakeWith(r)
	return
}

// MakeWith allows arbitrary text to be processed with an existing Render
// context. Most of the time the Document's Make is used (which calls
// MakeWith), but MakeWith itself is useful for handling macros embedded in
// templates.
func MakeWith(r *Render) (s string, err error) {
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

	root, err := Parse(r.Doc)
	if err != nil {
		return "", err
	}

	//r.addMacros(macros)
	cobra.LogV("rendering (render)")
	return r.render(root), nil
}
