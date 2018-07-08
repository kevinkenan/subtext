package parse

import (
	// "errors"
	"fmt"
	"strings"
	"text/template"
	// "unicode"
	// "unicode/utf8"
	"github.com/kevinkenan/cobra"
	"gopkg.in/yaml.v2"
)

var Data map[string]interface{} = map[string]interface{}{}

type Optional struct {
	Name    string
	Default string
}

func NewOptional(name, dflt string) *Optional {
	return &Optional{name, dflt}
}

type MacroDef struct {
	Name       string        // The macro's name to match command names
	Template   string        // The Go template that defines the macro
	Parameters []string      // Required parameters
	Optionals  yaml.MapSlice // Optional parameters in correct order
	Format     string        // The format, e.g. html or latex
	Block      bool          // True if macro should be rendered as a block
	Delims     [2]string     // Left and right delim used in the template
}

type MacroType struct {
	Name, Format string
}

type MacroMap map[MacroType]*Macro

func NewMacroMap() MacroMap {
	mm := MacroMap{}

	// Default macros
	macs := []*Macro{
		// System Macros
		NewMacro("sys.newmacro", "", []string{"def"}, nil),
		NewMacro("sys.newmacrof", "", []string{"def"}, nil),
		NewMacro("sys.config", "", []string{"configs"}, nil),
		NewMacro("sys.configf", "", []string{"configs"}, nil),
		NewMacro("sys.init.begin", "", nil, nil),
		NewMacro("sys.init.end", "", nil, nil),
		NewMacro("sys.exec", "", []string{"template"}, nil),
		NewMacro("sys.import", "", nil, nil),
		NewMacro("sys.setdata", "", []string{"data"}, nil),
		NewMacro("sys.setdataf", "", []string{"data"}, nil),
		// Regular macros
		NewMacro("echo", "((.text))", []string{"text"}, nil),
		NewBlockMacro("Echo", "((.text))", []string{"text"}, nil),
		NewMacro("paragraph.begin", "<", nil, nil),
		NewMacro("paragraph.end", ">\n", nil, nil),
		NewMacro("subtext", "subtext, version 0.0.1", nil, nil),
		NewBlockMacro("Subtext", "subtext, version 0.0.1", nil, nil),
	}

	mt := MacroType{Name: "", Format: ""}

	// Add default macros
	for _, m := range macs {
		mt.Name = m.Name
		mm[mt] = m
	}

	return mm
}

func (m MacroMap) GetMacro(name, format string) *Macro {
	mt := MacroType{name, format}
	mac, found := m[mt]
	if found {
		return mac
	}

	mt.Format = ""
	mac, found = m[mt]

	if found {
		return mac
	}

	return nil
}

type Macro struct {
	Name               string      // The macro's name to match command names
	TemplateText       string      // The Go template that defines the macro
	*template.Template             // the parsed template
	Parameters         []string    // Required parameters
	Optionals          []*Optional // Optional parameters in correct order
	Format     string        // The format, e.g. html or latex
	Block              bool        // True if macro should be rendered as a block
	Ld                 string      // Left delim used in the template
	Rd                 string      // Right delim used in the template
}

func NewBlockMacro(name, tmplt string, params []string, optionals []*Optional) *Macro {
	t := template.Must(template.New(name).Funcs(funcMap).Delims("((", "))").Option("missingkey=error").Parse(tmplt))
	return &Macro{
		Name:         name,
		Parameters:   params,
		Optionals:    optionals,
		TemplateText: tmplt,
		Template:     t,
		Block:        true,
		Ld:           "((",
		Rd:           "))"}
}

func NewMacro(name, tmplt string, params []string, optionals []*Optional) *Macro {
	t := template.Must(template.New(name).Funcs(funcMap).Delims("((", "))").Option("missingkey=error").Parse(tmplt))
	return &Macro{
		Name:         name,
		Parameters:   params,
		Optionals:    optionals,
		TemplateText: tmplt,
		Template:     t,
		Ld:           "((",
		Rd:           "))"}
}

func (m *Macro) Parse() {
	t := template.Must(template.New(m.Name).Funcs(funcMap).Delims(m.Ld, m.Rd).Option("missingkey=error").Parse(m.TemplateText))
	m.Template = t
}

func (m *Macro) String() string {
	w := new(strings.Builder)
	// w.WriteString("\n")
	w.WriteString(fmt.Sprintf("Name %s, ", m.Name))
	w.WriteString(fmt.Sprintf("  Template %s, ", m.TemplateText))
	w.WriteString(fmt.Sprintf("  Format %s, ", m.Format))
	w.WriteString(fmt.Sprintf("  Parms %s,", m.Parameters))
	w.WriteString(fmt.Sprintf("  ListOpts %s", m.ListOptions()))
	return w.String()
}

func (m *Macro) ListOptions() (opts []string) {
	opts = []string{}
	for _, o := range m.Optionals {
		opts = append(opts, o.Name)
	}
	return
}

