package parse

import (
	"fmt"
	"strings"
	// "unicode"
	// "unicode/utf8"
	"github.com/kevinkenan/cobra"
)

// Parse creates a node tree from the tokens produced by scan.
func Parse(name, input string, options *Options) (*Section, MacroMap, error) {
	cobra.Tag("parse").WithField("name", name).Add("plain", options.Plain).LogV("parsing input (parse)")

	p := &parser{
		scanner: scan(name, input, options.Plain),
		root:    NewSection(),
		empty:   true,
		macros:  NewMacroMap(),
		reflow:  options.Reflow,
	}

	if options.Plain {
		p.parMode = false
		p.parScanOn = false
		p.parScanFlag = false
		p.insidePar = false
	} else {
		p.parMode = true
		p.parScanOn = true
		p.parScanFlag = true
		p.insidePar = false
	}

	for _, m := range options.Macros {
		p.macros[m.Name] = m
	}

	return doParse(name, p)
}

// ParsePlain is the same as Parse but uses the scanPlain scanner.
func ParsePlain(name, input string, options *Options) (*Section, MacroMap, error) {
	return Parse(name, input, options)
}

func doParse(n string, p *parser) (*Section, MacroMap, error) {
	cobra.WithField("name", n).LogV("parsing (parse)")
	p.prevNode = p.root // Node(p.root)?
	return p.start()

}

// ----------------------------------------------------------------------------
// Parser ---------------------------------------------------------------------
// ----------------------------------------------------------------------------

type Options struct {
	Macros MacroMap
	Reflow bool
	Plain  bool
}

type pstate struct {
	sysCmd    bool
	simpleCmd bool
}

// parser represents the current state of the parser.
type parser struct {
	scanner            *scanner //
	root               *Section // Root node of the tree.
	input              string
	empty              bool   // true if the buffer is empty.
	buffer             *token // holds the next token if we peek or backup.
	prevNode           Node   // the previous node
	macros             MacroMap
	reflow             bool
	stateStack         []*pstate
	cmdDepth           int
	insideSysCmd       bool // true when we're processing a syscmd
	parMode            bool // true when the scanner is invoked with scan instead of scanPlain
	diableParScanFlags bool // when true, the scanner ignores ¶ commands
	parScanOn          bool // when true, the scanner generates paragraph commands
	parScanFlag        bool // set by ¶ command
	insidePar          bool // true if inside paragraph
	horizMode          bool // true if cmd exists within a paragraph
	blockMode          bool // true if we are currently in block mode
	blockModeChange    bool // true when the block mode has changed
}

func (p *parser) nextToken() (t *token) {
	return p.next()
}

func (p *parser) next() (t *token) {
	if p.empty {
		tt := p.scanner.nextToken()
		t = &tt      // not sure about why can't take the address of
		p.buffer = t // for backup()
	} else {
		t = p.buffer
		p.empty = true
	}
	return
}

func (p *parser) peek() (t *token) {
	if p.empty {
		tt := p.scanner.nextToken()
		t = &tt
		p.buffer = t
	} else {
		t = p.buffer
	}
	p.empty = false
	return
}

// backup reverts the last call to next(). Repeated calls to backup() have no
// effect.
func (p *parser) backup() {
	p.empty = false
}

func (p *parser) nextIf(ttype tokenType) (t *token) {
	if t = p.next(); t.typeof == ttype {
		return
	}
	p.errorf("found %q instead of %q", t.value, tokenTypeLookup(ttype))
	return
}

func (p *parser) linkNodeList(nodes NodeList) {
	for _, n := range nodes {
		cobra.Tag("link").WithField("details", n.Details()).Log("linking nodelist node")
		p.prevNode.SetNext(n)
		n.SetPrev(p.prevNode)
		p.prevNode = n
	}
}

func (p *parser) link(n Node) {
	cobra.Tag("link").WithField("details", n.Details()).Log("linking node")
	p.prevNode.SetNext(n)
	n.SetPrev(p.prevNode)
	p.prevNode = n
}

func (p *parser) isParScanAllowed() bool {
	return p.parMode && p.parScanFlag
}

func (p *parser) pushState(s *pstate) {
	p.stateStack = append(p.stateStack, s)
}

