package parse

import (
	"fmt"
	// "strings"
	// "unicode"
	// "unicode/utf8"
	"github.com/kevinkenan/cobra"
)

// Parse creates a node tree from the tokens produced by scan.
func Parse(name, input string, mac map[string]*.Macro) (*Section, error) {
	cobra.Tag("parse").WithField("name", name).LogV("parsing input (parse)")
	p := &parser{
		scanner: scan(name, input),
		root:    NewSection(),
		empty:   true,
		macros:  mac,
	}
	return doParse(name, p)
}

// ParsePlain is the same as Parse but uses the scanPlain scanner.
func ParsePlain(name, input string, mac map[string]*.Macro) (*Section, error) {
	cobra.Tag("parse").WithField("name", name).LogV("parsing in plain mode (parse)")
	p := &parser{
		scanner: scanPlain(name, input),
		root:    NewSection(),
		empty:   true,
		macros:  mac,
	}
	return doParse(name, p)
}

func doParse(n string, p *parser) (*Section, error) {
	cobra.WithField("name", n).LogV("parsing (parse)")
	p.prevNode = p.root // Node(p.root)?
	return p.start()

}

// ----------------------------------------------------------------------------
// Parser ---------------------------------------------------------------------
// ----------------------------------------------------------------------------

// parser represents the current state of the parser.
type parser struct {
	scanner *scanner //
	root    *Section // Root node of the tree.
	// depth    int
	input  string
	empty  bool   // true if the buffer is empty.
	buffer *token // holds the next token if we peek or backup.
	// cmdDepth int
	prevNode Node // the previous node
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

// Parse token stream ---------------------------------------------------------

func (p *parser) start() (n *Section, err error) {
	defer p.recover(&err)
	cobra.Tag("parse").LogV("parse root level nodes")
	// return p.debugLoop()
	// fmt.Println("here")
Loop:
	for {
		t := p.next()
		switch t.typeof {
		case tokenText:
			n := p.makeTextNode(t)
			p.root.append(n)
			p.link(n)
		case tokenCmdStart:
			n := p.makeCmd(t)
			p.root.append(n)
			p.link(n)
		case tokenSysCmdStart:
			// ns := p.makeSysCmd(t)
			n := p.makeCmd(t)
			n.SysCmd = true
			n.NodeValue = NodeValue(fmt.Sprintf("sys.%s", n.NodeValue))
			p.root.append(n)
			p.link(n)
			// for _, n := range ns {
			// 	p.root.append(n)
			// 	p.link(n)
			// }
		// case tokenParagraphStart:
		// 	p.root.append(NewParagraphStart(t.value))
		// case tokenParagraphEnd:
		// 	p.root.append(NewParagraphEnd(t.value))
		case tokenError:
			p.errorf("Line %d: %s", t.lnum, t.value)
		case tokenEOF:
			break Loop
		default:
			p.errorf("Line %d: unexpected token %q when starting with value %q", t.lnum, tokenTypeLookup(t.typeof), t.value)
		}
	}
	return p.root, nil
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
	cobra.WithField("name", n.GetCmdName()).LogV("parsing command (cmd)")
	switch p.peek().typeof {
	case tokenLeftSquare:
		p.parseCmdContext(n)
	case tokenLeftCurly:
		p.parseSimpleCmd(n)
	default:
		// p.cmdDepth -= 1
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
		case tokenText:
			n := NewTextNode(t.value)
			cobra.Tag("parse").LogV("adding textNode to NodeList")
			nl = append(nl, n)
			p.link(n)
		case tokenSysCmd:
			n := p.makeCmd(t)
			n.SysCmd = true
			cobra.Tag("parse").LogV("adding sysCmdStart to NodeList")
			nl = append(nl, n)
			p.link(n)
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
