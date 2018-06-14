package macros

import (
	// "errors"
	"fmt"
	"strings"
	"text/template"
	// "unicode"
	// "unicode/utf8"
	"github.com/kevinkenan/subtext/parse"
	// "github.com/kevinkenan/cobra"
)

type Macro struct {
	Name               string      // The macro's name to match command names
	TemplateText       string      // The Go template that defines the macro
	*template.Template             // the parsed template
	Parameters         []string    // Required parameters
	Optionals          []*Optional // Optional parameters in correct order
	Ld                 string      // Left delim used in the template
	Rd                 string      // Right delim used in the template
}

type Optional struct {
	Name    string
	Default string
}

func NewMacro(name, tmplt string, params []string, optionals []*Optional) *Macro {
	t := template.Must(template.New(name).Option("missingkey=error").Parse(tmplt))
	return &Macro{
		Name:         name,
		Parameters:   params,
		Optionals:    optionals,
		TemplateText: tmplt,
		Template:     t,
		Ld:           "{{",
		Rd:           "}}"}
}

func NewOptional(name, dflt string) *Optional {
	return &Optional{name, dflt}
}

func (m *Macro) String() string {
	w := new(strings.Builder)
	// w.WriteString("\n")
	w.WriteString(fmt.Sprintf("Name %s, ", m.Name))
	w.WriteString(fmt.Sprintf("  Template %s, ", m.TemplateText))
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
func (m *Macro) ValidateArgs(c *parse.Cmd) (parse.NodeMap, error) {
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
	for _, o := range m.Optionals {
		if _, found := selected[o.Name]; !found {
			nl, err := parse.ParsePlain(o.Name, o.Default)
			if err != nil {
				return nil, fmt.Errorf("parsing default: %s", err)
			}
			selected[o.Name] = nl.NodeList
		}
	}
	return selected, nil
}