func (m *Macro) isRequiredParameter(arg string) (bool, int) {
	for i, p := range m.Parameters {
		if arg == p {
			return true, i
		}
	}
	return false, 0
}

func (m *Macro) isOptionalParameter(arg string) (bool, int) {
	for i, p := range m.Optionals {
		if arg == p.Name {
			return true, i
		}
	}
	return false, 0
}

// CheckArgs returns a NodeMap of all the valid arguments or an error
// indicating why the arguments are not valid.
func (m *Macro) ValidateArgs(c *Cmd) (NodeMap, error) {
	selected, unknown, missing := c.SelectArguments(m.Parameters, m.ListOptions())
	if missing != nil {
		// Missing required arguments are fatal.
		s := ""
		if len(missing) > 1 {
			s = "s"
		}
		return nil, fmt.Errorf("Line %d: command %q is missing %d argument%s: %v",
			c.GetLineNum(), m.Name, len(missing), s, missing)
	}
	if unknown != nil {
		// Unknown arguments are fatal.
		s := ""
		if len(unknown) > 1 {
			s = "s"
		}
		return nil, fmt.Errorf("Line %d: command %q contains %d unknown argument%s: %v",
			c.GetLineNum(), m.Name, len(unknown), s, unknown)
	}
	// The arguments are valid so add any missing optionals.
	parseOptions := &Options{Plain: true}
	for _, o := range m.Optionals {
		if _, found := selected[o.Name]; !found {
			nl, _, err := Parse(o.Name, o.Default, parseOptions)
			if err != nil {
				return nil, fmt.Errorf("parsing default: %s", err)
			}
			selected[o.Name] = nl.NodeList
		}
	}
	return selected, nil
}

func (p *parser) addNewMacro(n *Cmd, flowStyle bool) error {
	name := "sys.newmacro"
	// Retrieve the sys.newmacro system command
	d := p.macros.GetMacro(name, p.format)
	if d == nil {
		return fmt.Errorf("Line %d: system command %q not defined.", n.GetLineNum(), name)
	}
	cobra.Tag("cmd").Strunc("macro", d.TemplateText).LogfV("retrieved system command definition")

	args, err := d.ValidateArgs(n)
	if err != nil {
		return fmt.Errorf("Line %d: ValidateArgs failed on system command %q: %q", n.GetLineNum(), name, err)
	}

	cobra.Tag("cmd").Strunc("syscmd", args["def"].String()).LogfV("system command: %s", args["def"])
	var mdef MacroDef

	if flowStyle {
		err = yaml.Unmarshal([]byte("{"+args["def"].String()+"}"), &mdef)
	} else {
		err = yaml.Unmarshal([]byte(args["def"].String()), &mdef)
	}

	if err != nil {
		return fmt.Errorf("Line %d: unmarshall error for system command %q: %q", n.GetLineNum(), name, err)
	}
	cobra.Tag("cmd").LogfV("marshalled syscmd: %+v", mdef)

	opts := []*Optional{}
	for _, opt := range mdef.Optionals {
		opts = append(opts, NewOptional(opt.Key.(string), opt.Value.(string)))
	}

	left, right := mdef.Delims[0], mdef.Delims[1]

	if left == "" {
		left = "(("
	}

	if right == "" {
		right = "))"
	}

	m := &Macro{
		Name:         mdef.Name,
		TemplateText: mdef.Template,
		Parameters:   mdef.Parameters,
		Optionals:    opts,
		Format:       mdef.Format,
		Block:        mdef.Block,
		Ld:           left,
		Rd:           right,
	}

	m.Parse()
	mt := MacroType{m.Name, m.Format}
	p.macros[mt] = m
	cobra.Tag("cmd").LogfV("loaded new macro")
	return nil
}

// func (p *parser) addData(n *Cmd, flowStyle bool) error {
// 	cobra.Tag("cmd").LogfV("begin addData")
// 	name := "sys.data"
// 	// Retrieve the sys.data system command
// 	d := p.macros.GetMacro(name, p.format)
// 	if d == nil {
// 		return fmt.Errorf("Line %d: system command %q not defined.", n.GetLineNum(), name)
// 	}

// 	args, err := d.ValidateArgs(n)
// 	if err != nil {
// 		return fmt.Errorf("Line %d: ValidateArgs failed on system command %q: %q", n.GetLineNum(), name, err)
// 	}

// 	cobra.Tag("cmd").Strunc("syscmd", args["data"].String()).LogfV("system command: %s", args["data"])
// 	data := make(map[interface{}]interface{})

// 	if flowStyle {
// 		err = yaml.Unmarshal([]byte("{"+args["data"].String()+"}"), data)
// 	} else {
// 		err = yaml.Unmarshal([]byte(args["data"].String()), data)
// 	}

// 	if err != nil {
// 		return fmt.Errorf("Line %d: unmarshall error for system command %q: %q", n.GetLineNum(), name, err)
// 	}

// 	for k, v := range data {
// 		Data[k.(string)] = v
// 	}

// 	cobra.Tag("cmd").LogfV("end addData")
// 	return nil
// }
