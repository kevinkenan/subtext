package parse

import (
	"fmt"
	// "strings"
	// "sort"
	"testing"
)

// Test SelectArguments -------------------------------------------------------
// The initial opening paragraph command is skipped in these tests.

type selectArgumentsTestCase struct {
	name        string
	command     string
	loc         int // The index of the command in root.NodeList
	reqParams   []string
	optParams   []string
	expSelected []string
	expUnknown  []string
	expMissing  []string
}

var selectArgumentsTestCases = []selectArgumentsTestCase{
	{"no args", "•test[]", 0, []string{"one"}, nil,
		nil, nil, []string{"one"}},
	{"no args and no parameters", "•test[]", 0, nil, nil,
		nil, nil, nil},
	{"empty bare arg", "•test{}", 0, []string{"one"}, nil,
		[]string{"one"}, nil, nil},
	{"empty anonymous arg", "•test[{}]", 0, []string{"one"}, nil,
		[]string{"one"}, nil, nil},
	{"anonymous arg with missing args", "•test{a}", 0, []string{"one", "two", "three"}, nil,
		nil, nil, []string{"two", "three"}},
	{"anonymous args with one unknown", "•test[{a}{b}]", 0, []string{"one"}, nil,
		nil, []string{"#2"}, nil},
	{"anonymous with only optional args", "•test[{abc}{xyz}]", 0, nil, []string{"one", "two"},
		[]string{"one", "two"}, nil, nil},
	{"anonymous with an optional arg", "•test[{a}{b}]", 0, []string{"one"}, []string{"two"},
		[]string{"one", "two"}, nil, nil},
	{"anonymous with optional and unknown args", "•test[{a}{b}{c}]", 0, []string{"one"}, []string{"two"},
		nil, []string{"#3"}, nil},
	{"named args", "•test[one={a}]", 0, []string{"one"}, nil,
		[]string{"one"}, nil, nil},
	{"named and optional args", "•test[one={a} two={b}]", 0, []string{"one"}, []string{"two"},
		[]string{"one", "two"}, nil, nil},
	{"named with unknown args", "•test[one={a} two={b}]", 0, []string{"one"}, nil,
		nil, []string{"two"}, nil},
	{"named with missing args", "•test[one={a}]", 0, []string{"one", "three"}, nil,
		nil, nil, []string{"three"}},
	{"named with unknown and optional args", "•test[one={a} two={b}]", 0, []string{"one"}, []string{"three"},
		nil, []string{"two"}, nil},
	{"named with unknown and missing args", "•test[three={a} one={b}]", 0, nil, []string{"one", "three"},
		[]string{"one", "three"}, nil, nil},
	{"named with no parameters at all", "•test[one={a} two={b}]", 0, nil, nil,
		nil, []string{"one", "two"}, nil},
}

func testSelectArguments(t *testing.T, test *selectArgumentsTestCase) {
	root, _ := ParsePlain(test.name, test.command)
	if len(root.NodeList) < test.loc-1 {
		t.Errorf("%s: loc (%d) is beyond the end of root.NodeList", test.name, test.loc)
		return
	}
	cmdNode := root.NodeList[test.loc]
	_, ok := cmdNode.(*Cmd)
	if !ok {
		t.Errorf("%s: node at loc %d is not a Cmd node", test.name, test.loc)
		return
	}
	selected, unknown, missing := cmdNode.(*Cmd).SelectArguments(test.reqParams, test.optParams)
	// fmt.Printf("%s: s: %v ; u: %v ; m: %v\n", test.name, selected, unknown, missing)
	if !checkNodeMapKeys(selected, test.expSelected) || !checkStringSlices(unknown, test.expUnknown) || !checkStringSlices(missing, test.expMissing) {
		t.Errorf("%s\n  *result: %v, %v, %v\n*expected: %v, %v, %v", test.name,
			names(selected), unknown, missing,
			test.expSelected, test.expUnknown, test.expMissing)
		// t.Errorf("%s\n  wrong arguments selected\n  *expected: %v\n       *got: %v", test.name, test.expSelected, names(selected))
	}
}

func TestSelectArguments(t *testing.T) {
	for _, test := range selectArgumentsTestCases {
		testSelectArguments(t, &test)
	}
}

