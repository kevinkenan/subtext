package core

import (
	"fmt"
	"strings"

	"github.com/kevinkenan/cobra"
)

// Parse creates a node tree from the tokens produced by scan.
func Parse(d *Document) (*Section, error) {
	cobra.Tag("parse").WithField("name", d.Name).Add("plain", d.Plain).LogV("parsing input (parse)")

	p := &parser{
		doc:     d,
		scanner: scan(d),
		root:    NewSection(),
		empty:   true,
		// reflow:  d.Reflow,
	}

	if d.Plain {
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

	// for _, m := range d.macrosIn {
	// 	p.macros[MacroType{m.Name, m.Format}] = m
	// }

	return doParse(d.Name, p)
}

// // Parse creates a node tree from the tokens produced by scan.
// func ParseText(text string, plain bool, d *Document) (*Section, error) {
// 	cobra.Tag("parse").WithField("name", d.Name).Add("plain", d.Plain).LogV("parsing input (parse)")

// 	s := NewScanner(d.Name, text, plain, d)

// 	p := &parser{
// 		doc:     d,
// 		macro:   true,
// 		scanner: scanWith(s),
// 		root:    NewSection(),
// 		empty:   true,
// 		// reflow:  d.Reflow,
// 	}

// 	if plain {
// 		p.parMode = false
// 		p.parScanOn = false
// 		p.parScanFlag = false
// 		p.insidePar = false
// 	} else {
// 		p.parMode = true
// 		p.parScanOn = true
// 		p.parScanFlag = true
// 		p.insidePar = false
// 	}

// 	// for _, m := range d.macrosIn {
// 	// 	p.macros[MacroType{m.Name, m.Format}] = m
// 	// }

// 	return doParse(d.Name, p)
// }

func ParseMacro(name, input string, doc *Document, depth int) (*Section, error) {
	// o.Name, o.Default, parseOptions
	//opts := &Options{Plain: true, Macros: r.macros, Format: n.Format}
	p := &parser{
		doc:     doc,
		macro:   true,
		scanner: scanMacro(name, input, doc, depth),
		root:    NewSection(),
		empty:   true,
		// reflow:  doc.Reflow,
	}

	// Plain settings
	p.parMode = false
	p.parScanOn = false
	p.parScanFlag = false
	p.insidePar = false

	// for _, m := range doc.macrosIn {
	// 	p.macros[MacroType{m.Name, m.Format}] = m
	// }

	return doParse(name, p)
}

// TODO: remove this if unneeded
// func ParsePlain(d *Document) (*Section, error) {
// 	return Parse(d)
// }

func doParse(n string, p *parser) (*Section, error) {
	cobra.WithField("name", n).LogV("parsing (parse)")
	p.prevNode = p.root // Node(p.root)?
	return p.start()

}

// ----------------------------------------------------------------------------
// Parser ---------------------------------------------------------------------
// ----------------------------------------------------------------------------

// type Options struct {
// 	Macros MacroMap
// 	Reflow bool
// 	Plain  bool
// 	Format string
// }

type pstate struct {
	sysCmd    bool
	simpleCmd bool
}

// parser represents the current state of the parser.
type parser struct {
	doc      *Document //
	scanner  *scanner  //
	macro    bool      // True if we are parsing a macro
	root     *Section  // Root node of the tree.
	input    string    //
	empty    bool      // true if the buffer is empty.
	buffer   *token    // holds the next token if we peek or backup.
	pass     int       // which pass, we only parse some items on the second pass
	prevNode Node      // the previous node
	// reflow             bool
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

// GetMacro is a convenience function to get a macro.
func (p *parser) GetMacro(name, format string) *Macro {
	return p.doc.Folio.GetMacro(name, format)
}

// GetMacro is a convenience function to get a macro.
func (p *parser) GetSysMacro(name, format string) *Macro {
	return p.doc.Folio.GetSysMacro(name, format)
}

// AddMacro is a convenience function to add a macro.
func (p *parser) AddMacro(m *Macro) {
	p.doc.Folio.AddMacro(m)
}

func (p *parser) nextToken() (t *token) {
	return p.next()
}

func (p *parser) next() (t *token) {
	if p.empty {
		tt := p.scanner.nextToken()
		t = &tt
		p.buffer = t
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
	if p.peek().typeof == tokenComment {
		p.parseComment()
	}
	if t = p.next(); t.typeof == ttype {
		return
	}
	p.errorf("found %q instead of %q", tokenTypeLookup(t.typeof), tokenTypeLookup(ttype))
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

// TODO: Remove if not used
func (p *parser) pushState(s *pstate) {
	p.stateStack = append(p.stateStack, s)
}

// TODO: Remove if not used
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

func (p *parser) start() (n *Section, err error) {
	defer p.recover(&err)
	cobra.Tag("parse").LogV("parse start")

	// if p.doc.Template != "" && !p.macro {
	// 	n := NewCmdNode(p.doc.Template+".begin", &token{
	// 		typeof: tokenCmdStart,
	// 		loc:    Loc(0),
	// 		lnum:   0,
	// 		value:  "",
	// 	})
	// 	p.root.append(NodeList{n})
	// }

	for {
		nl, done, err := p.parseBody()
		if err != nil {
			return nil, err
		}
		p.root.append(nl)
		if done {
			break
		}
	}

	// if p.doc.Template != "" && !p.macro {
	// 	n := NewCmdNode(p.doc.Template+".end", &token{
	// 		typeof: tokenCmdStart,
	// 		loc:    p.scanner.scanFile.pos,
	// 		lnum:   p.scanner.scanFile.line,
	// 		value:  "",
	// 	})
	// 	p.root.append(NodeList{n})
	// }

	return p.root, nil
}

func (p *parser) parseBody() (nl NodeList, fileDone bool, err error) {
	cobra.Tag("parse").Add("cmdDepth", p.cmdDepth).LogV("parseText")
	var cmdDone bool
	nl = NodeList{}

	for {
		switch t := p.next(); t.typeof {
		case tokenComment:
			p.parseComment()
		case tokenSpaceEater:
			p.parseSpaceEater(t)
		case tokenEmptyLine:
			p.parseEmptyLine(t, &nl)
		case tokenIndent:
			p.parseIndent(t, &nl)
		case tokenLineBreak:
			p.parseLineBreak(t, &nl)
		case tokenText:
			p.parseText(t, &nl)
		case tokenSysCmdStart:
			p.parseSysCmd(t, &nl)
		case tokenCmdStart:
			p.parseCmd(t, &nl)
		case tokenRightCurly:
			cmdDone = p.parseRightCurly(t)
		case tokenRightSquare:
			cmdDone = p.parseRightSquare(t)
		case tokenError:
			p.errorf("Line %d: %s", t.lnum, t.value)
		case tokenEOF:
			fileDone = p.parseEOF(t, &nl)
		default:
			p.errorf("Line %d: unexpected token %q in parseText", t.lnum, tokenTypeLookup(t.typeof))
		}

		if cmdDone || fileDone {
			return
		}
	}

	cobra.Tag("parse").LogV("This should be impossible")
	return
}

func (p *parser) parseComment() {
	for {
		switch p.next().typeof {
		case tokenLineBreak, tokenEOF:
			return
		default:
			continue
		}
	}
}

func (p *parser) parseSpaceEater(t *token) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
	p.eatSpaces()
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
}

func (p *parser) eatSpaces() {
	for {
		switch p.next().typeof {
		case tokenEmptyLine, tokenIndent, tokenLineBreak, tokenSpaceEater:
			cobra.Tag("parse").LogV("eating space")
			continue
		case tokenComment:
			p.parseComment()
			continue
		default:
			p.backup()
			return
		}
	}
}

func (p *parser) parseEmptyLine(t *token, nl *NodeList) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
	if !p.parScanOn {
		n, _ := p.makeTextNode(t)
		*nl = appendNode(*nl, n)
		p.link(n)
	}
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
}

func (p *parser) parseIndent(t *token, nl *NodeList) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")

	if !p.doc.Reflow {
		n, _ := p.makeTextNode(t)
		*nl = appendNode(*nl, n)
		p.link(n)
	}

	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
}

func (p *parser) parseLineBreak(t *token, nl *NodeList) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")

	if p.parScanOn {
		pn := p.parseParagraph(t)
		*nl = appendNode(*nl, pn...)
	} else {
		n, _ := p.makeTextNode(t)
		*nl = appendNode(*nl, n)
		p.link(n)
	}

	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
}

