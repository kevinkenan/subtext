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
	"sort"
	"strings"

	"github.com/kevinkenan/cobra"
)

// ----------------------------------------------------------------------------
// Miscellaneous Types --------------------------------------------------------
// ----------------------------------------------------------------------------

// Node Interface -------------------------------------------------------------

type Node interface {
	Typeof() NodeType
	String() string
	Count() int
	Details() string
	Ahead() Node
	Behind() Node
	SetNext(Node)
	SetPrev(Node)
}

// NodeType -------------------------------------------------------------------

type NodeType int

func (t *NodeType) Typeof() NodeType {
	return *t
}

// NodeValue -------------------------------------------------------------------

// NodeValue represents the value of the node. The meaning of the value
// depends on the node type.
type NodeValue string

// String includes the value of the node along with the values of any child
// nodes.
func (nt NodeValue) String() string {
	return string(nt)
}

// Peek -----------------------------------------------------------------------

// TODO: Remove Peek as it is no longer used.

type Peek struct {
	next Node
	prev Node
}

func (p *Peek) Ahead() Node {
	return p.next
}

func (p *Peek) SetNext(n Node) {
	p.next = n
}

func (p *Peek) Behind() Node {
	return p.prev
}

func (p *Peek) SetPrev(n Node) {
	p.prev = n
}

// NodeList -------------------------------------------------------------------

type NodeList []Node

func (nl NodeList) String() string {
	b := new(strings.Builder)
	for _, n := range nl {
		b.WriteString(n.String())
	}
	return b.String()
}

// func (nl NodeList) PluralStr() string {
// 	return nl.PluralStrTerm("argument", "s")
// }

// func (nl NodeList) PluralStrTerm(term, suffix string) string {
// 	if len(nl) != 1 {
// 		return fmt.Sprintf("%d %ss", len(nl), term)
// 	}
// 	return fmt.Sprintf("%d %s%s", len(nl), term, suffix)
// }

func (nl NodeList) Count() (c int) {
	c = 0
	for _, n := range nl {
		c += n.Count()
	}
	return
}

// NodeMap --------------------------------------------------------------------

type NodeMap map[string]NodeList

func (nm NodeMap) String() string {
	b := new(strings.Builder)
	for _, n := range nm {
		b.WriteString(n.String())
	}
	return b.String()
}

// func (nm NodeMap) PluralStr() string {
// 	return nm.PluralStrTerm("argument", "s")
// }

// func (nm NodeMap) PluralStrTerm(term, suffix string) string {
// 	if len(nm) != 1 {
// 		return fmt.Sprintf("%d %ss", len(nm), term)
// 	}
// 	return fmt.Sprintf("%d %s%s", len(nm), term, suffix)
// }

// func (nm NodeMap) ListStr() string {
// 	keys := []string{}
// 	for k := range nm {
// 		keys = append(keys, k)
// 	}
// 	list := strings.Join(keys[:len(keys)-1], ", ")
// 	return fmt.Sprintf("%s and %s", list, keys[len(keys)-1])
// }

func (nm NodeMap) Keys() (keys []string) {
	keys = []string{}
	for k := range nm {
		keys = append(keys, k)
	}
	return
}

func (nm NodeMap) Count() (c int) {
	c = 0
	for _, n := range nm {
		c += n.Count()
	}
	return
}

// ----------------------------------------------------------------------------
// Walkable Interface ---------------------------------------------------------
// ----------------------------------------------------------------------------

type Walkable interface {
	Walk(chan Node)
	walk(chan Node)
}

// ----------------------------------------------------------------------------
// Nodes ----------------------------------------------------------------------
// ----------------------------------------------------------------------------

// Node types
const (
	nSection        NodeType = iota // A list of nodes.
	nText                           // Text to be printed as is.
	nSysCmd                         // A system command.
	nParagraphStart                 // Start a new paragraph.
	nParagraphEnd                   // End the current paragraph.
	nCmd                            // Invokes a command
	nEof                            // The end of the input text.
	nError                          // An error occurred.
)

// Section --------------------------------------------------------------------

type Section struct {
	NodeType
	NodeValue
	NodeList
	Peek
}

func NewSection() *Section {
	cobra.Tag("node").LogV("section")
	return &Section{
		NodeType: nSection,
		NodeList: []Node{},
	}
}

func (nl *Section) Details() string {
	return fmt.Sprintf("section node containing %d nodes", len(nl.NodeList))
}

func (nl *Section) String() string {
	b := new(strings.Builder)
	for _, n := range nl.NodeList {
		b.WriteString(n.String())
	}
	return b.String()
}

