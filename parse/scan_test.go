package parse

import (
	"fmt"
	"testing"
)

type testCase struct {
	name  string  // The name of the test.
	input string  // The input to be tested.
	exp   []token // The expected output.
	loc   bool    // Indicates if we should test loc related data.
}

func newCase(name, input string, exp []token) testCase {
	return testCase{name, input, exp, false}
}

func newLocCase(name, input string, exp []token) testCase {
	return testCase{name, input, exp, true}
}

// tkn builds a partially complete token
func tkn(typeof tokenType, text string) token {
	return token{
		typeof: typeof,
		value:  text}
}

var (
	tEOF = tkn(tokenEOF, "")
)

// Test Cases -----------------------------------------------------------------

var commonTestCases = []testCase{
	newCase("empty", "", []token{tEOF}),
	newCase("spaces", " \t\n", []token{
		tkn(tokenText, " \t\n"),
		tEOF}),
	newCase("text", "text", []token{
		tkn(tokenText, "text"),
		tEOF}),
	newCase("simple unicode text", "A ƒ.\nB .", []token{
		tkn(tokenText, "A ƒ.\nB ."),
		tEOF}),
	newCase("simple literal text", "1`•3{}[]", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "•3{}[]"),
		tEOF}),
	newCase("literal `", "1``3", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "`3"),
		tEOF}),
	newLocCase("location", "123456", []token{
		token{tokenText, 0, 1, "123456"},
		token{tokenEOF, 6, 1, ""}}),
	newLocCase("two lines", "012\n345", []token{
		token{tokenText, 0, 2, "012\n345"},
		token{tokenEOF, 7, 2, ""}}),

	// Space Eater Tests
	newLocCase("basic space eater", "•% 1•%           \t2", []token{
		token{tokenText, 5, 1, "1"},
		token{tokenText, 22, 1, "2"},
		token{tokenEOF, 23, 1, ""}}),
	newLocCase("double space eater", "•% •% 1", []token{
		token{tokenText, 10, 1, "1"},
		token{tokenEOF, 11, 1, ""}}),
	newLocCase("multiline space eater", "•% \n\n 1", []token{
		token{tokenText, 8, 3, "1"},
		token{tokenEOF, 9, 3, ""}}),
	newCase("end of bare macro space eater", "•a% \n\n 1", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "a"),
		tkn(tokenText, "1"),
		tEOF}),
	newCase("end of short macro space eater", "•a{}% \n\n 1", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "a"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "1"),
		tEOF}),
	newCase("end of extended macro space eater", "•a[{}]% \n\n 1", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "a"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "1"),
		tEOF}),

	// Cmd Tests
	newCase("basic command", "1•cmd 2", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "cmd"),
		tkn(tokenText, " 2"),
		tEOF}),
	newCase("simple command with empty body", "1•2{}{}3", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "{}3"),
		tEOF}),
	newCase("two text blocks with nested command", "1•2[{a•b{}}]4", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "a"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "b"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("simple command with body", "1•2{3}4", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("simple command with confusing body", "1•2{{3}}4", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "{3"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "}4"),
		tEOF}),
	newCase("simple star command", "1•2*{3}*}4", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3}"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("command with empty list", "1•2[]3", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "3"),
		tEOF}),
	newCase("command with partial list", "1•2[3=]4", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "3"),
		tkn(tokenEqual, "="),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("command with empty text in a list", "1•2[ {} ]3", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "3"),
		tEOF}),
	newCase("semi", "1•2[ ;{} ]3", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenError, "invalid character ';' in command body")}),
	newCase("command with two args", "1•2[ {3} {4} ]5", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "4"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "5"),
		tEOF}),
	newCase("command with named argument", "1•2[ 3 = {4} ]5", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "3"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "4"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "5"),
		tEOF}),
	newCase("command with named and starred argument", "1•2[ 3*={4}*} a={b} ]5", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "3"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "4}"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenName, "a"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "b"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "5"),
		tEOF}),
	newCase("command with eolComment in body text", "1•2{3•|4\n5}6", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3"),
		tkn(tokenText, "5"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "6"),
		tEOF}),
	newCase("command with two line body", "1•2{3\n4}5", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3\n4"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "5"),
		tEOF}),
	newCase("unterminated command body", "•1{2\nstuff", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenError, "end of file while command is still open")}),
	// newCase("command with unclosed toggle comment in body", "•1{2◊3", []token{
	// 	tkn(tokenCmdStart, "H"),
	// 	tkn(tokenName, "1"),
	// 	tkn(tokenLeftCurly, "{"),
	// 	tkn(tokenText, "2"),
	// 	tkn(tokenError, "unterminated command body")}),
	newCase("@nested commands", "•1{2 •3{4} }5", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "2 "),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "3"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "4"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, " "),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "5"),
		tEOF}),
	newCase("tightly nested commands", "•1{•2{3}}4", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("tightly nested command lists", "•1[{•2[{3}]}]4", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "3"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("just a command char", "•", []token{
		tkn(tokenError, "unnamed command")}),
	newCase("unnamed command", "1•", []token{
		tkn(tokenText, "1"),
		tkn(tokenError, "unnamed command")}),
	newCase("bullet char", "1`•2", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "•2"),
		tEOF}),
	newCase("two command anonymous args on different lines", "•1[\n2={a}\n3={b}]4", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "2"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "a"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenName, "3"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "b"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("unclosed single command with an incomplete named argument", "1•2[test\n", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "test"),
		tkn(tokenError, "end of file while processing command")}),

	// Flags
	newCase("command with flags", "•2[<3=4 5>]", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftAngle, "<"),
		tkn(tokenRunes, "3=4"),
		tkn(tokenRunes, "5"),
		tkn(tokenRightAngle, ">"),
		tkn(tokenRightSquare, "]"),
		tEOF}),
	newCase("command with flags", "1•2[<3, ~4>{5}]", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftAngle, "<"),
		tkn(tokenRunes, "3"),
		tkn(tokenComma, ","),
		tkn(tokenTilde, "~"),
		tkn(tokenRunes, "4"),
		tkn(tokenRightAngle, ">"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "5"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tEOF}),

	newCase("everything", "1•2[<3>4={5}]", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftAngle, "<"),
		tkn(tokenRunes, "3"),
		tkn(tokenRightAngle, ">"),
		tkn(tokenName, "4"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "5"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tEOF}),

	// EOL Comments
	newCase("eolComment at EOF", "1•|23", []token{
		tkn(tokenText, "1"),
		tEOF}),
	newCase("eolComment", "1•|2\n3", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "3"),
		tEOF}),
	newLocCase("eolComment with embedded command", "1•|•2\n3", []token{
		token{tokenText, 0, 1, "1"},
		token{tokenText, 10, 2, "3"},
		token{tokenEOF, 11, 2, ""}}),
	newCase("eolComment with CR", "1•|2\r\n3", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "3"),
		tEOF}),
	newCase("eolCOmment followed by toggle comment", "1•|◊\n2◊3", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "2"),
		tEOF}),

	// Toggle comment tests.
	newLocCase("basic toggle comment", "1◊longtext◊3\n4", []token{
		token{tokenText, 0, 1, "1"},
		token{tokenText, 15, 2, "3\n4"},
		token{tokenEOF, 18, 2, ""}}),
	newCase("lozenge", "1`◊2", []token{
		tkn(tokenText, "1"),
		tkn(tokenText, "◊2"),
		tEOF}),
	newCase("toggle comment entire string", "◊123", []token{tEOF}),
	newCase("toggle comment entire string after one char", "1◊", []token{
		tkn(tokenText, "1"),
		tEOF}),
	newLocCase("toggle comment eats newline", "1◊\n2◊3", []token{
		token{tokenText, 0, 1, "1"},
		token{tokenText, 9, 2, "3"},
		token{tokenEOF, 10, 2, ""}}),
	newLocCase("single toggle comment", "1◊2", []token{
		token{tokenText, 0, 1, "1"},
		token{tokenEOF, 5, 1, ""}}),
	newLocCase("toggle comment begins text", "◊1◊2", []token{
		token{tokenText, 7, 1, "2"},
		token{tokenEOF, 8, 1, ""}}),
	newCase("toggle comment in command body", "•1{2◊3◊4}", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "2"),
		tkn(tokenText, "4"),
		tkn(tokenRightCurly, "}"),
		tEOF}),
	newCase("toggle comment in command list", "•1[2=◊{3}◊{4}]", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "2"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "4"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tEOF}),
	newCase("toggle comment in command list span", "•1[2={◊3}{◊4}]", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "1"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenName, "2"),
		tkn(tokenEqual, "="),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "4"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tEOF}),

	// sysCmd Tests
	newCase("bare system commands", "1•(a)2", []token{
		tkn(tokenText, "1"),
		tkn(tokenSysCmdStart, ""),
		tkn(tokenName, "a"),
		tkn(tokenText, "2"),
		tEOF}),
	newCase("simple system command", "1•(a){b}2", []token{
		tkn(tokenText, "1"),
		tkn(tokenSysCmdStart, ""),
		tkn(tokenName, "a"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "b"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "2"),
		tEOF}),
	newCase("extended system command", "1•(a)[{b}]2", []token{
		tkn(tokenText, "1"),
		tkn(tokenSysCmdStart, ""),
		tkn(tokenName, "a"),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "b"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
		tkn(tokenText, "2"),
		tEOF}),
	newCase("empty system command", "1•()2", []token{
		tkn(tokenText, "1"),
		tkn(tokenSysCmdStart, ""),
		tkn(tokenName, ""),
		tkn(tokenText, "2"),
		tEOF}),
	newCase("star system command", "1•(a*){b}*}2", []token{
		tkn(tokenText, "1"),
		tkn(tokenSysCmdStart, ""),
		tkn(tokenName, "a"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "b}"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "2"),
		tEOF}),
}

