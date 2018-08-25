package core

import (
	"fmt"
	"text/template"
	// "os"
	"strings"
	// "strconv"
	// "text/template"
	"github.com/kevinkenan/cobra"
	"gopkg.in/yaml.v2"
)

type RenderError struct {
	message string
}

func (r RenderError) Error() string {
	return r.message
}

// RenderExecution represents a render process and keeps track information
// needed during the rendering.
type Render struct {
	Doc           *Document
	InParagraph   bool     // true indicates that execution is in a paragraph.
	ParBuffer     *Cmd     //
	depth         int      // tracks recursion depth
	context       []string // macro/arg call stack
	skipNodeCount int      // skip the next nodes
	init          bool     // true if in init mode (no output is written)
	ref           bool     // true if references should be rendered
}

func NewRender(d *Document) *Render {
	return &Render{
		Doc: d,
	}
}

// GetMacro is a convenience function to get a macro.
func (r *Render) getMacro(name, format string) *Macro {
	return r.Doc.Folio.GetMacro(name, format)
}

// AddMacro is a convenience function to add a macro.
func (r *Render) addMacro(m *Macro) {
	r.Doc.Folio.AddMacro(m)
}

// AddMacro is a convenience function to merge a MacroMap
func (r *Render) addMacros(mm MacroMap) {
	r.Doc.Folio.AddMacros(mm)
}

func (r *Render) render(root *Section) string {
	cobra.Tag("render").LogV("begin render")
	s := new(strings.Builder)
	s.WriteString(r.renderSection(root))
	return s.String()
}

func (r *Render) renderSection(n *Section) string {
	s := new(strings.Builder)
	for _, l := range n.NodeList {
		cobra.Tag("render").LogV("next node in section")
		s.WriteString(r.renderNode(l))
	}
	return s.String()
}

func (r *Render) renderNode(n Node) string {
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
	case *Section:
		cobra.Tag("render").LogV("rendering section node")
		s.WriteString(r.renderSection(n.(*Section)))
	case *Text:
		// if r.init {
		// 	cobra.Tag("render").LogV("init mode so skipping text render")
		// 	return ""
		// }

		cobra.Tag("render").LogV("rendering text")
		text := n.(*Text).GetText()
		s.WriteString(text)
	case *Cmd:
		c := n.(*Cmd)
		cobra.Tag("render").WithField("argcount", len(c.ArgList)+len(c.ArgMap)).Add("name", c.NodeValue).LogV("rendering cmd node")

		if c.SysCmd {
			s.WriteString(r.processSysCmd(c))
		} else {
			s.WriteString(r.processCmd(c))
		}
	case *ErrorNode:
		cobra.Tag("render").LogV("rendering error node")
		s.WriteString(n.(*ErrorNode).GetErrorMsg())
	default:
		panic(RenderError{message: fmt.Sprintf("unexpected node %q\n", n)})
	}

	cobra.Tag("render").LogV("done rendering a node")
	r.depth -= 1
	return s.String()
}

func (r *Render) renderNodeList(n NodeList) string {
	cobra.Tag("render").WithField("length", len(n)).LogV("rendering node list")
	s := new(strings.Builder)
	for _, l := range n {
		s.WriteString(r.renderNode(l))
	}
	return s.String()
}

func (r *Render) processSysCmd(n *Cmd) string {
	out := ""
	name := n.GetCmdName()
	cobra.Tag("render").WithField("cmd", name).LogV("processing system command (cmd)")

	switch name {
	// case "sys.configf":
	// 	flowStyle = true
	// 	fallthrough
	// case "sys.config":
	// 	r.handleSysConfigCmd(n, flowStyle)
	case "sys.init.begin":
		r.init = true
	case "sys.init.end":
		r.init = false
	case "sys.setdataf":
		r.setData(n, true)
	case "sys.setdata":
		r.setData(n, false)
	case "sys.exec":
		out = r.exec(n)
	case "sys.refdef":
		r.setRef(n, false)
	case "sys.ref":
		if r.ref {
			out = r.getRef(n, false)
		}
	case "sys.import":
	default:
		panic(RenderError{message: fmt.Sprintf("Line %d: unknown system command: %q", n.GetLineNum(), name)})
	}

	return out
}

