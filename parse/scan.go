package parse

import (
	"fmt"
	"github.com/kevinkenan/cobra"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ----------------------------------------------------------------------------
// Token ----------------------------------------------------------------------
// ----------------------------------------------------------------------------

// TokenType indicates a type of token.
type tokenType int

const (
	tokenText             tokenType = iota // Plain text.
	tokenCmdStart                          // Indicates the start of a command definition.
	tokenName                              // The token's name
	tokenRunes                             // A string of runes.
	tokenLeftParenthesis                   // (
	tokenRightParenthesis                  // )
	tokenLeftSquare                        // [
	tokenRightSquare                       // ]
	tokenLeftCurly                         // {
	tokenRightCurly                        // }
	tokenLeftAngle                         // <
	tokenRightAngle                        // >
	tokenEqual                             // =
	tokenComma                             // ,
	tokenTilde                             // ~
	tokenError                             // value holds the message produced by a scanning error.
	tokenEOF                               // The end of the input text.
	// System Command Tokens
	tokenSysCmdStart         // Indicates the start of system commands
	tokenSysCmd              // A system command
	tokenCmdParagraphModeOff // Treat paragraphs as regular text.
	tokenCmdParagraphModeOn  // Treat paragraphs as paragraphs.
)

var tokenNames = []string{
	"tokenText",
	"tokenCmdStart",
	"tokenName",
	"tokenRunes",
	"tokenLeftParenthesis",
	"tokenRightParenthesis",
	"tokenLeftSquare",
	"tokenRightSquare",
	"tokenLeftCurly",
	"tokenRightCurly",
	"tokenLeftAngle",
	"tokenRightAngle",
	"tokenEqual",
	"tokenComma",
	"tokenTilde",
	"tokenError",
	"tokenEOF",
	"tokenSysCmdStart",
	"tokenSysCmd",
	"tokenCmdParagraphModeOff",
	"tokenCmdParagraphModeOn",
}

func tokenTypeLookup(typeNum tokenType) string {
	return tokenNames[typeNum]
}

// A token represents a unit of syntax which will be used by the parser.
type token struct {
	typeof tokenType // This token's type.
	loc    Loc       // The starting location of this token's text.
	lnum   int       // The line number of Loc.
	value  string    // This token's text.
}

func (t token) String() string {
	switch {
	case t.typeof == tokenEOF:
		return fmt.Sprintf("       {eof %d/%d}", t.loc, t.lnum)
	case t.typeof == tokenError:
		return fmt.Sprintf("       {error %d/%d}: %q", t.loc, t.lnum, t.value)
	default:
		return fmt.Sprintf("       %s: %q\n", tokenTypeLookup(t.typeof), t.value)
	}
	return ""
}

// ----------------------------------------------------------------------------
// Scanner --------------------------------------------------------------------
// ----------------------------------------------------------------------------

const (
	eof        = -1
	spaceChars = " \t\r\n"
)

// Loc is a byte location in the input string.
type Loc int

// scanner represents the current state.
type scanner struct {
	name          string     // name of the doc being scanned
	input         string     // the string being scanned
	cmdH          string     // rune indicating a horizontal-mode command
	cmdV          string     // rune indicating a vertical-mode command
	commentToggle string     // rune that toggles commenting
	parCmd        string     // rune indicating a paragraph command
	pos           Loc        // current position in the input
	start         Loc        // start position of this item
	width         Loc        // width of last rune read from input
	tokens        chan token // channel of scanned tokens
	cmdDepth      int        // nesting depth of commands
	altTerm       bool       // true if '*}' terminates a text block
	line          int        // number of newlines seen (starts at 1)
	// cmdStack indicates if a command's text block was called from within a
	// full command (with a context) or from a short command.
	cmdStack           []cmdType
	parMode            bool // true when the scanner is invoked with scan instead of scanPlain
	diableParScanFlags bool // when true, the scanner ignores ¶ commands
	allowParScan       bool // When false, the scanner is not allowed to insert paragraphs
	parScannerOn       bool // when true, the scanner generates paragraph commands
	parScanFlag        bool // set by ¶ command
	parOpen            bool // tracks if every par begin is matched by a par end
}

func NewScanner(name, input string) *scanner {
	return &scanner{
		name:          name,
		input:         input,
		cmdH:          "•",
		cmdV:          "§",
		commentToggle: "◊",
		parCmd:        "¶",
		tokens:        make(chan token),
		line:          1,
		allowParScan:  true,
		parMode:       true,
		parScannerOn:  true,
		parScanFlag:   true,
		parOpen:       false,
	}
}

type cmdType int

const (
	short    cmdType = iota // short command
	shortAlt                // short command with alt terminator
	full                    // full command
	fullAlt                 // full command with alt terminator
	syscmd
)

// scan generates paragraph commands while tokenizing the input string.
func scan(name, input string) *scanner {
	cobra.Tag("scan").WithField("name", name).LogV("scanning input in paragraph mode (scan)")
	s := NewScanner(name, input)
	return scanWith(s)
}

// scanPlain does not generate paragraph commands while tokenizing the input
// string.
func scanPlain(name, input string) *scanner {
	cobra.Tag("scan").WithField("name", name).LogV("scanning input in plain mode (scan)")
	s := NewScanner(name, input)
	s.allowParScan = true
	s.parMode = false
	s.parScannerOn = false
	s.parScanFlag = false
	s.parOpen = false
	return scanWith(s)
}

// scanWith allows the use of an externally created and configured scanner.
func scanWith(s *scanner) *scanner {
	go s.run()
	return s
}

// run runs the state machine for the scanner.
func (s *scanner) run() {
	cobra.Tag("scan").Tag("scan").LogV("start scanning")
	for state := scanBeginning; state != nil; {
		state = state(s)
	}
	close(s.tokens)
	cobra.Tag("scan").LogV("done scanning")
}

func (s *scanner) isParScanAllowed() bool {
	return s.isParMode() && s.isParScanFlag()
}

func (s *scanner) isParMode() bool {
	return s.parMode && s.allowParScan
}

func (s *scanner) isParScanFlagDisabled() bool {
	return s.diableParScanFlags
}

func (s *scanner) isParScanFlag() bool {
	return s.allowParScan && s.parScanFlag
}

func (s *scanner) setParScanFlag(b bool) bool {
	cobra.Tag("scan").WithField("state", b).LogV("setting parscan flag")
	if !s.diableParScanFlags {
		s.parScanFlag = b
	} else {
		cobra.Tag("scan").LogV("paragraph flags are disabled")
	}
	return s.parScanFlag
}

func (s *scanner) isParScanOn() bool {
	// return s.parScannerOn && s.allowParScan && s.parMode && s.parScanFlag
	return s.parScannerOn
}

func (s *scanner) isParScanOff() bool {
	// return s.parScannerOn && s.allowParScan && s.parMode && s.parScanFlag
	return !s.parScannerOn
}

func (s *scanner) setParScanOff() bool {
	return s.setParScan(false)
}

func (s *scanner) setParScanOn() bool {
	return s.setParScan(true)
}

func (s *scanner) setParScan(b bool) bool {
	if true { //s.allowParScan {
		cobra.Tag("scan").WithField("parscan", b).LogV("setting parscan")
		s.parScannerOn = b
	} else {
		cobra.WithField("line", s.line).LogV("paragraph scanning is not allowed")
	}
	return s.parScannerOn
}

func (s *scanner) isInsidePar() bool {
	return s.allowParScan && s.parOpen
}

func (s *scanner) setInsidePar(b bool) bool {
	if true { //s.allowParScan {
		s.parOpen = b
	}
	return s.parOpen
}

// pushCmdTextExit is called when you enter a text block
func (s *scanner) pushCmd(m cmdType) {
	s.cmdStack = append(s.cmdStack, m)
}

func (s *scanner) popCmd() (m cmdType) {
	l := len(s.cmdStack)
	if l == 0 {
		s.errorf("attempted to read past the end of the command stack")
	}
	m = s.cmdStack[l-1]
	s.cmdStack = s.cmdStack[:l-1]
	return
}

// isCmdCmd returns true if the rune is the cmd character.
func (s *scanner) isCmdCmd(r rune) bool {
	cmdH, _ := utf8.DecodeRuneInString(s.cmdH)
	cmdV, _ := utf8.DecodeRuneInString(s.cmdV)
	return r == cmdH || r == cmdV
}

// getCmdMode returns an "H" if the command should be interpreted in
// horizontal or a "V" if it should be interpreted in vertical mode.
func (s *scanner) getCmdMode(r rune) string {
	cmdH, _ := utf8.DecodeRuneInString(s.cmdH)
	cmdV, _ := utf8.DecodeRuneInString(s.cmdV)
	switch r {
	case cmdH:
		return "H"
	case cmdV:
		return "V"
	}
	return ""
}

// isHorizCmd returns true if it is a horizontal mode command.
func (s *scanner) isHorizCmd(r rune) bool {
	hcmd, _ := utf8.DecodeRuneInString(s.cmdH)
	return r == hcmd
}

// isCommentToggle returns true if the rune is the comment toggle character.
func (s *scanner) isCommentToggle(r rune) bool {
	q, _ := utf8.DecodeRuneInString(s.commentToggle)
	return r == q
}

// next returns the next rune in the input.
func (s *scanner) next() rune {
	if int(s.pos) >= len(s.input) {
		s.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.width = Loc(w)
	s.pos += s.width
	if r == '\n' {
		s.line++
	}
	return r
}

// peek returns but does not consume the next rune in the input.
func (s *scanner) peek() (r rune) {
	r = s.next()
	s.backup()
	return
}

// isNextRune tests to see if the next rune is the indicated rune. The call
// does not break backup.
func (s *scanner) isNextRune(r rune) bool {
	if int(s.pos) >= len(s.input) {
		return false
	}
	n, _ := utf8.DecodeRuneInString(s.input[s.pos:])
	return r == n
}

// backup steps back one rune. Can only be called once per call of next.
func (s *scanner) backup() {
	s.pos -= s.width
	// Correct newline count.
	if s.width == 1 && s.input[s.pos] == '\n' {
		s.line--
	}
}

// ignore skips over the pending input before this point.
func (s *scanner) ignore() {
	s.line += strings.Count(s.input[s.start:s.pos], "\n")
	s.start = s.pos
}

// jump ignores the next k bytes.
func (s *scanner) jump(k int) {
	s.pos += Loc(k)
	s.ignore()
}

// jumpNextRune ignores the next rune.
func (s *scanner) jumpNextRune() {
	_, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.pos += Loc(w)
	s.ignore()
}

// accept consumes the next rune if it's from the valid set.
func (s *scanner) accept(valid string) bool {
	if strings.ContainsRune(valid, s.next()) {
		return true
	}
	s.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (s *scanner) acceptRun(valid string) {
	for strings.ContainsRune(valid, s.next()) {
	}
	s.backup()
}

func (s *scanner) eatSpaces() {
	s.acceptRun(spaceChars)
	s.forget()
}

// forget skips over input that's been accepted using accept() or acceptRun().
// Unlike ignore(), forget() doesn't count line breaks since they have already
// been counted in the accept functions.
func (s *scanner) forget() {
	s.start = s.pos
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating s.nextToken.
func (s *scanner) errorf(format string, args ...interface{}) ƒ {
	cobra.Tag("scan").LogV("errorf: %q", fmt.Sprintf(format, args...))
	s.tokens <- token{
		typeof: tokenError,
		loc:    s.start,
		lnum:   s.line,
		value:  fmt.Sprintf(format, args...)}
	return nil
}

// nextToken returns the next token from the input.
// Called by the parser, not in the scanning goroutine.
func (s *scanner) nextToken() token {
	return <-s.tokens
}

// drain drains the output so the scanning goroutine will exit.
// Called by the parser, not in the scanning goroutine.
func (s *scanner) drain() {
	for range s.tokens {
	}
}

// emitInsertedToken creates a token with the given value (which wasn't found
// in the source text) and sends it to the client.
func (s *scanner) emitInsertedToken(t tokenType, val string) {
	cobra.Tag("tokens").WithField("type", tokenTypeLookup(t)).Strunc("value", val).LogfV("emitInsertedToken")
	s.tokens <- token{
		typeof: t,
		loc:    s.start,
		lnum:   s.line,
		value:  val}
}

// emitRawToken passes a token to the client.
func (s *scanner) emitRawToken(t tokenType) {
	cobra.Tag("tokens").Strunc("value", string(s.input[s.start:s.pos])).Add("type", tokenTypeLookup(t)).LogfV("emitRawToken")
	s.tokens <- token{
		typeof: t,
		loc:    s.start,
		lnum:   s.line,
		value:  s.input[s.start:s.pos]}
	s.start = s.pos
}

// emit only emits a text token if there is pending text. Other tokens are
// emitted as is.
func (s *scanner) emit(t tokenType) {
	switch {
	case t == tokenText:
		if s.start < s.pos {
			s.emitRawToken(t)
		}
	default:
		s.emitRawToken(t)
	}
}

// emitSysCmd passes a system command token to the client.
func (s *scanner) emitSysCmd(t tokenType, args string) {
	cobra.Tag("tokens").Strunc("value", string(s.input[s.start:s.pos])).LogfV("emitSysCmd")
	s.tokens <- token{
		typeof: t,
		loc:    s.start,
		lnum:   s.line,
		value:  args}
	s.start = s.pos
}

// ----------------------------------------------------------------------------
// State Machine --------------------------------------------------------------
// ----------------------------------------------------------------------------

// ƒ represents the state machine that returns the next state.
type ƒ func(*scanner) ƒ

func scanBeginning(s *scanner) ƒ {
	cobra.Tag("scan").LogV("beginning scan")
	if s.isParScanOn() {
		s.eatSpaces()
	}
Loop:
	for {
		switch r := s.peek(); {
		case isSpace(r) || isEndOfLine(r):
			if s.isParScanOn() {
				s.eatSpaces()
			} else {
				s.next()
			}
		case s.isCmdCmd(r):
			s.sendInitialParagraph()
			return scanNewCommand
		case r == '¶':
			s.emit(tokenText) // just white space
			s.next()
			switch nxt := s.next(); {
			case nxt == '+':
				cobra.Tag("scan").Add("line", s.line).LogV("¶+")
				s.ignore()
				if !s.isParScanFlagDisabled() {
					cobra.Tag("scan").LogV("turning paragraph scan on")
					s.setParScanFlag(true)
					s.setParScanOn()
					return scanBeginning
				}
			case nxt == '-':
				cobra.Tag("scan").Add("line", s.line).LogV("¶-")
				s.ignore()
				if !s.isParScanFlagDisabled() {
					cobra.Tag("scan").LogV("turning paragraph scan off")
					if s.isInsidePar() {
						cobra.Tag("scan").LogV("flushing open paragraph")
						s.insertParagraphEndCmd()
					}
					s.setParScanFlag(false)
					s.setParScanOff()
				}
			default:
				s.errorf("character %q not a valid character to follow ¶", nxt)
			}
		case s.isCommentToggle(r):
			s.emit(tokenText) // just white space
			scanCommentToggle(s)
		case isEndOfFile(r):
			s.emit(tokenText) // all the white space gathered so far
			break Loop
		default:
			if sent := s.sendInitialParagraph(); !sent {
				s.emit(tokenText)
			}
			cobra.Tag("scan").Add("line", s.line).Strunc("char", string(r)).LogV("done scanBeginning")
			return scanText
		}
	}
	s.emit(tokenEOF)
	cobra.Tag("scan").WithField("name", s.name).LogV("completed scanBeginning")
	return nil
}

// sendInitialParagraph sends a paragraph when needed at the beginning of the
// document and whenever paragraph scanning is turned back on.
func (s *scanner) sendInitialParagraph() (sent bool) {
	if s.isParScanOn() && !s.isInsidePar() {
		cobra.Tag("scan").LogV("insertParagraphBeginCmd (initial par)")
		s.setParScanOn()
		s.setInsidePar(true)
		s.emitInsertedToken(tokenCmdStart, "")
		s.emitInsertedToken(tokenName, "sys.paragraph.begin")
		s.emitInsertedToken(tokenLeftSquare, "[")
		s.emitInsertedToken(tokenLeftCurly, "{")
		s.emitRawToken(tokenText) // will contain all the opening white space
		s.emitInsertedToken(tokenRightCurly, "}")
		s.emitInsertedToken(tokenRightSquare, "]")
		sent = true
	} else {
		cobra.Tag("scan").LogV("no initial paragraph")
		s.emit(tokenText)
	}
	return
}

func scanText(s *scanner) ƒ {
	cobra.Tag("scan").LogV("scanText")
Loop:
	for {
		switch r := s.next(); {
		case isAlphaNumeric(r):
			if !s.isInsidePar() && s.isParScanOn() {
				s.backup()
				cobra.Tag("scan").WithField("length", len(s.input[s.start:s.pos])).Add("line", s.line).LogfV("alphanumeric buffer")
				s.insertParagraphBeginCmd()
				s.next()
			}
		case isEndOfLine(r):
			mark := s.pos - 1
			s.acceptRun(" \t") // accept horizontal white space
			pk := s.peek()
			if s.isParScanOn() && (isEndOfLine(pk) || isEndOfFile(pk)) {
				if s.isInsidePar() {
					s.pos = mark // reset the stream back to the beginning of the par break
					s.emit(tokenText)
					s.acceptRun(spaceChars)
					s.insertParagraphEndCmd() // par end cmd will contain all the white space
				} else {
					s.eatSpaces()
				}
			}
		case r == '¶':
			s.backup()
			s.emit(tokenText)
			s.next()
			switch nxt := s.next(); {
			case nxt == '+':
				cobra.Tag("scan").Add("line", s.line).LogV("encountered ¶+")
				s.ignore()
				if !s.isParScanFlagDisabled() {
					cobra.Tag("scan").LogV("turning paragraph scan on")
					s.setParScanFlag(true)
					s.setParScanOn()
					return scanBeginning
				}
			case nxt == '-':
				cobra.Tag("scan").Add("line", s.line).LogV("encountered ¶-")
				s.ignore()
				if !s.isParScanFlagDisabled() {
					cobra.Tag("scan").LogV("turning paragraph scan off")
					s.setParScanFlag(false)
					s.setParScanOff()
				}
			default:
				s.errorf("character %q not a valid character to follow ¶", nxt)
			}
		case r == '`':
			// escaping the next character
			s.backup()
			s.emit(tokenText)
			s.jumpNextRune()
			s.next()
		case r == '*' && s.altTerm:
			if s.cmdDepth > 0 && s.peek() == '}' {
				s.backup()
				s.emit(tokenText)
				s.next()
				s.ignore()
				s.next()
				s.emit(tokenRightCurly)
				return s.exitTextBlock()
			}
		case r == '}' && !s.altTerm:
			if s.cmdDepth > 0 {
				s.backup()
				s.emit(tokenText)
				s.next()
				s.emit(tokenRightCurly)
				return s.exitTextBlock()
			}
		case s.isCmdCmd(r):
			s.backup()
			s.emit(tokenText)
			return scanNewCommand
		case s.isCommentToggle(r):
			cobra.Tag("scan").Add("line", s.line).LogV("comment toggle")
			s.backup()
			s.emit(tokenText)
			scanCommentToggle(s)
		case isEndOfFile(r):
			cobra.Tag("scan").Add("line", s.line).LogV("eof encountered")
			if s.cmdDepth > 0 {
				s.errorf("end of file while command is still open")
			}
			break Loop
		default:
			cobra.Tag("scan").Add("line", s.line).Strunc("char", string(r)).LogfV("still scanning text")
			continue
			// return s.errorf("unexpected character %q while scanning text", r)
		}
	}

	cobra.Tag("scan").LogV("finishing scanText")

	if s.pos > s.start {
		if s.isParScanOn() {
			s.insertParagraphBeginCmd()
		}
		cobra.Tag("scan").Add("line", s.line).LogV("flushing token buffer (tokens)")
		s.emit(tokenText)
	}

	if s.isInsidePar() {
		s.insertParagraphEndCmd()
	}

	s.emit(tokenEOF)
	cobra.Tag("scan").WithField("name", s.name).Add("line", s.line).LogV("completed scan")
	return nil
}

func (s *scanner) insertParagraphBeginCmd() {
	if s.isInsidePar() || s.isParScanOff() {
		cobra.Tag("scan").LogV("aborting insertParagraphBeginCmd")
		return
	}
	cobra.Tag("scan").LogV("insertParagraphBeginCmd")
	s.setInsidePar(true)
	s.emitInsertedToken(tokenCmdStart, "")
	s.emitInsertedToken(tokenName, "sys.paragraph.begin")
	s.emitInsertedToken(tokenLeftSquare, "[")
	s.emitInsertedToken(tokenLeftCurly, "{")
	s.emitRawToken(tokenText)
	s.emitInsertedToken(tokenRightCurly, "}")
	s.emitInsertedToken(tokenRightSquare, "]")
}

func (s *scanner) insertParagraphEndCmd() {
	if !s.isInsidePar() || s.isParScanOff() {
		cobra.Tag("scan").LogV("aborting insertParagraphEndCmd")
		return
	}
	cobra.Tag("scan").LogV("insertParagraphEndCmd")
	s.setInsidePar(false)
	s.emitInsertedToken(tokenCmdStart, "")
	s.emitInsertedToken(tokenName, "sys.paragraph.end")
	s.emitInsertedToken(tokenLeftSquare, "[")
	s.emitInsertedToken(tokenLeftCurly, "{")
	// s.next()
	s.emitRawToken(tokenText)
	s.emitInsertedToken(tokenRightCurly, "}")
	s.emitInsertedToken(tokenRightSquare, "]")
}

// Scans for ◊ characters which toggles comments.
func scanCommentToggle(s *scanner) {
	// input:  text◊...◊text
	// s.pos:      ^
	s.jumpNextRune()
	i := strings.Index(s.input[s.pos:], "◊")
	if i < 0 {
		s.jump(len(s.input[s.pos:]))
		return
	}
	s.jump(i + len("◊"))
	return
}

// scanCommand creates a cmd token.
// Types of command:
//   * bare:   •cmd followed by non-alphanumeric char
//   * short:  •cmd{text block}
//   * full:   •cmd[context]
// A system command has the same form, but the name is in parantheses: •(cmd)...
func scanNewCommand(s *scanner) ƒ {
	// input:  •cmd[...]
	// s.pos:  ^
	cobra.Tag("scan").Add("line", s.line).LogV("scanNewCommand")
	cr := s.next()
	s.ignore()
	cmdMode := s.getCmdMode(cr)

	// Determine the command
	switch r := s.peek(); {
	case isAlphaNumeric(r):
		cobra.Tag("scan").WithField("mode", cmdMode).LogV("macro command")

		switch {
		case s.isHorizCmd(cr):
			// If we're not in a paragraph, this will start one for us.
			s.insertParagraphBeginCmd()
		case !s.isHorizCmd(cr):
			s.emit(tokenText)
			s.insertParagraphEndCmd()
		}

		s.setParScanOff()
		s.emitInsertedToken(tokenCmdStart, cmdMode)
		s.altTerm = scanName(s)

		switch s.peek() {
		case '[':
			return scanFullCmd
		case '{':
			return scanShortCmd
		}

		if s.peek() == '%' {
			s.next()
			s.ignore()
			s.eatSpaces()
		}

		cobra.Tag("scan").LogV("done scanning bare command")
		if s.cmdDepth < 1 && s.isParScanAllowed() && s.isParScanOff() {
			s.setParScanOn()
		}

		return scanText
	case r == '(':
		cobra.Tag("scan").LogV("system command")
		s.next()
		s.ignore()
		s.setParScanOff()
		s.emit(tokenSysCmdStart)
		s.altTerm = scanName(s)

		r = s.next()
		if r != ')' {
			return s.errorf("illegal character, %q, found in system command", r)
		}
		s.ignore()

		switch s.peek() {
		case '[':
			return scanFullCmd
		case '{':
			return scanShortCmd
		}

		if s.peek() == '%' {
			s.next()
			s.ignore()
			s.eatSpaces()
		}

		cobra.Tag("scan").LogV("done scanning bare system command")
		return scanText
	case r == '%': // space eater
		cobra.Tag("scan").Add("line", s.line).LogV("eating spaces")
		s.jumpNextRune()
		s.eatSpaces()
		return scanText
	case r == '|':
		cobra.Tag("scan").Add("line", s.line).LogV("line comment")
		return scanEolComment
	case isSpace(r) || isEndOfLine(r) || isEndOfFile(r):
		return s.errorf("unnamed command")
	default:
		return s.errorf("character %q not a valid command character", r)
	}
	// If we somehow get here, just say no.
	return s.errorf("illegal character, %q, found in command", s.next())
}

func scanShortCmd(s *scanner) ƒ {
	// input: •cmd{...}
	// s.pos:     ^
	for {
		switch r := s.next(); {
		case r == '{':
			s.emit(tokenLeftCurly)
			return s.enterTextBlock(short)
		case r == '*' && s.altTerm:
			if s.peek() == '}' {
				s.ignore()
				s.next()
				s.emit(tokenRightCurly)
				return s.exitTextBlock()
			}
		case r == '}' && !s.altTerm:
			s.emit(tokenRightCurly)
			return s.exitTextBlock()
		case isEndOfFile(r):
			s.errorf("end of file while processing command")
		default:
			s.errorf("invalid character '%q' in command", r)
		}
	}
}

func scanFullCmd(s *scanner) ƒ {
	// input: •cmd[...]
	// s.pos:     ^
	for {
		switch r := s.next(); {
		case r == '<':
			cobra.Tag("scan").Add("line", s.line).LogV("scan cmd flags")
			s.emit(tokenLeftAngle)
			scanCmdFlags(s)
		case r == '>':
			s.emit(tokenRightAngle)
		case isAlphaNumeric(r):
			s.altTerm = scanName(s)
		case r == '=':
			cobra.Tag("scan").LogV("scan cmd arg name")
			s.emit(tokenEqual)
		case r == '[':
			cobra.Tag("scan").LogV("scan cmd context")
			s.cmdDepth += 1
			s.emit(tokenLeftSquare)
		case r == ']':
			s.emit(tokenRightSquare)
			s.cmdDepth -= 1
			cobra.Tag("scan").Add("line", s.line).LogV("done scanning extended command")

			if s.peek() == '%' {
				s.next()
				s.ignore()
				s.eatSpaces()
			}

			if s.cmdDepth < 1 && s.isParScanAllowed() && s.isParScanOff() {
				s.setParScanOn()
				return scanBeginning
			}

			if s.isParScanOn() {
				s.eatSpaces()
			}

			return scanText
		case r == '{':
			cobra.Tag("scan").Add("line", s.line).LogV("scan cmd argument")
			s.emit(tokenLeftCurly)
			return s.enterTextBlock(full)
		case r == '}':
			s.emit(tokenRightCurly)
			return s.exitTextBlock()
		case s.isCommentToggle(r):
			s.backup()
			scanCommentToggle(s)
		case isSpace(r) || isEndOfLine(r):
			s.eatSpaces()
		case isEndOfFile(r):
			s.errorf("end of file while processing command")
		default:
			s.errorf("invalid character %q in command body", r)
		}
	}
}

func scanCmdFlags(s *scanner) {
	for {
		switch r := s.next(); {
		case isAlphaNumeric(r) || isExtendedChar(r):
			scanRunes(s)
		case r == ',':
			s.emit(tokenComma)
		case r == '~':
			s.emit(tokenTilde)
		case isSpace(r):
			s.eatSpaces()
		case r == '>':
			s.backup()
			return
		default:

		}
	}
}

// scanName creates a name token.
func scanName(s *scanner) bool {
	for {
		switch r := s.next(); {
		case isAlphaNumeric(r) || r == '_' || r == '.' || r == '-' || r == '*':
			continue
		case s.isCommentToggle(r):
			scanCommentToggle(s)
		default:
			alt := false
			s.backup()
			name := s.input[s.start:s.pos]
			s.ignore()
			cobra.Tag("scan").Add("line", s.line).WithField("name", name).LogfV("scanName")

			if strings.HasSuffix(name, "*") {
				cobra.Tag("scan").LogV("cmd specifies alt terminator")
				alt = true
				name = strings.TrimSuffix(name, "*")
			}

			s.emitInsertedToken(tokenName, name)
			return alt
		}
	}
}

// scanSysCmd creates a  sysCmd token.
func scanSysCmd(s *scanner) ƒ {
	// cobra.Tag("scan").LogV("system command")
	s.next()
	s.emit(tokenLeftParenthesis)
	for run := true; run; {
		switch r := s.peek(); {
		case isAlphaNumeric(r) || r == '_' || r == '.' || r == '-' || r == '=':
			s.next()
		case r == ')':
			cobra.Tag("scan").Strunc("name", s.input[s.start:s.pos]).Add("line", s.line).LogV("system command")

			if s.pos > s.start {
				s.emit(tokenSysCmd)
			}

			s.emit(tokenRightParenthesis)

			s.next()
			pk := s.peek()
			if pk == '{' {
				// input: •(cmd){...}
				// s.pos:       ^
				for {
					switch r := s.next(); {
					case r == '{':
						s.emit(tokenLeftCurly)
						return s.enterTextBlock(syscmd)
					case r == '}':
						s.emit(tokenRightCurly)
						return s.exitTextBlock()
					case isEndOfFile(r):
						s.errorf("end of file while processing command")
					default:
						s.errorf("invalid character '%q' in command", r)
					}
				}
			}

			run = false
		case r == ',':
			if s.pos > s.start {
				s.emit(tokenSysCmd)
			}

			s.next()
			s.emit(tokenComma)
		case isSpace(r), isEndOfLine(r):
			if s.pos > s.start {
				s.emit(tokenSysCmd)
			}

			s.eatSpaces()
		case isEndOfFile(r):
			s.errorf("end of file in sysCmd call")
		default:
			s.errorf("invalid character %q in sysCmd", r)
		}
	}
	return scanText
}

// scanRunes creates a token of arbitrary runes.
func scanRunes(s *scanner) {
	for {
		switch r := s.next(); {
		case isAlphaNumeric(r) || isExtendedChar(r):
			continue
		default:
			s.backup()
			s.emit(tokenRunes)
			return
		}
	}
}

// Scan comments that terminate at the end of the line.
func scanEolComment(s *scanner) ƒ {
	eol := strings.Index(s.input[s.pos:], "\n")

	if eol == -1 {
		// end of file.
		s.jump(len(s.input[s.pos:]))
		return scanText
	}

	s.jump(len("|") + eol)

	if s.isParScanOn() {
		s.eatSpaces()
	}

	return scanText
}

// ----------------------------------------------------------------------------
// Utilities ------------------------------------------------------------------
// ----------------------------------------------------------------------------

func (s *scanner) enterTextBlock(m cmdType) ƒ {
	switch {
	case m == short && s.altTerm:
		s.cmdDepth += 1
		s.pushCmd(shortAlt)
	case m == short && !s.altTerm:
		s.cmdDepth += 1
		s.pushCmd(short)
	case m == full && s.altTerm:
		s.pushCmd(fullAlt)
	case m == full && !s.altTerm:
		s.pushCmd(full)
	default:
		s.errorf("unknown command type when entering text block")
	}
	return scanText
}

func (s *scanner) exitTextBlock() (f ƒ) {
	if s.peek() == '%' {
		s.next()
		s.ignore()
		s.eatSpaces()
	}
	m := s.popCmd()
	switch m {
	case shortAlt:
		s.altTerm = true
		fallthrough
	case short:
		s.cmdDepth -= 1
		if s.cmdDepth < 1 && s.isParScanAllowed() && s.isParScanOff() {
			s.setParScanOn()
		}
		cobra.Tag("scan").Tag("scan").LogV("done scanning short command")
		f = scanText
	case fullAlt:
		s.altTerm = true
		fallthrough
	case full:
		f = scanFullCmd
	case syscmd:
		f = scanText
	default:
		s.errorf("unknown command type when exiting text block")
	}
	return
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

func isEndOfFile(r rune) bool {
	return r < 0
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isExtendedChar(r rune) bool {
	return strings.ContainsRune("!@#$%%^&*()_./+-/;:|\\=", r)
}
