package document

import (
	"fmt"
	// "os"
	"strings"
	"strconv"
	// "text/template"
	"github.com/kevinkenan/cobra"
	"github.com/kevinkenan/subtext/parse"
	"gopkg.in/yaml.v2"
)

// Document represents the text being processed.
type Document struct {
	Name       string
	Packages   []string
	Output     string
	Targets    []string
	Metadata   map[string]string
	Text       string
	Root       *parse.Section
	Options    *parse.Options
	Plain      bool // Don't generate paragraphs or aggressively eat whitespace
	ReflowPars bool // if true, remove new lines and collapse whitespace in paragraphs
	macrosIn   []*parse.Macro
}

type RenderError struct {
	message string
}

func (r RenderError) Error() string {
	return r.message
}

// RenderExecution represents a render process and keeps track information
// needed during the rendering.
type Render struct {
	*Document
	ParagraphMode bool
	InParagraph   bool       // true indicates that execution is in a paragraph.
	ParBuffer     *parse.Cmd //
	depth         int        // tracks recursion depth
	skipNodeCount int        // skip the next nodes
	init          bool       // true if in init mode (no output is written)
	macros        parse.MacroMap
}

// func NewRenderEngine(d *Document) *Render {
// 	return new(Render{Document: d})
// }

func NewDoc() *Document {
	d := Document{macrosIn: []*parse.Macro{}}
	// d.AddParagraphMacros()
	return &d
}

// func NewDoc() *Document {
// 	var m *parse.Macro
// 	// var opt parse.Optional
// 	// macs := make(map[string]*parse.Macro)
// 	opt := parse.Optional{Name: "second", Default: "def"}
// 	d := Document{macros: make(map[string]*parse.Macro)}
// 	m = parse.NewMacro("a", "<it>{{- .first -}}</it>{{.second}}", []string{"first"}, []*parse.Optional{&opt})
// 	d.macros[m.Name] = m
// 	m = parse.NewMacro("b", "<b>{{.first}}</b>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	m = parse.NewMacro("c", "<sc>{{.first}}</sc>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	m = parse.NewMacro("section", "<section>{{.first}}\n</section>", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	// m = parse.NewMacro("begin", "•b{test}<body>\n•^(pm=on)", []string{"first"}, nil) //[]*Optional{&opt})
// 	m = parse.NewMacro("begin", "<body>\n•b{test}\n", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	// m = parse.NewMacro("end", "\n</body>•^(pm=off)•-\n", []string{"first"}, nil) //[]*Optional{&opt})
// 	m = parse.NewMacro("end", "\n</body>\n", []string{"first"}, nil) //[]*Optional{&opt})
// 	d.macros[m.Name] = m
// 	return &d
// }

func (d *Document) AddMacro(m *parse.Macro) {
	d.macrosIn = append(d.macrosIn, m)
}

func (d *Document) AddParagraphMacros() {
	// Add sys.paragraph.* macros which are used when paragraph mode is off.
	d.AddMacro(parse.NewMacro("sys.paragraph.begin", "{{.parbreak}}", []string{"parbreak"}, nil))
	d.AddMacro(parse.NewMacro("sys.paragraph.end", "{{.parbreak}}", []string{"parbreak"}, nil))
	// d.AddMacro(parse.NewMacro("paragraph.begin", "<p>", []string{"orig"}, nil))
	// d.AddMacro(parse.NewMacro("paragraph.end", "</p>\n", []string{"orig"}, nil))
	// Add paragraph.* macros which are used when paragraph mode is on.
	d.AddMacro(parse.NewMacro("paragraph.begin", "", nil, []*parse.Optional{parse.NewOptional("ignore", "")}))
	d.AddMacro(parse.NewMacro("paragraph.end", "\n\n", nil, []*parse.Optional{parse.NewOptional("ignore", "")}))
}

func (d *Document) Make() (s string, err error) {
	r := &Render{Document: d, ParagraphMode: !d.Plain}
	s, err = MakeWith(d.Text, r, d.Options)
	return
}

// MakeWidth allows arbitrary text to be processed with an existing Render
// context. Most of the time the Document's Make is used (which calls
// MakeWith), but MakeWith itself is useful for handling macros embedded in
// templates.
func MakeWith(t string, r *Render, options *parse.Options) (s string, err error) {
	defer func() { cobra.LogV("finished rendering") }()
	defer func() {
		if e := recover(); e != nil {
			switch e.(type) {
			case RenderError, parse.Error:
				err = e.(error)
			default:
				panic(e)
			}
		}
	}()

	root, macros, err := parse.Parse(r.Name, t, options)

	if err != nil {
		return "", err
	} else {
		r.macros = macros
		cobra.LogV("rendering (render)")
		return r.render(root), nil
	}
}

