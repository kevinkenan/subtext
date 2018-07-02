package parse

import (
	"fmt"
	"strings"
	"testing"
)

type tokenList []*token

func (tl tokenList) String() string {
	s := new(strings.Builder)
	for _, t := range tl {
		s.WriteString(t.String())
	}
	return s.String()
}

type testCase struct {
	name  string    // The name of the test.
	input string    // The input to be tested.
	exp   tokenList // The expected output.
	loc   bool      // Indicates if we should test loc related data.
}

func newCase(name, input string, exp tokenList) testCase {
	return testCase{name, input, exp, false}
}

func newLocCase(name, input string, exp tokenList) testCase {
	return testCase{name, input, exp, true}
}

// tkn builds a partially complete token
func tkn(typeof tokenType, text string) *token {
	return &token{
		typeof: typeof,
		value:  text}
}

var (
	tText                = tkn(tokenText, "")
	tCmdStart            = tkn(tokenCmdStart, "")
	tName                = tkn(tokenName, "")
	tRunes               = tkn(tokenRunes, "")
	tEmptyLine           = tkn(tokenEmptyLine, "")
	tIndent              = tkn(tokenIndent, "")
	tEOLComment          = tkn(tokenEOLComment, "")
	tToggleComment       = tkn(tokenToggleComment, "")
	tLineBreak           = tkn(tokenLineBreak, "")
	tLeftParenthesis     = tkn(tokenLeftParenthesis, "")
	tRightParenthesis    = tkn(tokenRightParenthesis, "")
	tLeftSquare          = tkn(tokenLeftSquare, "")
	tRightSquare         = tkn(tokenRightSquare, "")
	tLeftCurly           = tkn(tokenLeftCurly, "")
	tRightCurly          = tkn(tokenRightCurly, "")
	tLeftAngle           = tkn(tokenLeftAngle, "")
	tRightAngle          = tkn(tokenRightAngle, "")
	tEqual               = tkn(tokenEqual, "")
	tComma               = tkn(tokenComma, "")
	tTilde               = tkn(tokenTilde, "")
	tSpaceEater          = tkn(tokenSpaceEater, "")
	tError               = tkn(tokenError, "")
	tEOF                 = tkn(tokenEOF, "")
	tSysCmdStart         = tkn(tokenSysCmdStart, "")
	tSysCmd              = tkn(tokenSysCmd, "")
	tParScanOn           = tkn(tokenParScanOn, "")
	tParScanOff          = tkn(tokenParScanOff, "")
	tCmdParagraphModeOff = tkn(tokenCmdParagraphModeOff, "")
	tCmdParagraphModeOn  = tkn(tokenCmdParagraphModeOn, "")
)

// Test Cases -----------------------------------------------------------------