var plainParTestCases = []testCase{
	newCase("plain normal paragraph handling", "1\n\n2", []token{
		tkn(tokenText, "1\n\n2"),
		tEOF}),
	newCase("plain paragraph with spaces", "1\n \n2", []token{
		tkn(tokenText, "1\n \n2"),
		tEOF}),
	newCase("plain basic parmode", "1\n\n   \n  2", []token{
		tkn(tokenText, "1\n\n   \n  2"),
		tEOF}),
	newCase("plain paragraphs in command body", "•2{A\n \nB}3", []token{
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "A\n \nB"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "3"),
		tEOF}),
	newCase("plain paragraphs and spaces in command body", "1•2{\n\n    \n3}4", []token{
		tkn(tokenText, "1"),
		tkn(tokenCmdStart, "H"),
		tkn(tokenName, "2"),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, "\n\n    \n3"),
		tkn(tokenRightCurly, "}"),
		tkn(tokenText, "4"),
		tEOF}),
	newCase("plain multiple double linebreaks", "1\n\n\n\n\n\n2", []token{
		tkn(tokenText, "1\n\n\n\n\n\n2"),
		tEOF}),
}

// Paragraph generation is off for these tests.

func TestPlainScanner(t *testing.T) {
	e := scannerPlainTest(t, commonTestCases)
	e = scannerPlainTest(t, plainParTestCases) || e
	if e {
		printTokenCodes()
	}
}

