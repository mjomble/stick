package parse

import (
	"testing"
)

type parseTest struct {
	name     string
	input    string
	expected *ModuleNode
	err      string
}

const noError = ""

// Position testing isnt implemented
var noPos = pos{0, 0}
var nilBody = func() *BodyNode { return nil }()

func newParseTest(name, input string, expected *ModuleNode) parseTest {
	return parseTest{name, input, expected, noError}
}

func newErrorTest(name, input string, err string) parseTest {
	return parseTest{name, input, mkModule(), err}
}

func mkModule(nodes ...Node) *ModuleNode {
	l := newModuleNode()
	for _, n := range nodes {
		l.append(n)
	}

	return l
}

var parseTests = []parseTest{
	// Errors
	newErrorTest("unclosed block", "{% block test %}", "parse error: unclosed tag \"block\" starting on line 1, offset 3"),
	newErrorTest("unclosed if", "{% if test %}", "parse error: unclosed tag \"if\" starting on line 1, offset 3"),
	newErrorTest("unexpected end (function call)", "{{ func('arg1'", "parse error: unexpected end of input on line 1, offset 14"),
	newErrorTest("unclosed parenthesis", "{{ func(arg1 }}", "parse error: expected one of [PUNCTUATION, PARENS_CLOSE], got \"PRINT_CLOSE\" on line 1, offset 13"),
	newErrorTest("unexpected punctuation", "{{ func(arg1? arg2) }}", "parse error: unexpected \"?\", expected \",\" on line 1, offset 12"),

	// Valid
	newParseTest("text", "some text", mkModule(newTextNode("some text", noPos))),
	newParseTest("hello", "Hello {{ name }}", mkModule(newTextNode("Hello ", noPos), newPrintNode(newNameExpr("name", noPos), noPos))),
	newParseTest("string expr", "Hello {{ 'Tyler' }}", mkModule(newTextNode("Hello ", noPos), newPrintNode(newStringExpr("Tyler", noPos), noPos))),
	newParseTest(
		"simple tag",
		"{% block something %}Body{% endblock %}",
		mkModule(newBlockNode("something", newBodyNode(noPos, newTextNode("Body", noPos)), noPos)),
	),
	newParseTest(
		"if",
		"{% if something %}Do Something{% endif %}",
		mkModule(newIfNode(newNameExpr("something", noPos), newBodyNode(noPos, newTextNode("Do Something", noPos)), nilBody, noPos)),
	),
	newParseTest(
		"if else",
		"{% if something %}Do Something{% else %}Another thing{% endif %}",
		mkModule(newIfNode(newNameExpr("something", noPos), newBodyNode(noPos, newTextNode("Do Something", noPos)), newBodyNode(noPos, newTextNode("Another thing", noPos)), noPos)),
	),
	newParseTest(
		"if else if",
		"{% if something %}Do Something{% else if another %}Another thing{% endif %}",
		mkModule(newIfNode(
			newNameExpr("something", noPos),
			newBodyNode(noPos, newTextNode("Do Something", noPos)),
			newBodyNode(noPos, newIfNode(
				newNameExpr("another", noPos),
				newBodyNode(noPos, newTextNode("Another thing", noPos)),
				nilBody,
				noPos)),
			noPos)),
	),
	newParseTest(
		"function expr",
		"{{ func('arg1', arg2) }}",
		mkModule(newPrintNode(newFuncExpr(newNameExpr("func", noPos), []Expr{newStringExpr("arg1", noPos), newNameExpr("arg2", noPos)}, noPos), noPos)),
	),
	newParseTest(
		"extends statement",
		"{% extends '::base.html.twig' %}",
		mkModule(newExtendsNode(newStringExpr("::base.html.twig", noPos), noPos)),
	),
	newParseTest(
		"basic binary operation",
		"{{ something + else }}",
		mkModule(newPrintNode(newBinaryExpr(newNameExpr("something", noPos), OpBinaryAdd, newNameExpr("else", noPos), noPos), noPos)),
	),
	newParseTest(
		"number literal binary operation",
		"{{ 4.123 + else - test }}",
		mkModule(newPrintNode(newBinaryExpr(newBinaryExpr(newNumberExpr("4.123", noPos), OpBinaryAdd, newNameExpr("else", noPos), noPos), OpBinarySubtract, newNameExpr("test", noPos), noPos), noPos)),
	),
	newParseTest(
		"parenthesis grouping expression",
		"{{ (4 + else) / 10 }}",
		mkModule(newPrintNode(newBinaryExpr(newGroupExpr(newBinaryExpr(newNumberExpr("4", noPos), OpBinaryAdd, newNameExpr("else", noPos), noPos), noPos), OpBinaryDivide, newNumberExpr("10", noPos), noPos), noPos)),
	),
	newParseTest(
		"correct order of operations",
		"{{ 10 + 5 / 5 }}",
		mkModule(newPrintNode(newBinaryExpr(newNumberExpr("10", noPos), OpBinaryAdd, newBinaryExpr(newNumberExpr("5", noPos), OpBinaryDivide, newNumberExpr("5", noPos), noPos), noPos), noPos)),
	),
	newParseTest(
		"correct ** associativity",
		"{{ 10 ** 2 ** 5 }}",
		mkModule(newPrintNode(newBinaryExpr(newNumberExpr("10", noPos), OpBinaryPower, newBinaryExpr(newNumberExpr("2", noPos), OpBinaryPower, newNumberExpr("5", noPos), noPos), noPos), noPos)),
	),
	newParseTest(
		"extended binary expression",
		"{{ 5 + 10 + 15 * 12 / 4 }}",
		mkModule(newPrintNode(newBinaryExpr(newBinaryExpr(newNumberExpr("5", noPos), OpBinaryAdd, newNumberExpr("10", noPos), noPos), OpBinaryAdd, newBinaryExpr(newBinaryExpr(newNumberExpr("15", noPos), OpBinaryMultiply, newNumberExpr("12", noPos), noPos), OpBinaryDivide, newNumberExpr("4", noPos), noPos), noPos), noPos)),
	),
	newParseTest(
		"unary not expression",
		"{{ not something }}",
		mkModule(newPrintNode(newUnaryExpr(OpUnaryNot, newNameExpr("something", noPos), noPos), noPos)),
	),
	newParseTest(
		"dot notation accessor",
		"{{ something.another.else }}",
		mkModule(newPrintNode(newGetAttrExpr(newGetAttrExpr(newNameExpr("something", noPos), newNameExpr("another", noPos), noPos), newNameExpr("else", noPos), noPos), noPos)),
	),
	newParseTest(
		"dot notation array access mix",
		"{{ something().another['test'].further }}",
		mkModule(newPrintNode(newGetAttrExpr(newGetAttrExpr(newGetAttrExpr(newFuncExpr(newNameExpr("something", noPos), []Expr{}, noPos), newNameExpr("another", noPos), noPos), newStringExpr("test", noPos), noPos), newNameExpr("further", noPos), noPos), noPos)),
	),
	newParseTest(
		"basic filter",
		"{{ something|default('another') }}",
		mkModule(newPrintNode(newFuncExpr(newNameExpr("default", noPos), []Expr{newNameExpr("something", noPos), newStringExpr("another", noPos)}, noPos), noPos)),
	),
	newParseTest(
		"filter with no args",
		"{{ something|default }}",
		mkModule(newPrintNode(newFuncExpr(newNameExpr("default", noPos), []Expr{newNameExpr("something", noPos)}, noPos), noPos)),
	),
}

func nodeEqual(a, b Node) bool {
	if a.String() != b.String() {
		return false
	}

	return true
}

func evaluateTest(t *testing.T, test parseTest) {
	tree, err := Parse(test.input)
	if test.err != noError && err != nil && test.err != err.Error() {
		t.Errorf("%s: got error\n\t%+v\nexpected error\n\t%v", test.name, err, test.err)
	} else if !nodeEqual(tree.root, test.expected) {
		t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, tree.root, test.expected)
		if err != nil {
			t.Errorf("%s: got error\n\t%v", test.name, err.Error())
		}
	}
}

func TestParse(t *testing.T) {
	for _, test := range parseTests {
		evaluateTest(t, test)
	}
}