var commonTestCases = []testCase{
	newCase("empty", "", tokenList{
		tEOF}),
	newCase("spaces", " \t\n", tokenList{
		tEmptyLine,
		tLineBreak,
		tEOF}),
	newCase("text", "text", tokenList{
		tText,
		tEOF}),
	newCase("text with linebreaks", "  \n  \n  text  \n     \n   ", tokenList{
		tEmptyLine,
		tLineBreak,
		tEmptyLine,
		tLineBreak,
		tIndent,
		tText,
		tLineBreak,
		tEmptyLine,
		tLineBreak,
		tEmptyLine,
		tEOF}),
	newCase("text with carriage return and newline", "text\r\ntext", tokenList{
		tText,
		tLineBreak,
		tText,
		tEOF}),
	newCase("text with just a carriage return", "text\rtext", tokenList{
		tText,
		tEOF}),
	newCase("simple unicode text", "A ƒ.\nB .", tokenList{
		tText,
		tLineBreak,
		tText,
		tEOF}),
	newCase("simple literal text", "1`•3{}[]", tokenList{
		tText,
		tText,
		tEOF}),
	// newCase("literal `", "1``3", []token{
	// 	tokenText, "1"),
	// 	tokenText, "`3"),
	// 	tEOF}),
	// newLocCase("location", "123456", []token{
	// 	token{tokenText, 0, 1, "123456"},
	// 	token{tokenEOF, 6, 1, ""}}),
	// newLocCase("two lines", "012\n345", []token{
	// 	token{tokenText, 0, 2, "012\n345"},
	// 	token{tokenEOF, 7, 2, ""}}),

	// Space Eater Tests
	newCase("basic space eater", "•% \n  \n text", tokenList{
		tSpaceEater,
		tEmptyLine,
		tLineBreak,
		tEmptyLine,
		tLineBreak,
		tIndent,
		tText,
		tEOF}),
	newCase("end of bare macro space eater", "•a% \n\n 1", tokenList{
		tCmdStart,
		tName,
		tSpaceEater,
		tEmptyLine,
		tLineBreak,
		tLineBreak,
		tIndent,
		tText,
		tEOF}),
	newCase("end of short macro space eater", "•a{}% \n\n 1", tokenList{
		tCmdStart,
		tName,
		tLeftCurly,
		tRightCurly,
		tSpaceEater,
		tEmptyLine,
		tLineBreak,
		tLineBreak,
		tIndent,
		tText,
		tEOF}),
	newCase("end of extended macro space eater", "•a[{}]% \n\n 1", tokenList{
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tRightCurly,
		tRightSquare,
		tSpaceEater,
		tEmptyLine,
		tLineBreak,
		tLineBreak,
		tIndent,
		tText,
		tEOF}),

	// Cmd Tests
	newCase("basic command", "1 •cmd 2", tokenList{
		tText,
		tCmdStart,
		tName,
		tText,
		tEOF}),
	newCase("simple command with empty body", "1•2{}{}3", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tRightCurly,
		tText,
		tEOF}),
	newCase("simple command with body", "1•2{3}4", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tRightCurly,
		tText,
		tEOF}),
	newCase("two text blocks with nested command", "1•2[{a•b{c}}]4", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("simple command with confusing body", "1•2{{3}}4", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tRightCurly,
		tText,
		tEOF}),
	newCase("simple star command", "1•2*{3}\n*}4", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tLineBreak, // if the star didn't work, the linebreak would be after the '}'
		tRightCurly,
		tText,
		tEOF}),
	newCase("command with empty list", "1•2[]3", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tRightSquare,
		tText,
		tEOF}),
	newCase("command with partial list", "1•2[3=]4", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tRightSquare,
		tText,
		tEOF}),
	newCase("command with empty text in a list", "1•2[ {} ]3", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("semi", "1•2[ ;{} ]3", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tError}),
	newCase("command with two args", "1•2[ {3} {4} ]5", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tText,
		tRightCurly,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("command with named argument", "1•2[ 3 = {4} ]5", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("command with incorrect starred argument", "1•2[ 3={4}*} a={b} ]5", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tError}),
	newCase("command with named and starred argument", "1•2[ 3*={4}*} a={b} ]5", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("command with eolComment in body text", "1•2{3•|4\n5}6", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tEOLComment,
		tText,
		tLineBreak,
		tText,
		tRightCurly,
		tText,
		tEOF}),
	newCase("command with two line body", "1•2{3\n4}5", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tLineBreak,
		tText,
		tRightCurly,
		tText,
		tEOF}),
	newCase("unterminated command body", "•1{2\nstuff", tokenList{
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tLineBreak,
		tError}),
	// newCase("command with unclosed toggle comment in body", "•1{2◊3", tokenList{
	//	tCmdStart,
	//	tName,
	//	tLeftCurly,
	//	tText,
	//	tError,
	newCase("@nested commands", "•1{2 •3{4} }5", tokenList{
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tRightCurly,
		tText,
		tRightCurly,
		tText,
		tEOF}),
	newCase("tightly nested commands", "•1{•2{3}}4", tokenList{
		tCmdStart,
		tName,
		tLeftCurly,
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightCurly,
		tText,
		tEOF}),
	newCase("tightly nested command lists", "•1[{•2[{3}]}]4", tokenList{
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("just a command char", "•", tokenList{
		tError}),
	newCase("unnamed command", "1•", tokenList{
		tText,
		tError}),
	newCase("bullet char", "1`•2", tokenList{
		tText,
		tText,
		tEOF}),
	newCase("two command anonymous args on different lines", "•1[\n2={a}\n3={b}]4", tokenList{
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("unclosed single command with an incomplete named argument", "1•2[test\n", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tError}),

	// Flags
	newCase("command with flags", "•2[<3=4 5>]", tokenList{
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftAngle,
		tRunes,
		tRunes,
		tRightAngle,
		tRightSquare,
		tEOF}),
	newCase("command with flags", "1•2[<3, ~4>{5}]", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftAngle,
		tRunes,
		tComma,
		tTilde,
		tRunes,
		tRightAngle,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tEOF}),

	newCase("everything", "1•2[<3>4={5}]", tokenList{
		tText,
		tCmdStart,
		tName,
		tLeftSquare,
		tLeftAngle,
		tRunes,
		tRightAngle,
		tName,
		tEqual,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tEOF}),

	// EOL Comments
	newCase("eolComment", "1•|2\n3", tokenList{
		tText,
		tEOLComment,
		tText,
		tLineBreak,
		tText,
		tEOF}),
	newCase("eolComment at EOL", "1•|\n3", tokenList{
		tText,
		tEOLComment,
		tLineBreak,
		tText,
		tEOF}),
	newCase("eolComment at EOF", "1•|23", tokenList{
		tText,
		tEOLComment,
		tText,
		tEOF}),
	// newLocCase("eolComment with embedded command", "1•|•2\n3", tokenList{
	// 	t{tokenText,
	// 	t{tokenText,
	// 	t{tokenEOF,
	newCase("eolComment with CR", "1•|2\r\n3", tokenList{
		tText,
		tEOLComment,
		tText,
		tLineBreak,
		tText,
		tEOF}),
	newCase("eolComment followed by toggle comment", "1•|◊\n2◊3", tokenList{
		tText,
		tEOLComment,
		tToggleComment,
		tLineBreak,
		tText,
		tToggleComment,
		tText,
		tEOF}),

	// Toggle comment tests.
	// newLocCase("basic toggle comment", "1◊longtext◊3\n4", tokenList{
	// 	t{tokenText,
	// 	t{tokenText,
	// 	t{tokenEOF,
	newCase("lozenge", "1`◊2", tokenList{
		tText,
		tText,
		tEOF}),
	newCase("toggle comment entire string", "◊this is everythging", tokenList{
		tToggleComment,
		tText,
		tEOF}),
	newCase("toggle comment entire string after one char", "1◊2", tokenList{
		tText,
		tToggleComment,
		tText,
		tEOF}),
	// newLocCase("toggle comment eats newline", "1◊\n2◊3", tokenList{
	// 	t{tokenText,
	// 	t{tokenText,
	// 	t{tokenEOF,
	// newLocCase("single toggle comment", "1◊2", tokenList{
	// 	t{tokenText,
	// 	t{tokenEOF,
	// newLocCase("toggle comment begins text", "◊1◊2", tokenList{
	// 	t{tokenText,
	// 	t{tokenEOF,
	newCase("toggle comment in command body", "•1{2◊3◊4}", tokenList{
		tCmdStart,
		tName,
		tLeftCurly,
		tText,
		tToggleComment,
		tText,
		tToggleComment,
		tText,
		tRightCurly,
		tEOF}),
	newCase("toggle comment in command list", "•1[2=◊{3}◊{4}]", tokenList{
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tToggleComment,
		tLeftCurly,
		tText,
		tRightCurly,
		tToggleComment,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tEOF}),
	newCase("toggle comment in command list span", "•1[2={◊3}{◊4}]", tokenList{
		tCmdStart,
		tName,
		tLeftSquare,
		tName,
		tEqual,
		tLeftCurly,
		tToggleComment,
		tText,
		tRightCurly,
		tLeftCurly,
		tToggleComment,
		tText,
		tRightCurly,
		tRightSquare,
		tEOF}),

	// sysCmd Tests
	newCase("bare system commands", "1•(a)2", tokenList{
		tText,
		tSysCmdStart,
		tName,
		tText,
		tEOF}),
	newCase("simple system command", "1•(a){b}2", tokenList{
		tText,
		tSysCmdStart,
		tName,
		tLeftCurly,
		tText,
		tRightCurly,
		tText,
		tEOF}),
	newCase("extended system command", "1•(a)[{b}]2", tokenList{
		tText,
		tSysCmdStart,
		tName,
		tLeftSquare,
		tLeftCurly,
		tText,
		tRightCurly,
		tRightSquare,
		tText,
		tEOF}),
	newCase("empty system command", "1•()2", tokenList{
		tText,
		tSysCmdStart,
		tName,
		tText,
		tEOF}),
	newCase("system command with space eater", "1•()%  \n2", tokenList{
		tText,
		tSysCmdStart,
		tName,
		tSpaceEater,
		tEmptyLine,
		tLineBreak,
		tText,
		tEOF}),
	newCase("star system command", "1•(a*){b}\n*}2", tokenList{
		tText,
		tSysCmdStart,
		tName,
		tLeftCurly,
		tText,
		tLineBreak,
		tRightCurly,
		tText,
		tEOF}),
}

var plainParTestCases = []testCase{
	// 	// newCase("plain normal paragraph handling", "1\n\n2", tokenList{
	// 	//	tText,
	// 	// 	tEOF}),
	// 	// newCase("plain paragraph with spaces", "1\n \n2", tokenList{
	// 	//	tText,
	// 	// 	tEOF}),
	// 	// newCase("plain basic parmode", "1\n\n   \n  2", tokenList{
	// 	//	tText,
	// 	// 	tEOF}),
	// 	// newCase("plain paragraphs in command body", "•2{A\n \nB}3", tokenList{
	// 	//	tCmdStart,
	// 	//	tName,
	// 	//	tLeftCurly,
	// 	//	tText,
	// 	//	tRightCurly,
	// 	//	tText,
	// 	// 	tEOF}),
	// 	// newCase("plain paragraphs and spaces in command body", "1•2{\n\n    \n3}4", tokenList{
	// 	//	tText,
	// 	//	tCmdStart,
	// 	//	tName,
	// 	//	tLeftCurly,
	// 	//	tText,
	// 	//	tRightCurly,
	// 	//	tText,
	// 	// 	tEOF}),
	// 	// newCase("plain multiple double linebreaks", "1\n\n\n\n\n\n2", tokenList{
	// 	//	tText,
	// 	// 	tEOF}),
}

// Paragraph generation is off for these tests.

func TestPlainScanner(t *testing.T) {
	scannerPlainTest(t, commonTestCases)
	scannerPlainTest(t, plainParTestCases)
}

func scannerPlainTest(t *testing.T, tests []testCase) bool {
	errors := false
	for _, tc := range tests {
		result := runPlainTest(tc)
		expResult := tc.exp
		// expResult := getOuterParSeq("begin", "")
		// expResult = append(expResult, tc.exp...)
		// expResult = append(expResult, getOuterParSeq("end", "")...)
		// verbose.Off()
		// if tc.name == "" {
		// 	fmt.Println("On--------------------------------")
		// 	verbose.On()
		// }
		if !equalType(result, expResult) {
			errors = true
			t.Errorf("> %s (ignoring location/linenumbers)\n    *result:  \n%s\n    *expected:\n%s\n", tc.name, result, expResult)
		}
	}
	return errors
}

// runTest also gathers the emitted tokens into a slice.
func runPlainTest(t testCase) (tokens tokenList) {
	s := scan(t.name, t.input, true)
	for {
		token := s.nextToken()
		tokens = append(tokens, &token)
		if token.typeof == tokenEOF || token.typeof == tokenError {
			break
		}
	}
	return
}

// func newParSeq(typ, text string) (tkns []token) {
// 	tkns = []token{
// 		tkn(tokenCmdStart, ""),
// 		tkn(tokenName, "sys.paragraph."+typ),
// 		tkn(tokenLeftSquare, "["),
// 		tkn(tokenLeftCurly, "{"),
// 		tkn(tokenText, text),
// 		tkn(tokenRightCurly, "}"),
// 		tkn(tokenRightSquare, "]"),
// 	}
// 	return
// }

// func emptyCmdSeq(name, mode string) (tkns []token) {
// 	tkns = []token{
// 		tkn(tokenCmdStart, mode),
// 		tkn(tokenName, name),
// 		tkn(tokenLeftCurly, "{"),
// 		tkn(tokenRightCurly, "}"),
// 	}
// 	return
// }

// func newCmdSeq(name, mode, val string) (tkns []token) {
// 	tkns = []token{
// 		tkn(tokenCmdStart, mode),
// 		tkn(tokenName, name),
// 		tkn(tokenLeftCurly, "{"),
// 		tkn(tokenText, val),
// 		tkn(tokenRightCurly, "}"),
// 	}
// 	return
// }

// func newCmdSeqPre(name, mode string) (tkns []token) {
// 	tkns = []token{
// 		tkn(tokenCmdStart, mode),
// 		tkn(tokenName, name),
// 		tkn(tokenLeftCurly, "{"),
// 	}
// 	return
// }

// func newCmdSeqPost() (tkns []token) {
// 	tkns = []token{
// 		tkn(tokenRightCurly, "}"),
// 	}
// 	return
// }

// type myTestCases struct {
// 	tokens []token
// }

// func NewTC() *myTestCases {
// 	return &myTestCases{[]token{}}
// }
// func (m *myTestCases) add(t []token) *myTestCases {
// 	m.tokens = append(m.tokens, t...)
// 	return m
// }
// func tkns(ttype tokenType, val string) []token {
// 	return []token{tkn(ttype, val)}
// }

// Test scan -------------------------------------------------------------

var parGenTestCases = []testParCase{
	// 	newParCase("just text", "1", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("middle paragraph", "1\n\n2", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("multiple paragraphs", "1\n\n2\n\n3\n\n4\n", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "4"))
	// 		t.add(newParSeq("end", "\n"))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("initial paragraph and whitespace", "\n\n \n \n1", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("busy middle paragraphs", "1\n\n \n\n\n2", "", "\n\n \n\n\n", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n \n\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("simple terminal paragraph", "1\n\n2\n\n3\n", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", "\n"))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("terminal paragraph", "1\n\n \n\n\n", "", "\n\n \n\n\n", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n \n\n\n"))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("middle paragraphs", "\n \n1\n\n \n2\n \n ", "\n \n", "\n \n ", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n \n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", "\n \n "))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("turn off par scanning", "1\n\n¶-\n\n2\n\n¶+3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(tkns(tokenText, "\n\n2\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("vertical mode command", "1\n\n§a{xyz}\n\n3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newCmdSeq("a", "V", "xyz"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("vertical mode command with trailing text", "1\n\n§a{xyz}abc\n\n3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newCmdSeq("a", "V", "xyz"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "abc"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("initial space and command", "•a{}12", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(emptyCmdSeq("a", "H"))
	// 		t.add(tkns(tokenText, "12"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("vertical mode cmd with 2 args", "§a[{xyz}{a\n\nX}] •abc\n\n3", "", "", func() []token {
	// 		t := NewTC()
	// 		// t.add(newParSeq("begin", ""))
	// 		// t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenCmdStart, "V"))
	// 		t.add(tkns(tokenName, "a"))
	// 		t.add(tkns(tokenLeftSquare, "["))
	// 		t.add(tkns(tokenLeftCurly, "{"))
	// 		t.add(tkns(tokenText, "xyz"))
	// 		t.add(tkns(tokenRightCurly, "}"))
	// 		t.add(tkns(tokenLeftCurly, "{"))
	// 		t.add(tkns(tokenText, "a\n\nX"))
	// 		t.add(tkns(tokenRightCurly, "}"))
	// 		t.add(tkns(tokenRightSquare, "]"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenCmdStart, "H"))
	// 		t.add(tkns(tokenName, "abc"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("simple command arg", "1\n\n•2{A}3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(newCmdSeq("2", "H", "A"))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("line breaks in command arg", "•2{A\n\nB}3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(newCmdSeqPre("2", "H"))
	// 		t.add(tkns(tokenText, "A"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "B"))
	// 		t.add(tkns(tokenRightCurly, "}"))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("vertical mode inside a horizontal", "•2{a§X{Y}b}3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(newCmdSeqPre("2", "H"))
	// 		t.add(tkns(tokenText, "a"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(newCmdSeq("X", "V", "Y"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "b"))
	// 		t.add(tkns(tokenRightCurly, "}"))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("horizontal mode inside a vertical command", "§1{a•X{Y}b}3", "", "", func() []token {
	// 		t := NewTC()
	// 		// t.add(newParSeq("begin", ""))
	// 		// t.add(newParSeq("end", ""))
	// 		t.add(newCmdSeqPre("1", "V"))
	// 		t.add(tkns(tokenText, "a"))
	// 		t.add(newCmdSeq("X", "H", "Y"))
	// 		t.add(tkns(tokenText, "b"))
	// 		t.add(tkns(tokenRightCurly, "}"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("pars in command arg", "1\n\n•2{¶+A\n\nB}3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(newCmdSeqPre("2", "H"))
	// 		t.add(tkns(tokenText, "A"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "B"))
	// 		t.add(newCmdSeqPost())
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("multiple double linebreaks", "1\n\n\n\n\n\n2", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n\n\n\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("final paragraph", "1\n\n\n\n\n\n", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", "\n\n\n\n\n\n"))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("initial paragraph", "\n\n  \n\n1", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "1"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("parScanFlag off at beginning", "¶-1\n\n2\n\n3\n\n4\n", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(tkns(tokenText, "1\n\n2\n\n3\n\n4\n"))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("parScanFlag off at beginning with command", "¶-1\n\n•a{2}\n\n", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(tkns(tokenText, "1\n\n"))
	// 		t.add(newCmdSeq("a", "H", "2"))
	// 		t.add(tkns(tokenText, "\n\n"))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	newParCase("two calls to parScanFlag", "¶-1\n\n¶+2\n\n3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(tkns(tokenText, "1\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
	// 	// newParCase("use ¶+ to supress paragraph break", "¶-1\n\n¶+2\n¶+\n3", "", "", func() []token {
	// 	// 	t := NewTC()
	// 	// 	t.add(tkns(tokenText, "1\n\n"))
	// 	// 	t.add(newParSeq("begin", ""))
	// 	// 	t.add(tkns(tokenText, "2\n"))
	// 	// 	t.add(tkns(tokenText, "\n3"))
	// 	// 	t.add(newParSeq("end", ""))
	// 	// 	t.add(tkns(tokenEOF, ""))
	// 	// 	return t.tokens
	// 	// }()),
	// 	newParCase("consecutive ¶+", "¶-1\n\n¶+¶+¶+2\n\n3", "", "", func() []token {
	// 		t := NewTC()
	// 		t.add(tkns(tokenText, "1\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "2"))
	// 		t.add(newParSeq("end", "\n\n"))
	// 		t.add(newParSeq("begin", ""))
	// 		t.add(tkns(tokenText, "3"))
	// 		t.add(newParSeq("end", ""))
	// 		t.add(tkns(tokenEOF, ""))
	// 		return t.tokens
	// 	}()),
}

type testParCase struct {
	name    string    // The name of the test.
	input   string    // The input to be tested.
	expInit string    // The expected initial white space.
	expTerm string    // The expected terminal white space.
	exp     tokenList // The expected output.
	loc     bool      // Indicates if we should test loc related data.
}

func newParCase(name, input, init, term string, exp tokenList) testParCase {
	return testParCase{name, input, init, term, exp, false}
}

// func TestScannerEmpty(t *testing.T) {
// 	var tokens []token
// 	scnr := scan("empty", "\n\n  ", true)
// 	for {
// 		token := scnr.nextToken()
// 		tokens = append(tokens, token)
// 		if token.typeof == tokenEOF || token.typeof == tokenError {
// 			break
// 		}
// 	}

// 	fmt.Println(tokens)
// 	switch {
// 	case tokens == nil:
// 		t.Errorf("No tokens in TestScannerEmpty")
// 	case len(tokens) != 1:
// 		t.Errorf("Wrong number of args in TestScannerEmpty. Got: %d", len(tokens))
// 	// case tokens[0].typeof != tokenText:
// 	// 	t.Errorf("Wrong token type in TestScannerEmpty")
// 	case tokens[0].typeof != tokenEOF:
// 		t.Errorf("Wrong token type in TestScannerEmpty %s", tokenTypeLookup(tokens[1].typeof))
// 	}
// }

// Paragraph generation is on for these tests. Verification of the standard
// initial paragraph command sequence is built into the test code so we don't
// need to specify that sequence in each test.

// func TestScanner(t *testing.T) {
// 	e := scanTest(t, parGenTestCases)
// 	if e {
// 		printTokenCodes()
// 	}
// }

func scanTest(t *testing.T, tests []testParCase) bool {
	errors := false
	tnum := -1
	var start, end = 0, len(tests)
	if tnum > 0 {
		start = tnum
		end = tnum + 1
	}
	for tn, tc := range tests[start:end] {
		// verbose.Off()
		// for tn, tc := range tests {
		result := runTest(&tc)
		expResult := tc.exp
		if !equal(result, expResult, tc.loc) {
			errors = true
			if tc.loc {
				// t.Errorf("> %s\n  *result:\n  %+v\n  *expected:\n%v\n", tc.name, result, expResult)
				t.Errorf(">[%d] %s\n  *result:\n  %+v\n", tn, tc.name, result)
			} else {
				// t.Errorf("> %s (ignoring location/linenumbers)\n    *result:\n%+v\n    *expected:%v\n", tc.name, result, expResult)
				t.Errorf(">[%d] %s (ignoring location/linenumbers)\n    *result:\n%+v\n", tn, tc.name, result)
				// fmt.Println(expResult)
			}
		}
	}
	return errors
}

func getOuterParSeq(parType, text string) tokenList {
	tkns := tokenList{
		tkn(tokenCmdStart, ""),
		tkn(tokenName, "sys.paragraph."+parType),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, text),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
	}
	if parType == "end" {
		tkns = append(tkns, tEOF)
	}
	return tkns
}

func verifyOuterPar(t *testing.T, ts tokenList, parType, space, name string) bool {
	exp := tokenList{
		tkn(tokenCmdStart, ""),
		tkn(tokenName, "sys.paragraph."+parType),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, space),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
	}
	if parType == "end" {
		exp = append(exp, tEOF)
	}
	if !equal(ts, exp, false) {
		t.Errorf("> %s (%s paragraph verification error)\n    *result:  %+v\n    *expected:%v\n", name, parType, ts, exp)
		return false
	}
	return true
}

// runTest also gathers the emitted tokens into a slice.
func runTest(t *testParCase) (tokens tokenList) {
	s := scan(t.name, t.input, false)
	for {
		token := s.nextToken()
		tokens = append(tokens, &token)
		if token.typeof == tokenEOF || token.typeof == tokenError {
			break
		}
	}
	return
}

// Utils ----------------------------------------------------------------------

func equalType(t1, t2 tokenList) bool {
	if len(t1) != len(t2) {
		return false
	}
	for k := range t1 {
		if t1[k].typeof != t2[k].typeof {
			return false
		}
	}
	return true
}

func equalValue(t1, t2 tokenList, checkLoc bool) bool {
	if len(t1) != len(t2) {
		return false
	}
	for k := range t1 {
		if t1[k].typeof != t2[k].typeof {
			return false
		}
		if t1[k].value != t2[k].value {
			return false
		}
	}
	return true
}

func equal(t1, t2 tokenList, checkLoc bool) bool {
	if len(t1) != len(t2) {
		return false
	}
	for k := range t1 {
		if t1[k].typeof != t2[k].typeof {
			return false
		}
		if t1[k].value != t2[k].value {
			return false
		}
		if checkLoc && t1[k].loc != t2[k].loc {
			return false
		}
		if checkLoc && t1[k].lnum != t2[k].lnum {
			return false
		}
	}
	return true
}

var printed bool

func printTokenCodes() {
	if !printed {
		printed = true
		for i, v := range tokenNames {
			fmt.Printf("%d %s\n", i, v)
		}
	}
}