func (r *Render) exec(n *Cmd) string {
	cobra.Tag("cmd").LogfV("begin exec")
	name := n.GetCmdName()
	cobra.Tag("render").WithField("cmd", name).LogV("rendering command (cmd)")
	cmdLog := cobra.Tag("cmd")

	// Get the macro definition.
	m := r.getMacro(name, "")
	if m == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: macro %q not defined in exec.", n.GetLineNum(), name)})
	}
	cmdLog.Copy().Strunc("macro", m.TemplateText).LogfV("retrieved macro definition")

	args, err := m.ValidateArgs(n, r.Doc)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on macro %q: %q", n.GetLineNum(), name, err)})
	}

	// renArgs := map[string]interface{}{}
	renArgs := newCmdArgs(r.Doc)
	for k, v := range args {
		renArgs[k] = r.renderNodeList(v)
		// renArgs[k] = v.String()
		cmdLog.Copy().Strunc("arg", k).Strunc("val", v).LogV("prepared command argument")
	}

	m = NewBlockMacro("exec", renArgs["template"].(string), nil, nil)

	// Apply the command's arguments to the macro.
	s, err := r.ExecuteMacro(m, renArgs, false)
	if err != nil {
		// fmt.Println(err)
		panic(RenderError{fmt.Sprintf("error rendering macro %q: %s", name, err)})
	}
	cmdLog.Copy().Add("name", name).Add("ld", m.Ld).Logf("executed macro, ready for parsing")

	// Handle commands embedded in the macro.
	// opts := &Options{Plain: true, Macros: r.macros}
	output, err := ParseMacro(name, s, r.Doc, r.depth)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: error in template for macro %q: %q", n.GetLineNum(), name, err)})
	} else {
		cmdLog.Copy().Add("nodes", output.Count()-1).LogfV(" macro, ready for rendering")
		outs := r.render(output)

		if n.Block && !r.Doc.Plain {
			outs = outs + "\n"
		}
		cobra.Tag("cmd").LogfV("end exec")
		return outs
	}
}

func (r *Render) setRef(cmd *Cmd, flowStyle bool) {
	cobra.Tag("cmd").LogfV("begin setRef")
	name := "sys.refdef"

	d := r.getMacro(name, "")
	if d == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: system command %q not defined.", cmd.GetLineNum(), name)})
	}

	args, err := d.ValidateArgs(cmd, r.Doc)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on %s: %s", cmd.GetLineNum(), name, err)})
	}

	cobra.Tag("cmd").Strunc("syscmd", args["data"].String()).LogfV("system command: %s", args["data"])

	data := make(map[interface{}]interface{})
	if flowStyle {
		err = yaml.Unmarshal([]byte("{"+args["data"].String()+"}"), data)
	} else {
		err = yaml.Unmarshal([]byte(args["data"].String()), data)
	}

	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: unmarshall error for system command %q: %q", cmd.GetLineNum(), name, err)})
	}

	r.Doc.Folio.SetData("ref."+args["label"].String(), args["ref"].String())

	cobra.Tag("cmd").LogfV("end setRef")
	return
}

func (r *Render) getRef(cmd *Cmd, flowStyle bool) (out string) {
	cobra.Tag("cmd").LogfV("begin getRef")
	name := "sys.ref"

	d := r.getMacro(name, "")
	if d == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: system command %q not defined.", cmd.GetLineNum(), name)})
	}

	args, err := d.ValidateArgs(cmd, r.Doc)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on %s: %s", cmd.GetLineNum(), name, err)})
	}

	cobra.Tag("cmd").Strunc("syscmd", args["data"].String()).LogfV("system command: %s", args["data"])

	if ref, found := r.Doc.Folio.Data["ref."+args["label"].String()]; !found {
		panic(RenderError{message: fmt.Sprintf("line %d: ref '%s' was not found", cmd.GetLineNum(), args["label"].String())})
	} else {
		out = string(ref.(string))
	}

	cobra.Tag("cmd").LogfV("end getRef")
	return
}

// TODO: Can I remove this function?
func (r *Render) setData(n *Cmd, flowStyle bool) {
	cobra.Tag("cmd").LogfV("begin setData")
	name := "sys.setdata"
	// Retrieve the sys.data system command
	d := r.getMacro(name, "")
	if d == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: system command %q not defined.", n.GetLineNum(), name)})
	}

	args, err := d.ValidateArgs(n, r.Doc)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on setData %q: %q", n.GetLineNum(), name, err)})
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
		Data[k.(string)] = v
	}
	fmt.Println("--- setData", Data)

	cobra.Tag("cmd").LogfV("end setData")
	return
}

type cmdArgs map[string]interface{}