func (r *Render) render(root *parse.Section) string {
	cobra.Tag("render").LogV("begin render")
	s := new(strings.Builder)
	s.WriteString(r.renderSection(root))
	return s.String()
}

func (r *Render) renderSection(n *parse.Section) string {
	s := new(strings.Builder)
	for _, l := range n.NodeList {
		cobra.Tag("render").LogV("next node in section")
		s.WriteString(r.renderNode(l))
	}
	return s.String()
}

func (r *Render) renderNode(n parse.Node) string {
	if r.skipNodeCount > 0 {
		cobra.Tag("render").WithField("skipNodeCount", r.skipNodeCount).LogV("skipping node")
		r.skipNodeCount -= 1
		return ""
	}
	r.depth += 1
	if r.depth > 50 {
		panic(RenderError{message: "exceeded call depth"})
	}
	s := new(strings.Builder)

	switch n.(type) {
	case *parse.Section:
		cobra.Tag("render").LogV("rendering section node")
		s.WriteString(r.renderSection(n.(*parse.Section)))

	case *parse.Text:
		if r.init {
			cobra.Tag("render").LogV("init mode so skipping text render")
			return ""
		}

		// if r.ParBuffer != nil {
		// 	cobra.Tag("render").LogV("processing paragraph buffer in text")
		// 	par := r.ParBuffer
		// 	r.ParBuffer = nil
		// 	s.WriteString(r.processCmd(par))
		// }

		cobra.Tag("render").LogV("rendering text")

		// // reflow paragraph if requested
		// var text string
		// if r.ParagraphMode && r.InParagraph && r.ReflowPars {
		// 	cobra.Tag("render").LogV("reflowing paragraph")
		// 	// text = strings.Join(strings.Fields(n.(*parse.Text).GetText()), " ")
		// 	text = strings.Replace(n.(*parse.Text).GetText(), "\n", " ", -1)
		// } else {
		// 	text = n.(*parse.Text).GetText()
		// }
		text := n.(*parse.Text).GetText()
		s.WriteString(text)

	case *parse.Cmd:
		c := n.(*parse.Cmd)
		cobra.Tag("render").WithField("argcount", len(c.ArgList)+len(c.ArgMap)).Add("name", c.NodeValue).LogV("rendering cmd node")

		if c.SysCmd {
			s.WriteString(r.processSysCmd(c))
		} else {
			s.WriteString(r.processCmd(c))
		}

		// s.WriteString(sc)

	// case *parse.SysCmd:
	// 	c := n.(*parse.SysCmd)
	// 	cobra.Tag("render").Add("name", c.NodeValue).LogV("rendering syscmd node")

	case *parse.ErrorNode:
		cobra.Tag("render").LogV("rendering error node")
		s.WriteString(n.(*parse.ErrorNode).GetErrorMsg())

	default:
		panic(RenderError{message: fmt.Sprintf("unexpected node %q\n", n)})
	}

	cobra.Tag("render").LogV("done rendering a node")
	r.depth -= 1
	return s.String()
}

func (r *Render) renderNodeList(n parse.NodeList) string {
	cobra.Tag("render").WithField("length", len(n)).LogV("rendering node list")
	s := new(strings.Builder)
	for _, l := range n {
		s.WriteString(r.renderNode(l))
	}
	return s.String()
}

func (r *Render) processSysCmd(n *parse.Cmd) string {
	out := ""
	// name := fmt.Sprintf("sys.%s", n.GetCmdName())
	name := n.GetCmdName()
	flowStyle := false
	cobra.Tag("render").WithField("cmd", name).LogV("processing system command (cmd)")
	switch name {
	case "configf":
		flowStyle = true
		fallthrough
	case "config":
		r.handleSysConfigCmd(n, flowStyle)
	case "sys.init.begin":
		r.init = true
	case "sys.init.end":
		r.init = false
	case "sys.setdataf":
		r.setData(n, true)
	case "sys.setdata":
		r.setData(n, false)
	case "sys.incr":
		r.increment(n)
	case "sys.exec":
		out = r.exec(n)
	case "sys.import":
	default:
		panic(RenderError{message: fmt.Sprintf("Line %d: unknown system command: %q", n.GetLineNum(), name)})
	}
	return out
}