func (s *Section) append(nl NodeList) {
	for _, n := range nl {
		s.NodeList = append(s.NodeList, n)
	}
}

func (sec *Section) Count() (c int) {
	c = 1
	for _, n := range sec.NodeList {
		c += n.Count()
	}
	return
}

func (nl *Section) WalkS(cs chan string) {
	c := make(chan Node)
	go nl.Walk(c)
	w := new(strings.Builder)
	for n := range c {
		switch n.(type) {
		case *Text:
			w.WriteString(fmt.Sprintf("Text Node: %q", n.(*Text).NodeValue))
		case *Section:
			w.WriteString(fmt.Sprintf("Section Node: contains %d nodes", len(n.(*Section).NodeList)))
		case *ErrorNode:
			w.WriteString(fmt.Sprintf("Error: %q", n.(*ErrorNode).NodeValue))
		case *Cmd:
			w.WriteString(fmt.Sprintf("Cmd Node: %q\n", n.(*Cmd).NodeValue))
			w.WriteString(fmt.Sprintf("         Args: %d\n", n.(*Cmd).Count()-1))
			w.WriteString(fmt.Sprintf("         Flags: <%s>\n", strings.Join(n.(*Cmd).Flags, ",")))
			// for _, f := range n.(*Cmd).Flags {
			// 	w.WriteString(fmt.Sprintf("%s,", f))
			// }
			// w.WriteString(fmt.Sprintln(">"))
			w.WriteString(fmt.Sprintf("         Anonymous: %t", n.(*Cmd).Anonymous))
			if n.(*Cmd).Anonymous {
				for i, nl := range n.(*Cmd).ArgList {
					w.WriteString(fmt.Sprintf("\n         Arg %d:", i))
					for _, nn := range nl {
						w.WriteString(fmt.Sprintf(" %q", nn))
					}
				}
			} else {
				if len(n.(*Cmd).ArgMap) > 0 {
					for k, v := range n.(*Cmd).ArgMap {
						w.WriteString(fmt.Sprintf("\n         Arg %q: %s", k, v))
					}
				} else {
					w.WriteString(fmt.Sprintln("\n         Args: None"))
				}
			}
		default:
			w.WriteString(fmt.Sprintf("> UNEXPECTED Node: %q\n", n.String()))
			w.WriteString(fmt.Sprintf("     Type Code: %d\n", n.Typeof()))
		}
		cs <- w.String()
		w.Reset()
	}
	close(cs)
}

func (nl *Section) Walk(c chan Node) {
	nl.walk(c)
	close(c)
}

func (nl *Section) walk(c chan Node) {
	for _, n := range nl.NodeList {
		c <- n
		switch n.(type) {
		case Walkable:
			n.(Walkable).walk(c)
		}
	}
}

func (nl *Section) OnExit(n *Section) {}

// Text Node ------------------------------------------------------------------

type Text struct {
	NodeType
	NodeValue
	Peek
}

func NewTextNode(t string) *Text {
	cobra.Tag("node").Strunc("val", t).LogfV("text")
	return &Text{NodeType: nText, NodeValue: NodeValue(t)}
}

func (t *Text) Details() string {
	return fmt.Sprintf("text node containing %d characters", len(t.NodeValue))
}

func (t *Text) Count() int {
	return 1
}

func (t *Text) GetText() string {
	return string(t.NodeValue)
}

// Paragraph Nodes ------------------------------------------------------------

// type ParagraphStart struct {
// 	NodeType
// 	NodeValue
// }

func NewParBeginNode(t *token) *Cmd {
	cobra.Tag("node").LogV("paragraph begin")
	return &Cmd{
		NodeType:  nCmd,
		NodeValue: NodeValue("paragraph.begin"),
		cmdToken:  t,
	}
}

func NewParEndNode(t *token) *Cmd {
	cobra.Tag("node").LogV("paragraph end")
	return &Cmd{
		NodeType:  nCmd,
		NodeValue: NodeValue("paragraph.end"),
		cmdToken:  t,
	}
}

// func (p *ParagraphStart) String() string {
// 	return "•paragraph.begin[]"
// }

// func (p *ParagraphStart) Count() int {
// 	return 1
// }

// type ParagraphEnd struct {
// 	NodeType
// 	NodeValue
// }

// func NewParagraphEnd(t string) *ParagraphEnd {
// 	cobra.Tag("node").LogV("paragraph end")
// 	return &ParagraphEnd{nParagraphEnd, NodeValue(t)}
// }

// func (p *ParagraphEnd) String() string {
// 	return "•paragraph.end[]"
// }

// func (p *ParagraphEnd) Count() int {
// 	return 1
// }

// SysCmd Node ------------------------------------------------------------------

