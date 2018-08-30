package core

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/kevinkenan/cobra"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/pflag"
	yaml "gopkg.in/yaml.v2"
)

// Folio is a collection of documents.
type Folio struct {
	Documents       map[DocFile]Document
	Data            map[string]interface{}
	Macros          MacroMap
	Packages        []string          // The requested list of macro packages
	LoadedPackages  map[string]bool   // List of all the loaded packages
	PkgSearchPaths  []string          // Where to look for macro packages
	PkgSearchIndex  int               // Where to begin searching next
	PkgLocations    map[string]string // Paths to all the known packages
	Cmd             *cobra.Command    // The CLI command that created the Folio
	defaultWarnings map[string]bool   // Map of all default macro warnings
}

func NewFolio() (f *Folio) {
	userhome, err := homedir.Dir()
	if err != nil {
		cobra.Outf("Warning: unable to locate home directory: %s", err)
	}

	userpkg := filepath.Join(userhome, ".subtext", "packages")

	f = &Folio{
		Documents:       make(map[DocFile]Document),
		Data:            make(map[string]interface{}),
		Macros:          NewMacroMap(),
		Packages:        []string{},
		LoadedPackages:  make(map[string]bool),
		PkgSearchPaths:  []string{"packages", userpkg},
		PkgLocations:    make(map[string]string),
		defaultWarnings: make(map[string]bool),
	}

	funcMap["setdata"] = f.SetData
	funcMap["getdata"] = f.GetData
	funcMap["indata"] = f.InData

	return
}

// CheckFlag returns true if the flag fname was explicitly set on the command
// line.
func (f *Folio) CheckFlag(fname string) (flagged bool) {
	if f.Cmd == nil {
		return
	}

	f.Cmd.Flags().Visit(func(flag *pflag.Flag) {
		if flag.Name == fname {
			flagged = true
		}
	})

	return
}

// AppendDoc initializes the document and adds it to the folio.
func (f *Folio) AppendDoc(d *Document) error {
	if d.Name == "" || d.Path == "" {
		fmt.Errorf("missing Name or Path when appending doc %q", d.Name)
	}

	d.Folio = f

	if err := d.initDoc(); err != nil {
		return err
	}

	f.Documents[DocFile{FileName: d.Name, FilePath: d.Path}] = *d
	return nil
}

// LoadPackages finds the requested packages and loads their macros.
func (f *Folio) LoadPackages(pkgs []string) error {
	var err error
	var pkgpath string

	// Search for the package.
search:
	for _, p := range pkgs {
		pkgname := strings.TrimSuffix(p, ".stm")

		if f.LoadedPackages[pkgname] {
			continue
		}

		for {
			if pkgp, found := f.PkgLocations[pkgname]; found {
				pkgpath = pkgp
				break
			}

			done, err := f.readNextPackageDir()
			if err != nil {
				return err
			}

			if done {
				cobra.Outf("unable to find package %q", p)
				continue search
			}
		}

		err = f.readMacroPkg(pkgpath)
		if err != nil {
			return err
		}

		f.LoadedPackages[pkgname] = true
	}

	return nil
}

// readNextPackageDir reads the package names in the next directory listed in
// PkgSearchPaths. If it returns true, then the search is done and there are
// no more directories to search.
func (f *Folio) readNextPackageDir() (bool, error) {
	var files []os.FileInfo

	i := f.PkgSearchIndex
	if i >= len(f.PkgSearchPaths) {
		return true, nil
	}

	pkgd := filepath.Clean(f.PkgSearchPaths[i])
	cobra.WithField("directory", pkgd).LogV("searching package path")

	finfo, err := os.Stat(pkgd)
	if err != nil {
		cobra.LogfV("unable to read directory %q: %s", pkgd, err)
		f.PkgSearchIndex++
		return false, nil
	}

	if !finfo.IsDir() {
		return true, fmt.Errorf("package path %q is not a directory", pkgd)
	}

	files, err = ioutil.ReadDir(pkgd)
	if err != nil {
		return true, fmt.Errorf("reading package directory: %s", err)
	}

	for _, fi := range files {
		fp := filepath.Join(pkgd, fi.Name())

		if fi.IsDir() {
			f.PkgLocations[fi.Name()] = fp
		} else {
			switch filepath.Ext(fp) {
			case ".stm":
				f.PkgLocations[strings.TrimSuffix(fi.Name(), ".stm")] = fp
			default:
				continue
			}
		}
	}

	f.PkgSearchIndex++

	return false, nil
}