func scannerPlainTest(t *testing.T, tests []testCase) bool {
	errors := false
	for _, tc := range tests {
		result := runPlainTest(&tc)
		expResult := tc.exp
		// expResult := getOuterParSeq("begin", "")
		// expResult = append(expResult, tc.exp...)
		// expResult = append(expResult, getOuterParSeq("end", "")...)
		// verbose.Off()
		// if tc.name == "" {
		// 	fmt.Println("On--------------------------------")
		// 	verbose.On()
		// }
		if !equal(result, expResult, tc.loc) {
			errors = true
			if tc.loc {
				t.Errorf("> %s\n  *result:  %+v\n  *expected:%v\n", tc.name, result, expResult)
			} else {
				t.Errorf("> %s (ignoring location/linenumbers)\n    *result:  %+v\n    *expected:%v\n", tc.name, result, expResult)
			}
		}
	}
	return errors
}

// runTest also gathers the emitted tokens into a slice.
func runPlainTest(t *testCase) (tokens []token) {
	s := scanPlain(t.name, t.input)
	for {
		token := s.nextToken()
		tokens = append(tokens, token)
		if token.typeof == tokenEOF || token.typeof == tokenError {
			break
		}
	}
	return
}

func newParSeq(typ, text string) (tkns []token) {
	tkns = []token{
		tkn(tokenCmdStart, ""),
		tkn(tokenName, "sys.paragraph."+typ),
		tkn(tokenLeftSquare, "["),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, text),
		tkn(tokenRightCurly, "}"),
		tkn(tokenRightSquare, "]"),
	}
	return
}

func emptyCmdSeq(name, mode string) (tkns []token) {
	tkns = []token{
		tkn(tokenCmdStart, mode),
		tkn(tokenName, name),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenRightCurly, "}"),
	}
	return
}

func newCmdSeq(name, mode, val string) (tkns []token) {
	tkns = []token{
		tkn(tokenCmdStart, mode),
		tkn(tokenName, name),
		tkn(tokenLeftCurly, "{"),
		tkn(tokenText, val),
		tkn(tokenRightCurly, "}"),
	}
	return
}