type SysCmd struct {
	NodeType
	NodeValue
	Arguments
	Peek
}

func NewSysCmdNode(name, t string) *SysCmd {
	cobra.Tag("node").LogV("syscmd")
	return &SysCmd{
		NodeType:  nSysCmd,
		NodeValue: NodeValue(name),
		Arguments: Arguments{true, []NodeList{}, nil}}
}

func (t *SysCmd) Details() string {
	return fmt.Sprintf("syscmd node for %s", t.NodeValue)
}

func (t *SysCmd) Count() int {
	return 1
}

func (t *SysCmd) GetText() string {
	return string(t.NodeValue)
}

func (t *SysCmd) GetCommand() (string, string) {
	cmd := strings.SplitN(t.NodeValue.String(), "=", 2)
	switch len(cmd) {
	case 1:
		return cmd[0], ""
	case 2:
		return cmd[0], cmd[1]
	default:
		return "", ""
	}
}

func (t *SysCmd) String() string {
	w := new(strings.Builder)
	w.WriteString("•")
	w.WriteString("(")
	a, b := t.GetCommand()
	w.WriteString(a)
	if b != "" {
		w.WriteString("=")
	}
	w.WriteString(b)
	w.WriteString(")")
	return w.String()
}

// Command Node -----------------------------------------------------------------

type Cmd struct {
	NodeType  // this will be the nodeCommand const
	NodeValue // the command's name
	Arguments
	Flags    []string
	cmdToken *token
	Peek
	SysCmd bool   // true if the command is a system command.
	Block  bool   // true if the command is a block
	Series bool   // true if the command is a serial command
	Format string // the parse format at the time the cmd was created
}

type Arguments struct {
	Anonymous bool       // true indicates ArgList is set, otherwise ArgMap is set.
	ArgList   []NodeList // list of anonymous arguments.
	ArgMap    NodeMap    // map of key/value arguments.
}

func NewCmdNode(name string, t *token) *Cmd {
	syscmd := t.typeof == tokenSysCmdStart
	if syscmd {
		name = "sys." + name
	}
	cobra.Tag("node").LogV("cmd")
	return &Cmd{
		NodeType:  nCmd,
		NodeValue: NodeValue(name),
		// NodeList{},
		Arguments: Arguments{true, []NodeList{}, nil},
		Flags:     []string{},
		cmdToken:  t,
		SysCmd:    syscmd,
	}
}

func (c *Cmd) Details() string {
	return fmt.Sprintf("cmd node for %q containing %d nodes", c.NodeValue, len(c.ArgList)+len(c.ArgMap))
}

func (c *Cmd) GetTokenValue() string {
	return c.cmdToken.value
}

func (c *Cmd) HasFlag(s string) bool {
	for _, f := range c.Flags {
		if f == s {
			return true
		}
	}
	return false
}

func (c *Cmd) HasFlagVar(s string) (string, bool) {
	s = s + "="
	for _, f := range c.Flags {
		if strings.HasPrefix(f, s) {
			return strings.TrimPrefix(f, s), true
		}
	}
	return "", false
}

// SelectArguments returns a map of the command's arguments which match the
// function's parameter arguments. Arguments that are not required or optional
// are returned in the 'unknown' slice. If the arguments don't include a
// required parameter, the parameter is listed in the 'missing' slice.
func (cmd *Cmd) SelectArguments(reqParams, optParams []string) (selected NodeMap, unknown, missing []string) {
	if cmd.Anonymous {
		selected, unknown, missing = cmd.selectAnonymousArguments(reqParams, optParams)
	} else {
		selected, unknown, missing = cmd.selectNamedArguments(reqParams, optParams)
	}
	return
}