func (p *parser) popState() *pstate {
	l := len(p.stateStack)
	if l == 0 {
		p.errorf("attempted to read past the end of the parse stack")
	}
	s := p.stateStack[l-1]
	p.stateStack = p.stateStack[:l-1]
	return s
}

func appendNode(nl NodeList, ns ...Node) NodeList {
	for _, n := range ns {
		nl = append(nl, n)
	}
	return nl
}

// Parse token stream ---------------------------------------------------------

func (p *parser) start() (n *Section, macs MacroMap, err error) {
	defer p.recover(&err)
	cobra.Tag("parse").LogV("parse start")
	for {
		nl, done, err := p.parseText()
		if err != nil {
			return nil, nil, err
		}
		p.root.append(nl)
		if done {
			break
		}
	}
	return p.root, p.macros, nil
}

func (p *parser) parseText() (nl NodeList, done bool, err error) {
	cobra.Tag("parse").Add("cmdDepth", p.cmdDepth).LogV("parseText")
	nl = NodeList{}
	done = false
	for {
		t := p.next()
		switch t.typeof {
		case tokenSpaceEater:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
		spaceEater:
			for {
				switch p.next().typeof {
				case tokenEmptyLine, tokenIndent, tokenLineBreak, tokenSpaceEater:
					cobra.Tag("parse").LogV("eating space")
					continue
				default:
					p.backup()
					break spaceEater
				}
			}
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenEmptyLine:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			if !p.parScanOn {
				n, _ := p.makeTextNode(t)
				// p.root.append(n)
				nl = appendNode(nl, n)
				p.link(n)
			}
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenIndent:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			if !p.reflow {
				n, _ := p.makeTextNode(t)
				// p.root.append(n)
				nl = appendNode(nl, n)
				p.link(n)
			}
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenLineBreak:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			if p.parScanOn {
				pn := p.parseParagraph(t)
				nl = appendNode(nl, pn...)
			} else {
				n, _ := p.makeTextNode(t)
				nl = appendNode(nl, n)
				p.link(n)
			}
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenText:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			n, l := p.makeTextNode(t)
			if p.parScanOn && !p.insidePar && l > 0 {
				p.insidePar = true
				nl = appendNode(nl, NewParBeginNode(nil))
				// fmt.Println(nl)
			}
			nl = appendNode(nl, n)
			p.link(n)
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenCmdStart:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			n, err := p.makeCmd(t)
			if err != nil {
				return nil, true, err
			}

			if p.parScanOn && !p.insidePar && !n.Block {
				p.insidePar = true
				nl = appendNode(nl, NewParBeginNode(t))
			} else if p.parScanOn && p.insidePar && n.Block {
				p.insidePar = false
				nl = appendNode(nl, NewParEndNode(t))
			}

			p.parseCmd(n)
			nl = appendNode(nl, n)

			if n.Block && p.parScanOn {
				p.blockSpaceEater()
			}

			p.link(n)
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenSysCmdStart:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			p.insideSysCmd = true

			n, err := p.makeCmd(t)
			if err != nil {
				return nil, true, err
			}
			p.parseCmd(n)

			switch n.GetCmdName() {
			case "sys.newmacrof":
				err = p.addNewMacro(n, true)
			case "sys.newmacro":
				err = p.addNewMacro(n, false)
			default:
				// n.NodeValue = NodeValue(fmt.Sprintf("sys.%s", n.NodeValue))
				// p.root.append(n)
				nl = append(nl, n)
				p.link(n)
			}

			p.insideSysCmd = false
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenRightCurly:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			if p.cmdDepth > 0 {
				cobra.Tag("parse").LogV("finished cmd text block")
				p.backup()
				cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
				return
			}
		case tokenRightSquare:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			if p.cmdDepth > 0 {
				cobra.Tag("parse").LogV("finished full command text block")
				p.backup()
				return
			}
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenError:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			p.errorf("Line %d: %s", t.lnum, t.value)
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
		case tokenEOF:
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
			if p.parScanOn && p.insidePar {
				// p.root.append(NewParEndNode(t))
				nl = append(nl, NewParEndNode(t))
			}
			done = true
			cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
			return
		default:
			p.errorf("Line %d: unexpected token %q when starting with value %q", t.lnum, tokenTypeLookup(t.typeof), t.value)
		}
	}
	cobra.Tag("parse").LogV("This should be impossilbe")
	return
}