func newCmdSeqPre(name, mode string) (tkns []token) {
	tkns = []token{
		tkn(tokenCmdStart, mode),
		tkn(tokenName, name),
		tkn(tokenLeftCurly, "{"),
	}
	return
}

func newCmdSeqPost() (tkns []token) {
	tkns = []token{
		tkn(tokenRightCurly, "}"),
	}
	return
}

type myTestCases struct {
	tokens []token
}

func NewTC() *myTestCases {
	return &myTestCases{[]token{}}
}
func (m *myTestCases) add(t []token) *myTestCases {
	m.tokens = append(m.tokens, t...)
	return m
}
func tkns(ttype tokenType, val string) []token {
	return []token{tkn(ttype, val)}
}

// Test scan -------------------------------------------------------------

var parGenTestCases = []testParCase{
	newParCase("just text", "1", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("middle paragraph", "1\n\n2", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("multiple paragraphs", "1\n\n2\n\n3\n\n4\n", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "4"))
		t.add(newParSeq("end", "\n"))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("initial paragraph and whitespace", "\n\n \n \n1", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("busy middle paragraphs", "1\n\n \n\n\n2", "", "\n\n \n\n\n", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n \n\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("simple terminal paragraph", "1\n\n2\n\n3\n", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", "\n"))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("terminal paragraph", "1\n\n \n\n\n", "", "\n\n \n\n\n", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n \n\n\n"))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("middle paragraphs", "\n \n1\n\n \n2\n \n ", "\n \n", "\n \n ", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n \n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", "\n \n "))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("turn off par scanning", "1\n\n¶-\n\n2\n\n¶+3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(tkns(tokenText, "\n\n2\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("vertical mode command", "1\n\n§a{xyz}\n\n3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newCmdSeq("a", "V", "xyz"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("vertical mode command with trailing text", "1\n\n§a{xyz}abc\n\n3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newCmdSeq("a", "V", "xyz"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "abc"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("initial space and command", "•a{}12", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(emptyCmdSeq("a", "H"))
		t.add(tkns(tokenText, "12"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("vertical mode cmd with 2 args", "§a[{xyz}{a\n\nX}] •abc\n\n3", "", "", func() []token {
		t := NewTC()
		// t.add(newParSeq("begin", ""))
		// t.add(newParSeq("end", ""))
		t.add(tkns(tokenCmdStart, "V"))
		t.add(tkns(tokenName, "a"))
		t.add(tkns(tokenLeftSquare, "["))
		t.add(tkns(tokenLeftCurly, "{"))
		t.add(tkns(tokenText, "xyz"))
		t.add(tkns(tokenRightCurly, "}"))
		t.add(tkns(tokenLeftCurly, "{"))
		t.add(tkns(tokenText, "a\n\nX"))
		t.add(tkns(tokenRightCurly, "}"))
		t.add(tkns(tokenRightSquare, "]"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenCmdStart, "H"))
		t.add(tkns(tokenName, "abc"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("simple command arg", "1\n\n•2{A}3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(newCmdSeq("2", "H", "A"))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("line breaks in command arg", "•2{A\n\nB}3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(newCmdSeq("2", "H", "A\n\nB"))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("vertical mode inside a horizontal", "•2{a§X{Y}b}3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(newCmdSeqPre("2", "H"))
		t.add(tkns(tokenText, "a"))
		// t.add(newParSeq("end", ""))
		t.add(newCmdSeq("X", "V", "Y"))
		t.add(tkns(tokenText, "b"))
		t.add(tkns(tokenRightCurly, "}"))
		// t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("horizontal mode inside a vertical command", "§1{a•X{Y}b}3", "", "", func() []token {
		t := NewTC()
		// t.add(newParSeq("begin", ""))
		// t.add(newParSeq("end", ""))
		t.add(newCmdSeqPre("1", "V"))
		t.add(tkns(tokenText, "a"))
		t.add(newCmdSeq("X", "H", "Y"))
		t.add(tkns(tokenText, "b"))
		t.add(tkns(tokenRightCurly, "}"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("pars in command arg", "1\n\n•2{¶+A\n\nB}3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(newCmdSeqPre("2", "H"))
		t.add(tkns(tokenText, "A"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "B"))
		t.add(newCmdSeqPost())
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("consecutive command with pars in arg", "1\n\n•2{¶+A\n\nB} •b{C\n\nD}3", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(newCmdSeqPre("2", "H"))
		t.add(tkns(tokenText, "A"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "B"))
		t.add(newCmdSeqPost())
		t.add(tkns(tokenText, " "))
		t.add(newCmdSeq("b", "H", "C\n\nD"))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("multiple double linebreaks", "1\n\n\n\n\n\n2", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n\n\n\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("final paragraph", "1\n\n\n\n\n\n", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", "\n\n\n\n\n\n"))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("initial paragraph", "\n\n  \n\n1", "", "", func() []token {
		t := NewTC()
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "1"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("parScanFlag off at beginning", "¶-1\n\n2\n\n3\n\n4\n", "", "", func() []token {
		t := NewTC()
		t.add(tkns(tokenText, "1\n\n2\n\n3\n\n4\n"))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("parScanFlag off at beginning with command", "¶-1\n\n•a{2}\n\n", "", "", func() []token {
		t := NewTC()
		t.add(tkns(tokenText, "1\n\n"))
		t.add(newCmdSeq("a", "H", "2"))
		t.add(tkns(tokenText, "\n\n"))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	newParCase("two calls to parScanFlag", "¶-1\n\n¶+2\n\n3", "", "", func() []token {
		t := NewTC()
		t.add(tkns(tokenText, "1\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
	// newParCase("use ¶+ to supress paragraph break", "¶-1\n\n¶+2\n¶+\n3", "", "", func() []token {
	// 	t := NewTC()
	// 	t.add(tkns(tokenText, "1\n\n"))
	// 	t.add(newParSeq("begin", ""))
	// 	t.add(tkns(tokenText, "2\n"))
	// 	t.add(tkns(tokenText, "\n3"))
	// 	t.add(newParSeq("end", ""))
	// 	t.add(tkns(tokenEOF, ""))
	// 	return t.tokens
	// }()),
	newParCase("consecutive ¶+", "¶-1\n\n¶+¶+¶+2\n\n3", "", "", func() []token {
		t := NewTC()
		t.add(tkns(tokenText, "1\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "2"))
		t.add(newParSeq("end", "\n\n"))
		t.add(newParSeq("begin", ""))
		t.add(tkns(tokenText, "3"))
		t.add(newParSeq("end", ""))
		t.add(tkns(tokenEOF, ""))
		return t.tokens
	}()),
}

type testParCase struct {
	name    string  // The name of the test.
	input   string  // The input to be tested.
	expInit string  // The expected initial white space.
	expTerm string  // The expected terminal white space.
	exp     []token // The expected output.
	loc     bool    // Indicates if we should test loc related data.
}

func newParCase(name, input, init, term string, exp []token) testParCase {
	return testParCase{name, input, init, term, exp, false}
}

func TestScannerEmpty(t *testing.T) {
	var tokens []token
	scnr := scan("empty", "\n\n  ")
	for {
		token := scnr.nextToken()
		tokens = append(tokens, token)
		if token.typeof == tokenEOF || token.typeof == tokenError {
			break
		}
	}
	switch {
	case tokens == nil:
		t.Errorf("No tokens in TestScannerEmpty")
	case len(tokens) != 1:
		t.Errorf("Wrong number of args in TestScannerEmpty. Got: %d", len(tokens))
	// case tokens[0].typeof != tokenText:
	// 	t.Errorf("Wrong token type in TestScannerEmpty")
	case tokens[0].typeof != tokenEOF:
		t.Errorf("Wrong token type in TestScannerEmpty %s", tokenTypeLookup(tokens[1].typeof))
	}
}

// Paragraph generation is on for these tests. Verification of the standard
// initial paragraph command sequence is built into the test code so we don't
// need to specify that sequence in each test.

func TestScanner(t *testing.T) {
	e := scanTest(t, parGenTestCases)
	if e {
		printTokenCodes()
	}
}

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

func getOuterParSeq(parType, text string) []token {
	tkns := []token{
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

func verifyOuterPar(t *testing.T, ts []token, parType, space, name string) bool {
	exp := []token{
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
func runTest(t *testParCase) (tokens []token) {
	s := scan(t.name, t.input)
	for {
		token := s.nextToken()
		tokens = append(tokens, token)
		if token.typeof == tokenEOF || token.typeof == tokenError {
			break
		}
	}
	return
}

// Utils ----------------------------------------------------------------------

func equal(t1, t2 []token, checkLoc bool) bool {
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