func (r *Render) exec(n *parse.Cmd) string {
	cobra.Tag("cmd").LogfV("begin exec")
	name := n.GetCmdName()
	cobra.Tag("render").WithField("cmd", name).LogV("rendering command (cmd)")
	cmdLog := cobra.Tag("cmd")

	// Get the macro definition.
	m := r.macros.GetMacro(name, "")
	if m == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: macro %q not defined.", n.GetLineNum(), name)})
	}
	cmdLog.Copy().Strunc("macro", m.TemplateText).LogfV("retrieved macro definition")

	args, err := m.ValidateArgs(n)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on macro %q: %q", n.GetLineNum(), name, err)})
	}

	// Load the validated args into a map for easy access.
	renArgs := map[string]interface{}{}
	for k, v := range args {
		renArgs[k] = v.String()
		cmdLog.Copy().Strunc("arg", k).Strunc("val", v).LogV("prepared command argument")
	}

	m = parse.NewBlockMacro("anon", renArgs["template"].(string), nil, nil)

	// renArgs = map[string]interface{}{}
	parse.Data["reflow"] = r.Options.Reflow
	parse.Data["format"] = r.Options.Format
	parse.Data["plain"] = r.Options.Plain
	parse.Data["flags"] = n.Flags
	renArgs["data"] = parse.Data

	// Apply the command's arguments to the macro.
	s, err := r.ExecuteMacro(m, renArgs)
	if err != nil {
		// fmt.Println(err)
		panic(RenderError{fmt.Sprintf("error rendering macro %q: %s", name, err)})
	}
	cmdLog.Copy().Add("name", name).Add("ld", m.Ld).Logf("executed macro, ready for parsing")

	// Handle commands embedded in the macro.
	opts := &parse.Options{Plain: true, Macros: r.macros}
	output, _, err := parse.Parse(name, s, opts)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: error in template for macro %q: %q", n.GetLineNum(), name, err)})
	} else {
		cmdLog.Copy().Add("nodes", output.Count()-1).LogfV("parsed macro, ready for rendering")
		outs := r.render(output)

		if n.Block && !r.Options.Plain {
			outs = outs + "\n"
		}
		cobra.Tag("cmd").LogfV("end exec")
		return outs
	}
}

func (r *Render) increment(n *parse.Cmd) {
	cobra.Tag("cmd").LogfV("begin incr")
	name := "sys.incr"

	d := r.macros.GetMacro(name, r.Options.Format)
	if d == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: system command %q not defined.", n.GetLineNum(), name)})
	}

	args, err := d.ValidateArgs(n)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on system command %q: %q", n.GetLineNum(), name, err)})
	}

	// data := parse.Data["data"].(map[string]interface{})
	ctrs := parse.Data["ctr"].(map[interface{}]interface{})
	v, _ :=strconv.Atoi("1")
	keyName := strings.TrimPrefix(args["key"].String(), ".data.ctr.")

	keyVal, found := ctrs[keyName].(int)
	if !found {
		panic(RenderError{message: fmt.Sprintf("Line %d: unable to find key %q to increment", n.GetLineNum(), keyName)})
	}

	keyVal += v
	ctrs[keyName] = keyVal
	parse.Data["ctrs"] = ctrs

	cobra.Tag("cmd").LogfV("end incr")
	return
}

func (r *Render) setData(n *parse.Cmd, flowStyle bool)  {
	cobra.Tag("cmd").LogfV("begin setData")
	name := "sys.setdata"
	// Retrieve the sys.data system command
	d := r.macros.GetMacro(name, r.Options.Format)
	if d == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: system command %q not defined.", n.GetLineNum(), name)})
	}

	args, err := d.ValidateArgs(n)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on system command %q: %q", n.GetLineNum(), name, err)})
	}

	cobra.Tag("cmd").Strunc("syscmd", args["data"].String()).LogfV("system command: %s", args["data"])

	data := make(map[interface{}]interface{})
	if flowStyle {
		err = yaml.Unmarshal([]byte("{"+args["data"].String()+"}"), data)
	} else {
		err = yaml.Unmarshal([]byte(args["data"].String()), data)
	}

	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: unmarshall error for system command %q: %q", n.GetLineNum(), name, err)})
	}

	for k, v := range data {
		parse.Data[k.(string)] = v
	}

	cobra.Tag("cmd").LogfV("end setData")
	return
}