func (p *parser) blockSpaceEater() {
loop:
	for {
		nxt := p.next()
		cobra.Tag("parse").Strunc("text", nxt.value).LogV("post block text")
		switch nxt.typeof {
		case tokenEmptyLine, tokenIndent, tokenLineBreak, tokenSpaceEater:
			cobra.Tag("parse").LogV("eating space")
			continue
		case tokenText:
			if len(strings.TrimSpace(nxt.value)) == 0 {
				continue
			}
			fallthrough
		default:
			p.backup()
			break loop
		}
	}
}

func (p *parser) parseParagraph(t *token) (nl NodeList) {
	nl = NodeList{}
	// pkt := p.peek().typeof
	lb := false

	// Eat empty space
loop:
	for {
		switch p.peek().typeof {
		case tokenLineBreak, tokenEmptyLine:
			n := p.next().typeof
			if n == tokenLineBreak {
				lb = true
				break loop
				// c = false
			}
		case tokenEOF:
			lb = true
			break loop
			// c = false
		default:
			break loop
		}
	}

	if p.insidePar {
		if lb {
			// p.root.append(NewParEndNode(t))
			cobra.Tag("parse").LogV("adding paragraph.end")
			nl = append(nl, NewParEndNode(t))
			p.insidePar = false
		} else {
			if p.reflow {
				// p.root.append(NewTextNode(" "))
				cobra.Tag("parse").LogV("adding reflow text node")
				nl = append(nl, NewTextNode(" "))
			} else {
				// p.root.append(NewTextNode("\n"))
				cobra.Tag("parse").LogV("adding text node")
				nl = append(nl, NewTextNode("\n"))
			}
		}
	}

	if p.reflow {
		// NewTextNode(strings.Replace(t.value, "\n", " ", -1))
	}

	return
}

func (p *parser) makeTextNode(t *token) (*Text, int) {
	l := len(t.value)
	s := t.value
	cobra.Tag("parse").LogV("creating a text node")
	if p.reflow {
		s = strings.TrimRight(s, " \t")
		switch p.peek().typeof {
		case tokenLineBreak, tokenEOF, tokenRightCurly, tokenRightSquare:
		default:
			if len(s) < l {
				s = s + " "
			}
		}
	}
	n := NewTextNode(s)
	return n, len(s)
}

func (p *parser) makeCmd(t *token) (n *Cmd, err error) {
	// p.cmdDepth += 1
	cobra.Tag("parse").LogV("creating a cmd node")
	n = NewCmdNode(p.nextIf(tokenName).value, t)
	name := n.GetCmdName()
	cobra.WithField("name", name).LogV("parsing command (cmd)")

	mac, found := p.macros[name]
	if !found {
		err = fmt.Errorf("Line %d: command %q not defined.", n.GetLineNum(), name)
		return nil, err
	}

	n.Block = mac.Block
	return
}

func (p *parser) parseCmd(n *Cmd) {
	switch p.peek().typeof {
	case tokenLeftSquare:
		p.parseCmdContext(n)
	case tokenLeftCurly:
		p.parseSimpleCmd(n)
	}
	return
}

func (p *parser) parseSimpleCmd(m *Cmd) {
	cobra.Tag("parse").LogV("parsing a simple cmd")
	var nl NodeList
	var err error
	// m.ArgList = []NodeList{p.parseTextBlock(m)}
	p.cmdDepth += 1
	p.nextIf(tokenLeftCurly)

	parScanState := p.parScanOn
	if m.Block {
		p.parScanOn = false
	}

	if m.SysCmd {
		nl = p.assembleText()
	} else {
		nl, _, err = p.parseText()
		if err != nil {
			panic(fmt.Errorf(err.Error()))
		}
	}

	p.parScanOn = parScanState
	m.ArgList = []NodeList{nl}
	p.nextIf(tokenRightCurly)
	p.cmdDepth -= 1
	return
}

func (p *parser) assembleText() NodeList {
	w := new(strings.Builder)
	for {
		t := p.next()
		switch t.typeof {
		case tokenRightCurly:
			p.backup()
			return NodeList{NewTextNode(w.String())}
		default:
			w.WriteString(t.value)
		}
	}
}