func (c cmdArgs) FlagSet(s string) bool {
	if c["Flags"] == nil {
		return false
	}

	switch c["Flags"].(type) {
	case []string:
		for _, f := range c["Flags"].([]string) {
			if f == s {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func newCmdArgs(d *Document) (c cmdArgs) {
	c = make(cmdArgs)
	c["Doc"] = d
	c["Data"] = d.Folio.Data
	// c["Body"] = d.Output
	return
}

// processPageTemplate differes from processCmd in that the output of
// executing the template is not itself rendered. This allows example macro
// definitions to be placed in the body text. If we used processCmd, those
// example definitions would be executed.
func (r *Render) processPageTemplate(n *Cmd) string {
	name := n.GetCmdName()
	r.pushContext(name)
	cobra.Tag("render").WithField("cmd", name).LogV("rendering page template (cmd)")
	cmdLog := cobra.Tag("cmd")

	// Get the macro definition.
	m := r.getMacro(name, n.Format)
	if m == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: macro %q (format %q) not defined.", n.GetLineNum(), name, n.Format)})
	}
	cmdLog.Copy().Strunc("macro", m.TemplateText).LogfV("retrieved macro definition")

	mac, err := ParseMacro(name, m.TemplateText, r.Doc, r.depth)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: error in page template %q: %q", n.GetLineNum(), name, err)})
	}

	// render the macro and parse the resulting template
	tmplt := r.render(mac)
	t := template.Must(template.New(name).Funcs(funcMap).Delims("[[", "]]").Option("missingkey=error").Parse(tmplt))

	// execute the template
	s := strings.Builder{}
	data := make(map[string]string)
	data["Body"] = r.Doc.Output
	err = t.Delims(m.Ld, m.Rd).Option("missingkey=error").Execute(&s, data)
	if err != nil {
		panic(RenderError{fmt.Sprintf("error executing page template %q: %s", name, err)})
	}

	r.popContext()

	return s.String()
}

func (r *Render) processCmd(n *Cmd) string {
	var err error
	name := n.GetCmdName()
	r.pushContext(name)
	cobra.Tag("render").WithField("cmd", name).LogV("rendering command (cmd)")
	cmdLog := cobra.Tag("cmd")

	// Get the macro definition.
	m := r.getMacro(name, n.Format)
	if m == nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: macro %q (format %q) not defined.", n.GetLineNum(), name, n.Format)})
	}
	cmdLog.Copy().Strunc("macro", m.TemplateText).LogfV("retrieved macro definition")

	if m.InitTemplate != nil {
		data := map[string]interface{}{}
		data["Data"] = r.Doc.Folio.Data
		_, err := r.ExecuteMacro(m, data, true)
		if err != nil {
			panic(RenderError{fmt.Sprintf("error executing init template %q: %s", name, err)})
		}
		cmdLog.Copy().Add("name", name).Add("ld", m.Ld).Logf("executed init macro")
	}

	args, err := m.ValidateArgs(n, r.Doc)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: ValidateArgs failed on macro %q: %s", n.GetLineNum(), name, err)})
	}

	renArgs := newCmdArgs(r.Doc)
	renArgs["Flags"] = n.Flags
	// Load the validated args into a map for easy access.
	for k, v := range args {
		renArgs["Context"] = r.context
		r.pushContext("#" + k)
		renArgs[k] = r.renderNodeList(v)
		cmdLog.Copy().Strunc("arg", k).Strunc("val", renArgs[k]).LogV("prepared command argument")
		r.popContext()
	}

	// Apply the command's arguments to the macro.
	s, err := r.ExecuteMacro(m, renArgs, false)
	if err != nil {
		panic(RenderError{fmt.Sprintf("error rendering macro %q: %s", name, err)})
	}
	cmdLog.Copy().Add("name", name).Add("ld", m.Ld).Logf("executed macro, ready for parsing")

	// Handle commands embedded in the macro.
	var output *Section
	// plain := false

	// if strings.HasPrefix(name, "paragraph") {
	// 	plain = true
	// }

	// output, err = ParseText(s, plain, r.Doc)
	output, err = ParseMacro(name, s, r.Doc, r.depth)
	if err != nil {
		panic(RenderError{message: fmt.Sprintf("Line %d: error in template for macro %q: %q", n.GetLineNum(), name, err)})
	}
	cmdLog.Copy().Add("nodes", output.Count()-1).LogfV("parsed macro, ready for rendering")

	outs := r.render(output)

	if n.Block && !r.Doc.Plain {
		outs = outs + "\n"
	}

	r.popContext()

	return outs
}

func (r *Render) pushContext(s string) {
	r.context = append(r.context, s)
}

func (r *Render) popContext() string {
	l := len(r.context)
	if l == 0 {
		panic(RenderError{message: fmt.Sprintf("attempted to read past the end of the command context")})
	}
	c := r.context[l-1]
	r.context = r.context[:l-1]
	return c
}

func (r *Render) ExecuteMacro(m *Macro, data cmdArgs, init bool) (string, error) {
	s := strings.Builder{}
	t := m.Template
	if init {
		t = m.InitTemplate
	}
	err := t.Delims(m.Ld, m.Rd).Option("missingkey=error").Execute(&s, data)
	if err != nil {
		return "", err
	}
	return s.String(), nil
}