func (r *Render) handleSysConfigCmd(n *parse.Cmd, flowStyle bool) {
	name := "sys.config"
	// Retrieve the sys.newmacro system command
	d := r.macros.GetMacro(name, r.Options.Format)
	if d == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: system command %q not defined.", n.GetLineNum(), name)})
	}
	cobra.Tag("cmd").Strunc("macro", d.TemplateText).LogfV("retrieved system command definition")

	args, err := d.ValidateArgs(n)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on system command %q: %q", n.GetLineNum(), name, err)})
	}

	cobra.Tag("cmd").Strunc("syscmd", args["configs"].String()).LogfV("system command: %s", args["configs"])

	cfg := make(map[interface{}]interface{})
	if flowStyle {
		err = yaml.Unmarshal([]byte("{"+args["configs"].String()+"}"), &cfg)
	} else {
		err = yaml.Unmarshal([]byte(args["configs"].String()), &cfg)
	}
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: unmarshall error for system command %q: %q", n.GetLineNum(), name, err)})
	}
	cobra.Tag("cmd").LogfV("marshalled syscmd: %+v", cfg)

	for k, v := range cfg {
		cobra.Tag("render").Add("key", k).Add("val", v).LogV("setting config from sys command")
		cobra.Set(k.(string), v)
	}

	r.ReflowPars = cobra.GetBool("reflow")
}

func (r *Render) processCmd(n *parse.Cmd) string {
	name := n.GetCmdName()
	cobra.Tag("render").WithField("cmd", name).LogV("rendering command (cmd)")
	cmdLog := cobra.Tag("cmd")

	// // If we are in paragraph mode, scanner generated paragraphs (prefixed
	// // with "sys.") require extra processing to remove empty paragraphs. If
	// // the paragraph isn't empty, we remove the prefix so that regular
	// // paragraph handling is triggered.
	// if r.ParagraphMode {
	// 	switch {
	// 	case name == "sys.paragraph.begin":
	// 		n.NodeValue = parse.NodeValue("paragraph.begin")
	// 		r.InParagraph = true
	// 		r.ParBuffer = n
	// 		return ""
	// 	case name == "sys.paragraph.end":
	// 		r.InParagraph = false
	// 		if r.ParBuffer != nil {
	// 			r.ParBuffer = nil
	// 			cobra.Tag("render").LogfV("empty paragraph so skipping nodes")
	// 			return ""
	// 		}
	// 		name = "paragraph.end"
	// 	}
	// }

	format := r.Options.Format
	if n.HasFlag("noformat") {
		format = ""
	} else if f, ok := n.HasFlagVar("format"); ok {
		format = f
	}
	// Get the macro definition.
	m := r.macros.GetMacro(name, format)
	if m == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: macro %q not defined.", n.GetLineNum(), name)})
	}
	cmdLog.Copy().Strunc("macro", m.TemplateText).LogfV("retrieved macro definition")

	args, err := m.ValidateArgs(n)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on macro %q: %q", n.GetLineNum(), name, err)})
	}
	// The args are themselves nodes which contain text and commands. They
	// need to be rendered as well.
	// renArgs := map[string]string{}
	// for k, v := range args {
	// 	renArgs[k] = r.renderNodeList(v)
	// }

	// Load the validated args into a map for easy access.
	renArgs := map[string]interface{}{}
	for k, v := range args {
		renArgs[k] = v.String()
		cmdLog.Copy().Strunc("arg", k).Strunc("val", v).LogV("prepared command argument")
	}

	// data := 
	// testdata2 := map[string]string{}
	// data := map[string]map[string]string{"data": testdata}
	// sys := map[string]map[string]map[string]string{"sys": data}
	
	parse.Data["reflow"] = r.Options.Reflow
	parse.Data["format"] = r.Options.Format
	parse.Data["plain"] = r.Options.Plain
	parse.Data["flags"] = n.Flags
	renArgs["data"] = parse.Data

	// Apply the command's arguments to the macro.
	s, err := r.ExecuteMacro(m, renArgs)
	if err != nil {
		// fmt.Println(err)
		panic(RenderError{fmt.Sprintf("error rendering macro %q: %s", name, err)})
	}
	cmdLog.Copy().Add("name", name).Add("ld", m.Ld).Logf("executed macro, ready for parsing")

	// Handle commands embedded in the macro.
	opts := &parse.Options{Plain: true, Macros: r.macros}
	output, _, err := parse.Parse(name, s, opts)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: error in template for macro %q: %q", n.GetLineNum(), name, err)})
	} else {
		cmdLog.Copy().Add("nodes", output.Count()-1).LogfV("parsed macro, ready for rendering")
		outs := r.render(output)

		if n.Block && !r.Options.Plain {
			outs = outs + "\n"
		}

		return outs
	}

	// cobra.Tag("cmd").LogfV(">> %q", s)
	// return output.String()
}

func (r *Render) ExecuteMacro( m *parse.Macro, data map[string]interface{}) (string, error) {
	s := strings.Builder{}
	err := m.Template.Delims(m.Ld, m.Rd).Option("missingkey=error").Execute(&s, data)
	if err != nil {
		return "", err
	}
	return s.String(), nil
}