// func (p *parser) parseSysCmd(m *Cmd) {
// 	cobra.Tag("parse").LogV("parsing syscmd")
// 	m.ArgList = []NodeList{p.parseTextBlock(m)}
// 	return
// }

// func (p *parser) parseSysCmd(m *Cmd) {
// 	cobra.Tag("parse").LogV("parsing syscmd")
// 	p.next()
// 	// m.ArgList = []NodeList{p.parseTextBlock(m)}
// 	p.cmdDepth += 1
// 	for {
// 		t := p.next()
// 		switch t.typeof {
// 		case tokenRightCurly:

// 		}
// 	}
// 	p.cmdDepth -= 1
// 	m.ArgList = []NodeList{nl}
// 	return
// }

func (p *parser) parseCmdContext(m *Cmd) {
	cobra.Tag("parse").LogV("parsing cmd context")
	t := p.nextIf(tokenLeftSquare)
	t = p.peek()

	if t.typeof == tokenLeftAngle {
		p.parseCmdFlags(m)
	}

	parScanState := p.parScanOn
	if m.Block {
		p.parScanOn = false
	}

	p.cmdDepth += 1

	t = p.peek()
	switch t.typeof {
	case tokenName:
		m.Anonymous = false
		p.parseNamedArgs(m)
	case tokenLeftCurly:
		m.Anonymous = true
		p.parsePostionalArgs(m)
	}

	p.parScanOn = parScanState
	p.nextIf(tokenRightSquare)
	p.cmdDepth -= 1

	// t = p.next()
	// switch t.typeof {
	// case tokenRightSquare:
	// 	return
	// default:
	// 	p.errorf("unexpected %q in a command context on line %d", t.value, t.lnum)
	// }
	return
}

func (p *parser) parseNamedArgs(m *Cmd) {
	pMap := make(NodeMap)
	var nl NodeList
	var err error

	for {
		t := p.nextIf(tokenName)
		argName := t.value
		cobra.Tag("parse").WithField("arg", argName).LogV("parsing named args")
		p.nextIf(tokenEqual)
		t = p.nextIf(tokenLeftCurly)

		if m.SysCmd {
			nl = p.assembleText()
		} else {
			nl, _, err = p.parseText()
			if err != nil {
				panic(fmt.Errorf(err.Error()))
			}
		}

		pMap[argName] = nl
		p.nextIf(tokenRightCurly)

		t = p.peek()
		if t.typeof == tokenRightSquare {
			m.ArgMap = pMap
			return
		}
	}

	return
}

func (p *parser) parsePostionalArgs(m *Cmd) {
	var nl NodeList
	var err error
	for {
		p.nextIf(tokenLeftCurly)

		if m.SysCmd {
			nl = p.assembleText()
		} else {
			nl, _, err = p.parseText()
			if err != nil {
				panic(fmt.Errorf(err.Error()))
			}
		}

		m.ArgList = append(m.ArgList, nl)
		p.linkNodeList(nl)
		p.nextIf(tokenRightCurly)

		t := p.peek()
		if t.typeof == tokenRightSquare {
			return
		}
	}
	return
}

func (p *parser) parseCmdFlags(m *Cmd) {
	t := p.nextIf(tokenLeftAngle)
	for {
		t = p.next()
		switch t.typeof {
		case tokenRunes:
			cobra.Tag("parse").WithField("flag", t.value).LogV("parsing cmd flags")
			m.Flags = append(m.Flags, t.value)
		case tokenComma:
			continue
		case tokenRightAngle:
			return
		default:
			p.errorf("unexpected %q in command flags on line %d", t.value, t.lnum)
		}
	}
	return
}

func (p *parser) errorf(format string, args ...interface{}) {
	p.root = nil
	// format = fmt.Sprintf("template: %s:%d: %s", t.ParseName, t.token[0].line, format)
	panic(Error(fmt.Sprintf(format, args...)))
}

func (p *parser) recover(errk *error) {
	if e := recover(); e != nil {
		*errk = e.(Error)
	}
}

type Error string

func (e Error) Error() string {
	return string(e)
}
