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

// ParseMacro is the same as Parse bu
func ParseMacro(name, input string) (*Section, MacroMap, error) {
	options := &Options{
		Macros: *new(MacroMap),
		Plain:  true,
	}
	return Parse(name, input, options)
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
	sysCmd bool
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

// Parse token stream ---------------------------------------------------------

func (p *parser) start() (n *Section, macs MacroMap, err error) {
	defer p.recover(&err)
	cobra.Tag("parse").LogV("parse root level nodes")
Loop:
	for {
		t := p.next()
		switch t.typeof {
		case tokenSpaceEater:
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
		case tokenEmptyLine:
			if p.parScanOn {
				continue
			}
			n := p.makeTextNode(t)
			p.root.append(n)
			p.link(n)
		case tokenIndent:
			if p.reflow {
				continue
			} else {
				n := p.makeTextNode(t)
				p.root.append(n)
				p.link(n)
			}
		case tokenLineBreak:
			if p.parScanOn {
				p.parseParagraph(t)
			} else {
				n := p.makeTextNode(t)
				p.root.append(n)
				p.link(n)
			}
		case tokenText:
			if p.parScanOn && !p.insidePar {
				p.insidePar = true
				p.root.append(NewParBeginNode(t))
			}
			n := p.makeTextNode(t)
			p.root.append(n)
			p.link(n)
		case tokenCmdStart:
			n := p.makeCmd(t)
			p.root.append(n)
			p.link(n)
		case tokenSysCmdStart:
			cobra.Tag("parse").LogV("begin tokenSysCmdStart")
			p.insideSysCmd = true

			n := p.makeCmd(t)
			n.SysCmd = true
			n.NodeValue = NodeValue(fmt.Sprintf("sys.%s", n.NodeValue))

			switch n.GetCmdName() {
			case "sys.newmacrof":
				err = p.addNewMacro(n, true)
			case "sys.newmacro":
				err = p.addNewMacro(n, false)
			default:
				p.root.append(n)
				p.link(n)
			}

			p.insideSysCmd = false
			cobra.Tag("parse").LogV("end tokenSysCmdStart")
		case tokenError:
			p.errorf("Line %d: %s", t.lnum, t.value)
		case tokenEOF:
			if p.parScanOn && p.insidePar {
				p.root.append(NewParEndNode(t))
			}
			break Loop
		default:
			p.errorf("Line %d: unexpected token %q when starting with value %q", t.lnum, tokenTypeLookup(t.typeof), t.value)
		}
		if err != nil {
			return nil, nil, err
		}
	}
	return p.root, p.macros, nil
}

func (p *parser) parseParagraph(t *token) {
	pkt := p.peek().typeof
	lb := false

	switch pkt {
	case tokenLineBreak, tokenEmptyLine:
		for n := p.next().typeof; n == tokenLineBreak || n == tokenEmptyLine; n = p.next().typeof {
			if n == tokenLineBreak {
				lb = true
			}
		}
		p.backup()
	}

	if p.insidePar {
		if lb {
			p.root.append(NewParEndNode(t))
			p.insidePar = false
		} else {
			if p.reflow {
				p.root.append(NewTextNode(" "))
			} else {
				p.root.append(NewTextNode("\n"))
			}
		}
	}

	if p.reflow {
		NewTextNode(strings.Replace(t.value, "\n", " ", -1))
	}
}

func (p *parser) makeTextNode(t *token) (n *Text) {
	cobra.Tag("parse").LogV("creating a text node")
	n = NewTextNode(t.value)
	return
}

func (p *parser) makeCmd(t *token) (n *Cmd) {
	// p.cmdDepth += 1
	cobra.Tag("parse").LogV("creating a cmd node")
	n = NewCmdNode(p.nextIf(tokenName).value, t)
	name := n.GetCmdName()

	// if strings.HasPrefix(name, "sys.paragraph") && p.inBlockMode {
	// 	return
	// }

	// m, ok := p.macros[name]
	// if ok {
	// 	p.inBlockMode = m.Block
	// }

	// if p.inPar && p.inBlockMode {

	// }

	cobra.WithField("name", name).LogV("parsing command (cmd)")

	switch p.peek().typeof {
	case tokenLeftSquare:
		p.parseCmdContext(n)
	case tokenLeftCurly:
		p.parseSimpleCmd(n)
	default:
		return
	}

	return
}

func (p *parser) parseSimpleCmd(m *Cmd) {
	cobra.Tag("parse").LogV("parsing a simple cmd")
	m.ArgList = []NodeList{p.parseTextBlock(m)}
	return
}

func (p *parser) parseSysCmd(m *Cmd) {
	cobra.Tag("parse").LogV("parsing syscmd")
	m.ArgList = []NodeList{p.parseTextBlock(m)}
	return
}

func (p *parser) parseCmdContext(m *Cmd) {
	cobra.Tag("parse").LogV("parsing cmd context")
	t := p.nextIf(tokenLeftSquare)
	t = p.peek()
	if t.typeof == tokenLeftAngle {
		p.parseCmdFlags(m)
	}
	t = p.peek()
	switch t.typeof {
	case tokenName:
		m.Anonymous = false
		p.parseNamedArgs(m)
	case tokenLeftCurly:
		m.Anonymous = true
		p.parsePostionalArgs(m)
	}
	t = p.next()
	switch t.typeof {
	case tokenRightSquare:
		return
	default:
		p.errorf("unexpected %q in a command context on line %d", t.value, t.lnum)
	}
	return
}

func (p *parser) parseNamedArgs(m *Cmd) {
	pMap := make(NodeMap)
	var nl NodeList
	for {
		t := p.next()
		switch t.typeof {
		case tokenName:
			argName := t.value
			cobra.Tag("parse").WithField("arg", argName).LogV("parsing named args")
			p.nextIf(tokenEqual)
			nl = p.parseTextBlock(m)
			pMap[argName] = nl
		case tokenRightSquare:
			m.ArgMap = pMap
			p.backup()
			return
		default:
			p.errorf("unexpected %q while parsing a command context on line %d", t.value, t.lnum)
		}
	}
	return
}

func (p *parser) parsePostionalArgs(m *Cmd) {
	var nl NodeList
	for {
		t := p.peek()
		switch t.typeof {
		case tokenLeftCurly:
			nl = p.parseTextBlock(m)
			m.ArgList = append(m.ArgList, nl)
			p.linkNodeList(nl)
		case tokenRightSquare:
			return
		default:
			p.errorf("unexpected %q while parsing command arguments on line %d", t.value, t.lnum)
		}
	}
	return
}

func (p *parser) parseTextBlock(m *Cmd) (nl NodeList) {
	cobra.Tag("parse").LogV("parsing text block")
	nl = NodeList{}
	t := p.nextIf(tokenLeftCurly)
	for {
		t = p.next()
		switch t.typeof {
		case tokenEmptyLine:
			if p.parScanOn && !p.insideSysCmd {
				continue
			}
			n := p.makeTextNode(t)
			nl = append(nl, n)
			p.link(n)
		case tokenIndent:
			if p.reflow && !p.insideSysCmd {
				continue
			} else {
				n := p.makeTextNode(t)
				p.root.append(n)
				p.link(n)
			}
		case tokenLineBreak:
			if p.parScanOn && !p.insideSysCmd {
				p.parseParagraph(t)
			} else {
				n := p.makeTextNode(t)
				nl = append(nl, n)
				p.link(n)
			}
		case tokenText:
			cobra.Tag("parse").LogV("adding textNode to NodeList")
			if p.parScanOn && !p.insidePar {
				p.insidePar = true
				p.root.append(NewParBeginNode(t))
			}
			n := NewTextNode(t.value)
			nl = append(nl, n)
			p.link(n)
		// case tokenSysCmd:
		// 	n := p.makeCmd(t)
		// 	n.SysCmd = true
		// 	p.insideSysCmd = true
		// 	cobra.Tag("parse").LogV("adding sysCmdStart to NodeList")
		// 	nl = append(nl, n)
		// 	p.link(n)
		case tokenCmdStart:
			n := p.makeCmd(t)
			cobra.Tag("parse").LogV("adding cmdStart to NodeList")
			nl = append(nl, n)
			p.link(n)
		case tokenRightCurly:
			cobra.Tag("parse").LogV("finished text block")
			return
		default:
			p.errorf("unexpected %q while parsing text block on line %d", t.value, t.lnum)
		}
	}
	cobra.LogV("We should never get here")
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