// selecteAnonymousArguments assigns arguments to parameters in the order
// specifed by the reqParams argument. If we have more args than reqParams,
// the remaining args are assigned to optionals in the order specified by
// optParams. For instance, if parameters is ["alpha", "beta"] then the first
// element of the ArgList will use the key "alpha" and the second element will
// use the key "beta".
func (cmd *Cmd) selectAnonymousArguments(reqParams, optParams []string) (NodeMap, []string, []string) {
	cobra.Tag("cmd").WithField("name", cmd.NodeValue).LogV("selecting anonymous arguments")
	selected := NodeMap{}

	if len(cmd.ArgList) < len(reqParams) {
		// Not enough args to satisfy required parameters.
		cobra.Tag("cmd").WithField("args", len(cmd.ArgList)).Add("params", len(reqParams)).LogV("fewer args than parameters")
		missing := []string{}

		for i := len(cmd.ArgList); i < len(reqParams); i++ {
			missing = append(missing, reqParams[i])
		}

		return nil, nil, missing
	}

	if len(cmd.ArgList) > len(reqParams)+len(optParams) {
		// We have too many arguments.
		cobra.Tag("cmd").WithField("args", len(cmd.ArgList)).Add("params", len(reqParams)).LogV("more args than parameters")
		unknown := []string{}

		for i := len(reqParams) + len(optParams); i < len(cmd.ArgList); i++ {
			unknown = append(unknown, fmt.Sprintf("#%d", i+1))
		}

		return nil, unknown, nil
	}

	// If we're here, then the number of args is valid.
	for i, p := range reqParams {
		// Add the required arguments.
		selected[p] = cmd.ArgList[i]
	}

	for i, j := len(reqParams), 0; i < len(cmd.ArgList); i, j = i+1, j+1 {
		// Add the optional arguments.
		selected[optParams[j]] = cmd.ArgList[i]
	}

	cobra.Tag("cmd").WithField("selected", len(selected)).LogV("valid number of args")
	return selected, nil, nil
}

// selecteNamedArguments examines the named arguments to see if they match
// required or optional parameters.
func (cmd *Cmd) selectNamedArguments(reqParams, optParams []string) (NodeMap, []string, []string) {
	selected := make(map[string]NodeList)
	unknown, missing := []string{}, []string{}
	for _, p := range reqParams {
		arg, ok := cmd.ArgMap[p]
		if ok {
			// We have an arg that matches a required parameter.
			selected[p] = arg
		} else {
			// We have a parameter with no matching arg.
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return nil, nil, missing
	}
	if len(selected) == len(cmd.ArgMap) {
		// All the args are required and no parameters are missing.
		return selected, nil, nil
	}
	// If we're here, we must have extra args so check if they are optional.
	for k, v := range cmd.ArgMap {
		if _, found := selected[k]; found {
			// Remove required args that have already been selected.
			continue
		}
		found := false
		for _, p := range optParams {
			if k == p {
				// The arg matches a valid optional parameter.
				found = true
				break
			}
		}
		if found {
			selected[k] = v
		} else {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return nil, unknown, nil
	}
	return selected, nil, nil
}

func (m *Cmd) GetLineNum() int {
	return m.cmdToken.lnum
}

func (m *Cmd) GetCmdName() string {
	return string(m.NodeValue)
}

func (m *Cmd) addArgument(n NodeList) {
	m.ArgList = append(m.ArgList, n)
}

func (m *Cmd) setArgumentList(p []NodeList) {
	m.Anonymous = true
	m.ArgList = p
}

func (m *Cmd) setArgumentMap(p map[string]NodeList) {
	m.Anonymous = false
	m.ArgMap = p
}

// String returns a textual representation of the command. This is only useful
// in tests or while debugging.
func (n *Cmd) String() string {
	w := new(strings.Builder)
	w.WriteString("•")
	// Write the Cmd's name.
	w.WriteString(n.NodeValue.String())
	// Begin the context
	w.WriteString("[")
	// Write the flags.
	if len(n.Flags) > 0 {
		w.WriteString("<")
		w.WriteString(strings.Join(n.Flags, ","))
		w.WriteString(">")
	}
	// Write the Arguments
	if n.Anonymous {
		for _, nl := range n.ArgList {
			w.WriteString(fmt.Sprintf("{%s}", nl.String()))
		}
		// if len(n.ArgList) == 0 {
		// 	w.WriteString("{}")
		// }
	} else {
		// Not anonymous. We sort the map to guarantee the order for unit
		// tests.
		var keys []string
		for k := range n.ArgMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := n.ArgMap[k]
			w.WriteString(fmt.Sprintf("%s={%s}", k, v.String()))
		}
	}
	w.WriteString("]")
	return w.String()
}

func (m *Cmd) Count() (c int) {
	c = 1
	if m.Anonymous && m.ArgList != nil {
		for _, nl := range m.ArgList {
			c += nl.Count()
		}
	} else if m.ArgMap != nil {
		for _, nl := range m.ArgMap {
			c += nl.Count()
		}
	}
	return
}

// Error Node -----------------------------------------------------------------

type ErrorNode struct {
	NodeType
	NodeValue
	Peek
}

func NewErrorNode(t string) *ErrorNode {
	cobra.Tag("node").LogV("error")
	return &ErrorNode{NodeType: nError, NodeValue: NodeValue(t)}
}

func (e *ErrorNode) Details() string {
	return fmt.Sprintf("error node: %s", e.NodeValue)
}

func (t *ErrorNode) Count() int {
	return 1
}

func (t *ErrorNode) GetErrorMsg() string {
	return string(t.NodeValue)
}