func (p *parser) parseParagraph(t *token) (nl NodeList) {
	nl = NodeList{}
	// pkt := p.peek().typeof
	lb := false

loop: // eat empty space
	for {
		switch p.peek().typeof {
		case tokenComment:
			p.parseComment()
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
			pn := NewParEndNode(t)
			pn.Format = p.doc.Format
			nl = append(nl, pn)
			p.insidePar = false
		} else {
			if p.doc.Reflow {
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

	return
}

func (p *parser) parseText(t *token, nl *NodeList) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
	n, l := p.makeTextNode(t)

	if p.parScanOn && !p.insidePar && l > 0 {
		p.insidePar = true
		pn := NewParBeginNode(nil)
		pn.Format = p.doc.Format
		*nl = appendNode(*nl, pn)
	}

	*nl = appendNode(*nl, n)
	p.link(n)
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
}

func (p *parser) makeTextNode(t *token) (*Text, int) {
	l := len(t.value)
	s := t.value
	cobra.Tag("parse").LogV("creating a text node")

	if p.doc.Reflow {
		s = strings.TrimRight(s, " \t")

		switch p.peek().typeof {
		case tokenComment:
			p.parseComment()
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

func (p *parser) parseSysCmd(t *token, nl *NodeList) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
	var err error
	p.insideSysCmd = true

	_, cmd, err := p.makeCmd(t, nl)
	if err != nil {
		return
	}

	switch cmd.GetCmdName() {
	case "sys.newmacrof":
		err = p.doc.Folio.Macros.addNewMacro(cmd, p.doc, true)
	case "sys.newmacro":
		err = p.doc.Folio.Macros.addNewMacro(cmd, p.doc, false)
	// case "sys.configf":
	// 	err = p.processSysConfigCmd(cmd, true)
	// case "sys.config":
	// 	err = p.processSysConfigCmd(cmd, false)
	default:
		*nl = append(*nl, cmd)
		p.link(cmd)
	}

	if err != nil {
		p.errorf(err.Error())
	}

	p.insideSysCmd = false
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
	return
}

func (p *parser) parseCmd(t *token, nl *NodeList) (cmd *Cmd) {
	par, cmd, err := p.makeCmd(t, nl)
	if err != nil {
		p.errorf(err.Error())
	}

	if par != nil {
		*nl = appendNode(*nl, par)
	}

	*nl = appendNode(*nl, cmd)

	if cmd.Series || (cmd.Block && p.parScanOn) {
		p.blockSpaceEater()
	}

	p.link(cmd)
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
	return
}

func (p *parser) blockSpaceEater() {
	for {
		nxt := p.next()
		cobra.Tag("parse").Strunc("text", nxt.value).LogV("post block text")

		switch nxt.typeof {
		case tokenComment:
			p.parseComment()
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
			return
		}
	}
}

func (p *parser) makeCmd(t *token, nl *NodeList) (par, cmd *Cmd, err error) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")
	cmd = NewCmdNode(p.nextIf(tokenName).value, t)

	switch p.peek().typeof {
	case tokenComment:
		p.parseComment()
	case tokenLeftSquare:
		par, err = p.parseCmdContext(cmd)
	case tokenLeftCurly:
		par = p.initCmd(cmd)
		p.parseSimpleCmd(cmd)
	default:
		par = p.initCmd(cmd)
	}

	return
}

func (p *parser) initCmd(c *Cmd) (par *Cmd) {
	name := c.GetCmdName()
	cobra.WithField("name", name).LogV("parsing command (cmd)")

	format := p.doc.Format
	if c.HasFlag("noformat") {
		format = ""
	} else if f, ok := c.HasFlagVar("format"); ok {
		format = f
	}
	cobra.Tag("parse").Add("format", format).LogV("set cmd format")

	mac := p.GetMacro(name, format)
	if mac == nil {
		p.errorf("Line %d: command %q (format %q) not defined.", c.GetLineNum(), name, format)
		return
	}

	c.Format = format
	c.Block = mac.Block
	c.Series = mac.Series

	switch name {
	case "paragraph.begin":
		if p.parScanOn && !p.insidePar {
			p.insidePar = true
			// par = NewParBeginNode(c.cmdToken)
			// par.Format = p.doc.Format
			// *nl = appendNode(*nl, par)
		}
	case "paragraph.end":
		if p.parScanOn && p.insidePar {
			p.insidePar = false
			// par = NewParEndNode(c.cmdToken)
			// par.Format = p.doc.Format
			// *nl = appendNode(*nl, par)
		}
	default:
		if !p.insideSysCmd && p.parScanOn && !p.insidePar && !c.Block {
			p.insidePar = true
			par = NewParBeginNode(c.cmdToken)
			par.Format = p.doc.Format
		} else if !p.insideSysCmd && p.parScanOn && p.insidePar && c.Block {
			p.insidePar = false
			par = NewParEndNode(c.cmdToken)
			par.Format = p.doc.Format
		}
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
		nl, _, err = p.parseBody()
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
		case tokenComment:
			p.parseComment()
		case tokenRightCurly:
			p.backup()
			return NodeList{NewTextNode(w.String())}
		default:
			w.WriteString(t.value)
		}
	}
}

func (p *parser) parseCmdContext(m *Cmd) (par *Cmd, err error) {
	cobra.Tag("parse").LogV("parsing cmd context")
	t := p.nextIf(tokenLeftSquare)
	t = p.peek()

	if t.typeof == tokenLeftAngle {
		p.parseCmdFlags(m)
	}

	par = p.initCmd(m)

	parScanState := p.parScanOn
	if m.Block {
		p.parScanOn = false
	}

	p.cmdDepth += 1

reparse:
	p.eatSpaces()
	t = p.peek()
	switch t.typeof {
	case tokenLineBreak:
		p.next()
		goto reparse
	case tokenComment:
		p.parseComment()
		goto reparse
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

	return
}

func (p *parser) parseNamedArgs(m *Cmd) {
	pMap := make(NodeMap)
	var nl NodeList
	var err error

	for {
		p.eatSpaces()
		t := p.nextIf(tokenName)
		argName := t.value
		cobra.Tag("parse").WithField("arg", argName).LogV("parsing named args")
		p.nextIf(tokenEqual)
		t = p.nextIf(tokenLeftCurly)

		if m.SysCmd {
			nl = p.assembleText()
		} else {
			nl, _, err = p.parseBody()
			if err != nil {
				panic(fmt.Errorf(err.Error()))
			}
		}

		pMap[argName] = nl
		p.nextIf(tokenRightCurly)
		p.eatSpaces()

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
	p.eatSpaces()
	for {
		p.nextIf(tokenLeftCurly)

		if m.SysCmd {
			nl = p.assembleText()
		} else {
			nl, _, err = p.parseBody()
			if err != nil {
				panic(fmt.Errorf(err.Error()))
			}
		}

		m.ArgList = append(m.ArgList, nl)
		p.linkNodeList(nl)
		p.nextIf(tokenRightCurly)
		p.eatSpaces()

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
		case tokenComment:
			p.parseComment()
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

func (p *parser) parseRightCurly(t *token) (cmdDone bool) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")

	if p.cmdDepth > 0 {
		cobra.Tag("parse").LogV("finished cmd text block")
		cmdDone = true
		p.backup()
	}

	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).Add("cmdDepth", p.cmdDepth).LogV("end")
	return
}

func (p *parser) parseRightSquare(t *token) (cmdDone bool) {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")

	if p.cmdDepth > 0 {
		cobra.Tag("parse").LogV("finished full command text block")
		cmdDone = true
		p.backup()
	}

	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
	return
}

func (p *parser) parseEOF(t *token, nl *NodeList) bool {
	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("begin")

	if p.parScanOn && p.insidePar {
		pn := NewParEndNode(t)
		pn.Format = p.doc.Format
		*nl = append(*nl, pn)
	}

	cobra.Tag("parse").Add("token", tokenTypeLookup(t.typeof)).LogV("end")
	return true
}

// func (p *parser) processSysConfigCmd(n *Cmd, flowStyle bool) error {
// 	name := "sys.config"
// 	// Retrieve the sys.newmacro system command
// 	d := p.GetMacro(name, "")
// 	if d == nil {
// 		return fmt.Errorf("Line %d: system command %q not defined.", n.GetLineNum(), name)
// 	}
// 	cobra.Tag("cmd").Strunc("macro", d.TemplateText).LogfV("retrieved system command definition")

// 	args, err := d.ValidateArgs(n, p.doc)
// 	if err != nil {
// 		return fmt.Errorf("Line %d: ValidateArgs failed on system command %q: %q", n.GetLineNum(), name, err)
// 	}

// 	cobra.Tag("cmd").Strunc("syscmd", args["configs"].String()).LogfV("system command: %s", args["configs"])

// 	cfg := make(map[interface{}]interface{})
// 	if flowStyle {
// 		err = yaml.Unmarshal([]byte("{"+args["configs"].String()+"}"), &cfg)
// 	} else {
// 		err = yaml.Unmarshal([]byte(args["configs"].String()), &cfg)
// 	}
// 	if err != nil {
// 		return fmt.Errorf("Line %d: unmarshall error for system command %q: %q", n.GetLineNum(), name, err)
// 	}
// 	cobra.Tag("cmd").LogfV("marshalled syscmd: %+v", cfg)

// 	for k, v := range cfg {
// 		cobra.Tag("cmd").Add("key", k).Add("val", v).LogV("setting config from sys command")
// 		cobra.Set(k.(string), v)
// 	}

// 	p.reflow = p.doc.Reflow

// 	return nil
// }

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