func (f *Folio) readMacroPkg(pkgpath string) error {
	// var err error
	// var files []os.FileInfo

	finfo, err := os.Stat(pkgpath)
	if err != nil {
		return err
	}

	if finfo.IsDir() {
		files, err := ioutil.ReadDir(pkgpath)
		if err != nil {
			return fmt.Errorf("unable to read directory %q: %s", pkgpath, err)
		}

		for _, fi := range files {
			fp := filepath.Join(pkgpath, fi.Name())

			if fi.IsDir() {
				// skip nested directories
				continue
			}

			switch filepath.Ext(fp) {
			case ".stm":
				err = f.readMacros(fp)
				if err != nil {
					return err
				}
			default:
				continue
			}
		}
	} else {
		err = f.readMacros(pkgpath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *Folio) readMacros(fpath string) (err error) {
	fname := filepath.Base(fpath)
	fin, err := ioutil.ReadFile(fpath)
	if err != nil {
		return
	}

	return f.loadMacros(fname, fpath, string(fin))
}

func (f *Folio) loadMacros(fname, fpath, fin string) (err error) {
	doc := NewDoc(fname, fpath)
	doc.Text = fin
	doc.Plain = true
	doc.Folio = f

	_, err = MakeWith(&Render{Doc: doc})
	cobra.Tag("doc").WithField("package", fname).LogV("loaded package")

	return
}

func (f *Folio) GetMacro(name, format string) (mac *Macro) {
	mac, found := f.Macros.GetMacro(name, format)
	if mac == nil {
		return
	}

	warned := f.defaultWarnings[name]
	if !found && cobra.GetBool("default-warnings") && !warned {
		cobra.Outf("warning: default macro used: %q", name)
		f.defaultWarnings[name] = true
	}

	return
}

func (f *Folio) GetSysMacro(name, format string) (mac *Macro) {
	mac, _ = f.Macros.GetMacro(name, format)
	return
}

// AddMacro adds a single Macro to the map.
func (f *Folio) AddMacro(m *Macro) {
	f.Macros.AddMacro(m)
}

// AddMacros merges the MacroMap passed as an argument into Folio's MacroMap.
func (f *Folio) AddMacros(mm MacroMap) {
	f.Macros.AddMacros(mm)
}

func (f *Folio) MakeDocs() (s string, err error) {
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

func (f *Folio) SetData(key string, val interface{}) string {
	f.Data[key] = val
	return ""
}

func (f *Folio) InData(key string) bool {
	_, found := f.Data[key]
	return found
}

func (f *Folio) GetData(key, dflt string) (interface{}, error) {
	keys := strings.Split(key, ".")
	var vals interface{}
	var found bool
	vals = f.Data

	for _, k := range keys {
		switch vals.(type) {
		case map[string]interface{}:
			vals, found = vals.(map[string]interface{})[k]
			if !found {
				return dflt, nil
			}
		case map[interface{}]interface{}:
			vals = vals.(map[interface{}]interface{})[k]
		default:
			vals = dflt
			return vals, nil
			// return "", nil
			// return nil, fmt.Errorf("key %q not found", key)
		}
	}

	return vals, nil
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
	Template     string            // The name of the wrapper template
	Date         time.Time         // The date of the document
	Ignore       bool              // If true, this file is not included in the output
	Rendered     bool              // True when the document has been rendered and output
	Packages     []string          // List of packages to add.
	Output       string            // The rendered output
	Targets      []string          //
	Metadata     map[string]string //
	Text         string            // The raw text of the file
	contentBegin int               // The index in Text where the config ends and the content begins
	Initialized  bool              // True if the document has already been initialized
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

// initDoc loads the file and processes the config section if present.
func (d *Document) initDoc() (err error) {
	if d.Initialized {
		return nil
	}

	if d.Text == "" {
		err = d.loadText()
		if err != nil {
			return
		}
	}

	if len(d.Text) < 3 || d.Text[:3] != ">>>" {
		return nil
	}

	confEnd := 3 + strings.Index(d.Text, "---\n")
	if confEnd == -1 {
		return fmt.Errorf("missing end to config section in %q", d.Name)
	}

	d.contentBegin = confEnd + 1
	cfg := make(map[interface{}]interface{})
	if err = yaml.Unmarshal([]byte(d.Text[4:confEnd]), &cfg); err != nil {
		return fmt.Errorf("unable to read config for %q: %q", d.Name, err)
	}
	cobra.Tag("doc").LogfV("read config for %q", d.Name)

	for k, v := range cfg {
		cobra.Tag("doc").Add("key", k).Add("val", v).LogV("setting config parameter")
		// cobra.Set(k.(string), v)
		switch k {
		case "reflow":
			d.Reflow = v.(bool)
		case "format":
			d.Format = v.(string)
		case "title":
			d.Title = v.(string)
		case "date":
			d.Date = v.(time.Time)
		case "ignore":
			d.Ignore = v.(bool)
		case "output":
			d.OutputName = v.(string)
		case "template":
			d.Template = v.(string)
		case "mode":
			mode := v.(string)
			if strings.ToLower(mode) == "plain" {
				d.Plain = true
			}
		case "packages":
			d.Packages, err = readPackageList(v)
			if err != nil {
				return err
			}
			err = d.Folio.LoadPackages(d.Packages)
			if err != nil {
				return err
			}
		}
	}

	if d.Folio.CheckFlag("plain") {
		d.Plain = cobra.GetBool("plain")
	}
	if d.Folio.CheckFlag("reflow") {
		d.Reflow = cobra.GetBool("reflow")
	}
	if d.Folio.CheckFlag("format") {
		d.Format = cobra.GetString("format")
	}

	d.Initialized = true
	return nil
}

func (d *Document) loadText() (err error) {
	var in []byte

	if d.Path == "<stdin>" {
		var input []byte

		reader := bufio.NewReader(os.Stdin)
		for {
			in, err := reader.ReadByte()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			input = append(input, in)
		}

		d.Text = string(input)
	} else {
		in, err = ioutil.ReadFile(d.Path)
		if err != nil {
			return
		}
		d.Text = string(in)
	}

	return nil
}

func (d *Document) String() string {
	return d.Title
}

// readPackageList reads the package list passed in from a document's config
// preamble.
func readPackageList(list interface{}) ([]string, error) {
	pkgs := []string{}

	pkgv := reflect.ValueOf(list)
	if pkgv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("package config should be a list")
	}

	for _, item := range list.([]interface{}) {
		iv := reflect.ValueOf(item.(interface{}))

		switch iv.Kind() {
		case reflect.String:
			pkgs = append(pkgs, item.(string))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			pkgs = append(pkgs, strconv.FormatInt(iv.Int(), 10))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			pkgs = append(pkgs, strconv.FormatUint(iv.Uint(), 10))
		case reflect.Float32, reflect.Float64:
			pkgs = append(pkgs, strconv.FormatFloat(iv.Float(), 'f', -1, 64))
		case reflect.Bool:
			pkgs = append(pkgs, strconv.FormatBool(iv.Bool()))
		default:
			switch item.(type) {
			case Stringer:
				pkgs = append(pkgs, item.(Stringer).String())
			default:
				return nil, fmt.Errorf("unknown type in package list: %q (%T)", iv, item)
			}
		}
	}

	return pkgs, nil
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

	// out := r.render(root)
	out := r.renderToString(root)

	r.Doc.Output = out

	if r.Doc.Template != "" {
		tcmd := NewCmdNode(r.Doc.Template, &token{
			typeof: tokenCmdStart,
			loc:    Loc(0),
			lnum:   0,
			value:  "",
		})
		tcmd.Format = r.Doc.Format
		out = r.processPageTemplate(tcmd)
	}

	r.Doc.Output = out
	r.Doc.Rendered = true
	return out, nil
}