func names(nodes NodeMap) (names []string) {
	names = []string{}
	for k := range nodes {
		names = append(names, k)
	}
	return
}

// checkNodeMapKeys returns true if all the keys specified in the []string
// argument are in the given NodeMap.
func checkNodeMapKeys(nodes NodeMap, keys []string) bool {
	if len(nodes) != len(keys) {
		return false
	}
	for _, k := range keys {
		_, ok := nodes[k]
		// fmt.Println("here: " + k)
		if !ok {
			return false
		}
	}
	return true
}

func checkStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if isSubset(a, b) {
		// We know len(a) == len(b), so subset implies equality.
		return true
	}
	return false
}

// isSubset returns true if all members of a are in b.
func isSubset(a, b []string) bool {
	for _, av := range a {
		found := false
		for _, bv := range b {
			if av == bv {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Test Parsing ---------------------------------------------------------------
// The initial opening paragraph command is skipped in these tests.

type parseTestCase struct {
	name         string
	input        string
	expOut       string
	expNodeCount int // number of expected nodes.
	expErr       bool
	verbose      bool
}

var parseTestCases = []parseTestCase{
	{"empty", "", "", 1, false, false},
	{"basic", "test", "test", 2, false, false},
	{"contains three literals", "test ``, `• and `◊.", "test `, • and ◊.", 5, false, false},
	{"basic with linebreak", "test\nline two", "test\nline two", 2, false, false},
	{"bare command", "1 •xyz[] 3", "1 •xyz[<>{}] 3", 4, false, false},
	{"simple empty command", "1 •2{} 3", "1 •2[<>{}] 3", 4, false, false},
	{"two anonymous command args", "1 •2[{a}{b}] 3", "1 •2[<>{a}{b}] 3", 6, false, false},
	{"simple command", "1 •2{3}4", "1 •2[<>{3}]4", 5, false, false},
	{"flags", "1 •2[<34 \n5=6>]7", "1 •2[<34,5=6>{}]7", 4, false, false},
	{"anonymous command on different lines", "1 •2[{a}\n{•b{c}}] 4", "1 •2[<>{a}{•b[<>{c}]}] 4", 7, false, false},
	{"anonymous command with line breaks", "1 •2[{a\nb} {c\nd}] 4", "1 •2[<>{a\nb}{c\nd}] 4", 6, false, false},
	{"two named args on different lines", "1 •2[x={a}\ny={b}]3", "1 •2[<>x={a}y={b}]3", 6, false, false},
	{"complex context with named args", "1 •2[1={a}2={b •x{c}}] 4", "1 •2[<>1={a}2={b •x[<>{c}]}] 4", 8, false, false},
	{"line breaks", "\n\n1\n\n2\n\n3\n", "\n\n1\n\n2\n\n3\n", 3, false, false},
	{"line breaks with parscan flag", "¶+\n\n1\n\n2\n\n3\n", "\n\n1\n\n2\n\n3\n", 3, false, false},

	// SysCmd tests
	{"SysCmd", "test•(this)now", "test•(this)now", 4, false, false},
	{"SysCmd advanced", "test•(this=that what)now", "test•(this=that)•(what)now", 5, false, false},
	{"SysCmd with linebreak", "test•(this\nthat)now", "test•(this)•(that)now", 5, false, false},

	// Error tests
	{"basic kv command", "1•", "", 0, true, false},
}

var parParseTestCases = []parseTestCase{
	// Paragaph tests
	{"basic paragraph", "one\n\ntwo",
		"•sys.paragraph.begin[<>{}]" +
			"one•sys.paragraph.end[<>{\n\n}]" +
			"•sys.paragraph.begin[<>{}]" +
			"two" +
			"•sys.paragraph.end[<>{}]",
		11, false, false},
	{"simple command with paragraph", "1•2{A\n\nB}3",
		"•sys.paragraph.begin[<>{}]1•2[<>{A\n\nB}]3•sys.paragraph.end[<>{}]",
		9, false, false},
	{"context command with paragraph", "1•2[<x>{A\n \nB}{\n\nC}]3",
		"•sys.paragraph.begin[<>{}]1•2[<x>{A\n \nB}{\n\nC}]3•sys.paragraph.end[<>{}]",
		10, false, false},
	{"line breaks with parscan flag", "\n\n1¶-\n\n¶+2\n\n3\n",
		"•sys.paragraph.begin[<>{}]1\n\n2•sys.paragraph.end[<>{\n\n}]•sys.paragraph.begin[<>{}]3•sys.paragraph.end[<>{\n}]",
		13, false, false},
	{"line breaks with parscan flag off", "¶-\n\n1\n\n2\n\n3\n", "\n\n1\n\n2\n\n3\n", 3, false, false},
	{"vertical mode test", "\n\n§a{b}\n\ncde\n",
		"•sys.paragraph.begin[<>{}]•sys.paragraph.end[<>{}]•a[<>{b}]•sys.paragraph.begin[<>{}]cde•sys.paragraph.end[<>{\n}]", 12, false, false},
}

func TestParse(t *testing.T) {
	tnum := -1
	var start, end = 0, len(parParseTestCases)
	if tnum > 0 {
		start = tnum
		end = tnum + 1
	}
	testParse(t, parParseTestCases, false, start, end)
}

func TestParsePlain(t *testing.T) {
	tnum := -1
	var start, end = 0, len(parseTestCases)
	if tnum > 0 {
		start = tnum
		end = tnum + 1
	}
	testParse(t, parseTestCases, true, start, end)
}

func testParse(t *testing.T, tests []parseTestCase, plain bool, start, end int) {
	for tc, test := range tests[start:end] {
		var (
			result *Section
			err    error
		)
		if plain {
			result, err = ParsePlain(test.name, test.input)
		} else {
			result, err = Parse(test.name, test.input)
		}
		if test.verbose {
			if err == nil {
				fmt.Printf("Verbose: %v\n", test.name)
				v(result)
			} else {
				fmt.Printf("%s: '%q' -> error: '%q'\n", test.name, test.input, err)
			}
		}
		if test.expErr && err == nil {
			t.Errorf("%s\n  *result: unexpected success", test.name)
		}
		if !test.expErr && err != nil {
			t.Errorf("%s\n  *result: unexpected error: %s", test.name, err)
		}
		if err == nil {
			nodeCount := result.Count()
			s := result.NodeList.String()
			if s != test.expOut || nodeCount != test.expNodeCount {
				t.Errorf("[%d] %s\n  *result:   %q (%d nodes)\n  *expected: %q (%d nodes)\n", tc, test.name, s, nodeCount, test.expOut, test.expNodeCount)
			}
		}
	}
}

func v(root *Section) {
	c := make(chan Node)
	go root.Walk(c)
	fmt.Printf(":: %s\n", root.String())
	fmt.Printf("> Root Section Node: contains %d nodes\n", len(root.NodeList))
	for n := range c {
		switch n.(type) {
		case *Text:
			fmt.Printf("> Text Node: %q\n", n.(*Text).NodeValue)
		case *Section:
			fmt.Printf("> Section Node: contains %d nodes\n", len(n.(*Section).NodeList))
		case *ErrorNode:
			fmt.Printf("> Error: %q\n", n.(*ErrorNode).NodeValue)
		case *Cmd:
			fmt.Printf("> Cmd Node: %q\n", n.(*Cmd).NodeValue)
			fmt.Printf("     Count: %d nodes\n", n.(*Cmd).Count())
			fmt.Print("     Flags: <")
			for _, f := range n.(*Cmd).Flags {
				fmt.Printf("%s", f)
			}
			fmt.Println(">")
			fmt.Printf("     Anonymous: %t\n", n.(*Cmd).Anonymous)
			if n.(*Cmd).Anonymous {
				for i, nl := range n.(*Cmd).ArgList {
					fmt.Printf("     Text Block %d:\n", i)
					for _, nn := range nl {
						fmt.Printf("       %q\n", nn)
					}
				}
			} else {
				if len(n.(*Cmd).ArgMap) > 0 {
					for k, v := range n.(*Cmd).ArgMap {
						fmt.Printf("     Argument %q: %s\n", k, v)
					}
				} else {
					fmt.Println("     Arguments: None")
				}
			}
		default:
			fmt.Printf("> UNEXPECTED Node: %q\n", n.String())
			fmt.Printf("     Type Code: %d\n", n.Typeof())
		}
	}
}
