package parser

import (
	"bytes"
	"testing"

	"github.com/lyraproj/issue/issue"
)

func TestEmpty(t *testing.T) {
	expectBlock(t, ``, `(block)`)
}

func TestInvalidUnicode(t *testing.T) {
	expectError(t, "$var = \"\xa0\xa1\"", `invalid unicode character at offset 8`)
	expectError(t, "$var = 23\xa0\xa1", `invalid unicode character at offset 9`)
}

func TestInteger(t *testing.T) {
	expectDump(t, `0`, `0`)
	expectDump(t, `123`, `123`)
	expectDump(t, `+123`, `123`)
	expectDump(t, `0XABC`, `(int {:radix 16 :value 2748})`)
	expectDump(t, `0772`, `(int {:radix 8 :value 506})`)
	expectError(t, `3g`, `digit expected (line: 1, column: 2)`)
	expectError(t, `3ö`, `digit expected (line: 1, column: 2)`)
	expectError(t, `0x3g21`, `hexadecimal digit expected (line: 1, column: 4)`)
	expectError(t, `078`, `octal digit expected (line: 1, column: 3)`)
}

func TestNegativeInteger(t *testing.T) {
	expectDump(t, `-123`, `-123`)
}

func TestFloat(t *testing.T) {
	expectDump(t, `0.123`, `0.123`)
	expectDump(t, `123.32`, `123.32`)
	expectDump(t, `+123.32`, `123.32`)
	expectDump(t, `-123.32`, `-123.32`)
	expectDump(t, `12e12`, `1.2e+13`)
	expectDump(t, `12e-12`, `1.2e-11`)
	expectDump(t, `12.23e12`, `1.223e+13`)
	expectDump(t, `12.23e-12`, `1.223e-11`)

	expectError(t, `123.a`, `digit expected (line: 1, column: 5)`)
	expectError(t, `123.4a`, `digit expected (line: 1, column: 6)`)

	expectError(t, `123.45ex`, `digit expected (line: 1, column: 8)`)
	expectError(t, `123.45e3x`, `digit expected (line: 1, column: 9)`)
}

func TestBoolean(t *testing.T) {
	expectDump(t, `false`, `false`)
	expectDump(t, `true`, `true`)
}

func TestDefault(t *testing.T) {
	expectDump(t, `default`, `(default)`)
}

func TestUndef(t *testing.T) {
	expectDump(t, `undef`, `nil`)
}

func TestSingleQuoted(t *testing.T) {
	expectDump(t, `'undef'`, `"undef"`)
	expectDump(t, `'escaped single \''`, `"escaped single '"`)
	expectDump(t, `'unknown escape \k'`, `"unknown escape \\k"`)
}

func TestDoubleQuoted(t *testing.T) {
	expectDump(t,
		`"string\nwith\t\\t,\s\\s, \\r, and \\n\r\n"`,
		`"string\nwith\t\\t, \\s, \\r, and \\n\r\n"`)

	expectDump(t,
		`"unknown \k escape"`,
		`"unknown \\k escape"`)

	expectDump(t,
		`"control \u{14}"`,
		`"control \o024"`)

	expectDump(t,
		`"$var"`,
		`(concat (str (var "var")))`)

	expectDump(t,
		`"hello $var"`,
		`(concat "hello " (str (var "var")))`)

	expectDump(t,
		`"hello ${var}"`,
		`(concat "hello " (str (var "var")))`)

	expectDump(t,
		`"hello ${}"`,
		`(concat "hello " (str nil))`)

	expectDump(t,
		`"Before ${{ a => true, b => "hello"}} and after"`,
		`(concat "Before " (str (hash (=> (qn "a") true) (=> (qn "b") "hello"))) " and after")`)

	expectDump(t, `"x\u{1f452}y"`, `"x👒y"`)

	expectError(t,
		`"$Var"`,
		`malformed interpolation expression (line: 1, column: 2)`)

	expectError(t,
		issue.Unindent(`
      $x = "y
      notice($x)`),
		"unterminated double quoted string (line: 1, column: 6)")

	expectError(t,
		issue.Unindent(`
      $x = "y${var"`),
		"unterminated double quoted string (line: 1, column: 13)")

	expectDump(t, `"x\u2713y"`, `"x✓y"`)
}

func TestRegexp(t *testing.T) {
	expectDump(t,
		`$a = /.*/`,
		`(= (var "a") (regexp ".*"))`)

	expectDump(t, `/pattern\/with\/slash/`, `(regexp "pattern/with/slash")`)
	expectDump(t, `/pattern\/with\\\/slash/`, `(regexp "pattern/with\\\\/slash")`)
	expectDump(t, `/escaped \t/`, `(regexp "escaped \\t")`)

	expectDump(t,
		issue.Unindent(`
      /escaped #rx comment
      continues
      .*/`),
		`(regexp "escaped #rx comment\ncontinues\n.*")`)

	expectError(t,
		`$a = /.*`,
		`unexpected token '/' (line: 1, column: 6)`)
}

func TestReserved(t *testing.T) {
	expectDump(t,
		`$a = attr`,
		`(= (var "a") (reserved "attr"))`)

	expectDump(t,
		`$a = private`,
		`(= (var "a") (reserved "private"))`)
}

func TestHeredoc(t *testing.T) {
	expectHeredoc(t, issue.Unindent(`
      @(END)
      END`),
		"")

	expectHeredoc(t, issue.Unindent(`
      @(END)
      This is
      heredoc text
      END`),
		"This is\nheredoc text\n")

	expectError(t, issue.Unindent(`
      @(END)
      This is
      heredoc text`),
		"unterminated heredoc (line: 1, column: 1)")

	expectDump(t,
		issue.Unindent(`
      { a => @(ONE), b => @(TWO) }
      The first
      heredoc text
      -ONE
      The second
      heredoc text
      -TWO`),
		`(hash (=> (qn "a") (heredoc {:text "The first\nheredoc text"})) (=> (qn "b") (heredoc {:text "The second\nheredoc text"})))`)

	expectDump(t,
		issue.Unindent(`
      ['first', @(SECOND), 'third', @(FOURTH), 'fifth',
        This is the text of the
        second entry
        |-SECOND
        And here is the text of the
        fourth entry
        |-FOURTH
        'sixth']`),
		`(array "first" (heredoc {:text "This is the text of the\nsecond entry"}) "third" (heredoc {:text "And here is the text of the\nfourth entry"}) "fifth" "sixth")`)

	expectError(t,
		issue.Unindent(`
      @(END
      /t)
      This\nis\nheredoc\ntext
      -END`),
		`unterminated @( (line: 1, column: 1)`)

	expectError(t,
		issue.Unindent(`
      @(END)
      This\nis\nheredoc\ntext

      `),
		`unterminated heredoc (line: 1, column: 1)`)

	expectError(t,
		issue.Unindent(`
      @(END)`),
		`unterminated heredoc (line: 1, column: 1)`)
}

func TestHeredocSyntax(t *testing.T) {
	expectDump(t, issue.Unindent(`
      @(END:syntax)
      This is
      heredoc text
      END`),
		`(heredoc {:syntax "syntax" :text "This is\nheredoc text\n"})`)

	expectError(t, issue.Unindent(`
      @(END:json:yaml)
      This is
      heredoc text`),
		`more than one syntax declaration in heredoc (line: 1, column: 11)`)
}

func TestHeredocFlags(t *testing.T) {
	expectHeredoc(t,
		issue.Unindent(`
      @(END/t)
      This\tis\t
      heredoc text
      -END`),
		"This\tis\t\nheredoc text")

	expectHeredoc(t,
		issue.Unindent(`
      @(END/s)
      This\sis\sheredoc\stext
      -END`),
		`This is heredoc text`)

	expectHeredoc(t,
		issue.Unindent(`
      @(END/r)
      This\ris\rheredoc\rtext
      -END`),
		"This\ris\rheredoc\rtext")

	expectHeredoc(t,
		issue.Unindent(`
      @(END/n)
      This\nis\nheredoc\ntext
      -END`),
		"This\nis\nheredoc\ntext")

	expectHeredoc(t,
		issue.Unindent(`
      @(END:syntax/n)
      This\nis\nheredoc\ntext
      -END`),
		"This\nis\nheredoc\ntext", `syntax`)

	expectError(t,
		issue.Unindent(`
      @(END/k)
      This\nis\nheredoc\ntext
      -END`),
		`illegal heredoc escape 'k' (line: 1, column: 7)`)

	expectError(t,
		issue.Unindent(`
      @(END/t/s)
      This\nis\nheredoc\ntext
      -END`),
		`more than one declaration of escape flags in heredoc (line: 1, column: 8)`)
}

func TestHeredocStripNL(t *testing.T) {
	expectHeredoc(t,
		"@(END)\r\nThis is\r\nheredoc text\r\n-END",
		"This is\r\nheredoc text")
}

func TestHeredocMargin(t *testing.T) {
	expectHeredoc(t,
		issue.Unindent(`
      @(END/t)
        This\tis
        heredoc text
        | END
      `),
		"This\tis\nheredoc text\n")

	expectHeredoc(t,
		issue.Unindent(`
      @(END)
        | END
      `),
		"")

	// Lines that have less margin than what's stripped are not stripped
	expectHeredoc(t,
		issue.Unindent(`
      @(END/t)
        This\tis
       heredoc text
        | END
      `),
		"This\tis\n heredoc text\n")
}

func TestHeredocMarginAndNewlineTrim(t *testing.T) {
	expectHeredoc(t,
		issue.Unindent(`
      @(END/t)
        This\tis
        heredoc text
        |- END`),
		"This\tis\nheredoc text")

	expectHeredoc(t,
		issue.Unindent(`
      @(END)
        |-END
      `),
		"")
}

func TestHeredocInterpolate(t *testing.T) {
	expectHeredoc(t,
		issue.Unindent(`
      @("END")
        This is
        heredoc $text
        |- END`),
		`(heredoc {:text (concat "This is\nheredoc " (str (var "text")))})`)

	expectHeredoc(t,
		issue.Unindent(`
      @("END")
        This is
        heredoc $a \$b
        |- END`),
		`(heredoc {:text (concat "This is\nheredoc " (str (var "a")) " \\" (str (var "b")))})`)

	expectHeredoc(t,
		issue.Unindent(`
      @("END"/$)
        This is
        heredoc $a \$b
        |- END`),
		`(heredoc {:text (concat "This is\nheredoc " (str (var "a")) " $b")})`)

	expectHeredoc(t,
		issue.Unindent(`
      @(END)
        This is
        heredoc $text
        |- END`),
		issue.Unindent(`
      This is
      heredoc $text`))

	expectError(t,
		issue.Unindent(`
      @("END""MORE")
        This is
        heredoc $text
        |- END`),
		`more than one tag declaration in heredoc (line: 1, column: 8)`)

	expectError(t,
		issue.Unindent(`
      @("END
      ")
        This is
        heredoc $text
        |- END`),
		`unterminated @( (line: 1, column: 1)`)

	expectError(t,
		issue.Unindent(`
      @("")
        This is
        heredoc $text
        |-`),
		`empty heredoc tag (line: 1, column: 1)`)

	expectError(t,
		issue.Unindent(`
      @()
        This is
        heredoc $text
        |-`),
		`empty heredoc tag (line: 1, column: 1)`)
}

func TestHeredocNewlineEscape(t *testing.T) {
	expectHeredoc(t,
		issue.Unindent(`
      @(END/L)
        Do not break \
        this line
        |- END`),
		issue.Unindent(`
      Do not break this line`))

	expectHeredoc(t,
		issue.Unindent(`
      @(END/L)
        Do not break \
        this line\
        |- END`),
		issue.Unindent(`
      Do not break this line\`))

	expectHeredoc(t,
		issue.Unindent(`
      @(END/t)
        Do break \
        this line
        |- END`),
		issue.Unindent(`
      Do break \
      this line`))

	expectHeredoc(t,
		issue.Unindent(`
      @(END/u)
        A checkmark \u2713 symbol
        |- END`),
		issue.Unindent(`
      A checkmark ✓ symbol`))
}

func TestHeredocUnicodeEscape(t *testing.T) {
	expectHeredoc(t,
		issue.Unindent(`
      @(END/u)
        A hat \u{1f452} symbol
        |- END`),
		issue.Unindent(`
      A hat 👒 symbol`))

	expectHeredoc(t,
		issue.Unindent(`
      @(END/u)
        A checkmark \u2713 symbol
        |- END`),
		issue.Unindent(`
      A checkmark ✓ symbol`))

	expectError(t,
		issue.Unindent(`
      @(END/u)
        A hat \u{1f452 symbol
        |- END`),
		`malformed unicode escape sequence (line: 2, column: 9)`)

	expectError(t,
		issue.Unindent(`
      @(END/u)
        A hat \u{1f45234} symbol
        |- END`),
		`malformed unicode escape sequence (line: 2, column: 9)`)

	expectError(t,
		issue.Unindent(`
      @(END/u)
        A hat \u{1} symbol
        |- END`),
		`malformed unicode escape sequence (line: 2, column: 9)`)

	expectError(t,
		issue.Unindent(`
      @(END/u)
        A checkmark \u271 symbol
        |- END`),
		`malformed unicode escape sequence (line: 2, column: 15)`)

	expectError(t,
		issue.Unindent(`
      @(END/u)
        A checkmark \ux271 symbol
        |- END`),
		`malformed unicode escape sequence (line: 2, column: 15)`)
}

func TestMLCommentAfterHeredocTag(t *testing.T) {
	expectHeredoc(t, issue.Unindent(`
      @(END) /* comment after tag */
      This is
      heredoc text
      END`),
		"This is\nheredoc text\n")
}

func TestCommentAfterHeredocTag(t *testing.T) {
	expectHeredoc(t, issue.Unindent(`
      @(END) # comment after tag
      This is
      heredoc text
      END`),
		"This is\nheredoc text\n")
}

func TestVariable(t *testing.T) {
	expectDump(t,
		`$var`,
		`(var "var")`)

	expectDump(t,
		`$var::b`,
		`(var "var::b")`)

	expectDump(t,
		`$::var::b`,
		`(var "::var::b")`)

	expectDump(t,
		`$::var::_b`,
		`(var "::var::_b")`)

	expectDump(t,
		`$2`,
		`(var 2)`)

	expectDump(t,
		`$`,
		`(var "")`)

	expectError(t,
		`$var:b`,
		`unexpected token ':' (line: 1, column: 5)`)

	expectError(t,
		`$Var`,
		`invalid variable name (line: 1, column: 2)`)

	expectError(t,
		`$:var::b`,
		`invalid variable name (line: 1, column: 1)`)

	expectError(t,
		`$::var::B`,
		`invalid variable name (line: 1, column: 1)`)

	expectError(t,
		`$::var::_b::c`,
		`invalid variable name (line: 1, column: 1)`)

	expectError(t,
		`$::_var::b`,
		`unexpected token '_' (line: 1, column: 4)`)
}

func TestArray(t *testing.T) {
	expectDump(t,
		`[1,2,3]`,
		`(array 1 2 3)`)

	expectDump(t,
		`[1,2,3,]`,
		`(array 1 2 3)`)

	expectDump(t,
		`[1,2,a=>3]`,
		`(array 1 2 (hash (=> (qn "a") 3)))`)

	expectDump(t,
		`[1,2,a=>3,b=>4]`,
		`(array 1 2 (hash (=> (qn "a") 3) (=> (qn "b") 4)))`)

	expectDump(t,
		`[1,2,a=>3,b=>4,5]`,
		`(array 1 2 (hash (=> (qn "a") 3) (=> (qn "b") 4)) 5)`)

	expectDump(t,
		`[1,2,{a=>3},b=>4,5]`,
		`(array 1 2 (hash (=> (qn "a") 3)) (hash (=> (qn "b") 4)) 5)`)

	expectError(t,
		`[1,2 3]`,
		`expected one of ',' or ']', got 'integer literal' (line: 1, column: 6)`)

	expectError(t,
		`[1,2,3`,
		`expected one of ',' or ']', got 'EOF' (line: 1, column: 7)`)
}

func TestHash(t *testing.T) {
	expectDump(t,
		`{ a => true, b => false, c => undef, d => 12, e => 23.5, c => 'hello' }`,
		`(hash (=> (qn "a") true) (=> (qn "b") false) (=> (qn "c") nil) (=> (qn "d") 12) (=> (qn "e") 23.5) (=> (qn "c") "hello"))`)

	expectDump(t,
		`{a => 1, b => 2,}`,
		`(hash (=> (qn "a") 1) (=> (qn "b") 2))`)

	expectDump(t,
		`{type => consumes, function => site, application => produces,}`,
		`(hash (=> (qn "type") (qn "consumes")) (=> (qn "function") (qn "site")) (=> (qn "application") (qn "produces")))`)

	expectError(t,
		`{a => 1, b, 2}`,
		`expected '=>' to follow hash key (line: 1, column: 12)`)

	expectError(t,
		`{a => 1 b => 2}`,
		`expected one of ',' or '}', got 'identifier' (line: 1, column: 9)`)

	expectError(t,
		`{a => 1, b => 2`,
		`expected one of ',' or '}', got 'EOF' (line: 1, column: 16)`)
}

func TestBlock(t *testing.T) {
	expectBlock(t,
		issue.Unindent(`
      $t = 'the'
      $r = 'revealed'
      $map = {'ipl' => 'meaning', 42.0 => 'life'}
      "$t ${map['ipl']} of ${map[42.0]}${[3, " is not ${r}"][1]} here"`),
		`(block `+
			`(= (var "t") "the") `+
			`(= (var "r") "revealed") `+
			`(= (var "map") (hash (=> "ipl" "meaning") (=> 4.2e+01 "life"))) `+
			`(concat (str (var "t")) " " (str (access (var "map") "ipl")) " of " (str (access (var "map") 4.2e+01)) (str (access (array 3 (concat " is not " (str (var "r")))) 1)) " here"))`)

	expectBlock(t,
		issue.Unindent(`
      $t = 'the';
      $r = 'revealed';
      $map = {'ipl' => 'meaning', 42.0 => 'life'};
      "$t ${map['ipl']} of ${map[42.0]}${[3, " is not ${r}"][1]} here"`),
		`(block `+
			`(= (var "t") "the") `+
			`(= (var "r") "revealed") `+
			`(= (var "map") (hash (=> "ipl" "meaning") (=> 4.2e+01 "life"))) `+
			`(concat (str (var "t")) " " (str (access (var "map") "ipl")) " of " (str (access (var "map") 4.2e+01)) (str (access (array 3 (concat " is not " (str (var "r")))) 1)) " here"))`)

	expectError(t,
		issue.Unindent(`
      $a = 'a',
      $b = 'b'`),
		`Extraneous comma between statements (line: 1, column: 10)`)
}

func TestFunctionDefinition(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      function myFunc(Integer[0,3] $first, $untyped, String $nxt = 'hello') >> Float {
         23.8
      }`),
		`(function {`+
			`:name "myFunc" `+
			`:params {`+
			`:first {:type (access (qr "Integer") 0 3)} `+
			`:untyped {} `+
			`:nxt {:type (qr "String") :value "hello"}} `+
			`:body [23.8] `+
			`:returns (qr "Float")})`)

	expectDump(t,
		issue.Unindent(`
      function myFunc(Integer *$numbers) >> Integer {
         $numbers.size
      }`),
		`(function {`+
			`:name "myFunc" `+
			`:params {`+
			`:numbers {:type (qr "Integer") :splat true}} `+
			`:body [`+
			`(call-method {:functor (. (var "numbers") (qn "size")) :args []})] `+
			`:returns (qr "Integer")})`)

	expectError(t,
		issue.Unindent(`
      function foo($1) {}`),
		`expected variable declaration (line: 1, column: 16)`)

	expectError(t,
		issue.Unindent(`
      function myFunc(Integer *numbers) >> Integer {
         numbers.size
      }`),
		`expected variable declaration (line: 1, column: 33)`)

	expectError(t,
		issue.Unindent(`
      function myFunc(Integer *$numbers) >> $var {
         numbers.size
      }`),
		`expected type name (line: 1, column: 43)`)

	expectError(t,
		issue.Unindent(`
      function 'myFunc'() {
         true
      }`),
		`expected a name to follow keyword 'function' (line: 1, column: 10)`)

	expectError(t,
		issue.Unindent(`
      function myFunc() true`),
		`expected token '{', got 'boolean literal' (line: 1, column: 19)`)

	expectError(t,
		issue.Unindent(`
      function myFunc() >> Boolean true`),
		`expected token '{', got 'boolean literal' (line: 1, column: 30)`)
}

func TestPlanDefinition(t *testing.T) {
	expectDump(t, `plan foo { }`,
		`(plan {:name "foo" :body []})`, TasksEnabled)

	expectDump(t,
		issue.Unindent(`
      plan foo {
        $a = 10
        $b = 20
     }`),
		`(plan {:name "foo" :body [(= (var "a") 10) (= (var "b") 20)]})`,
		TasksEnabled)

	expectDump(t, `plan foo($p1 = 'yo', $p2) { }`,
		`(plan {:name "foo" :params {:p1 {:value "yo"} :p2 {}} :body []})`, TasksEnabled)

	expectError(t, `$a = plan`,
		`expected a name to follow keyword 'plan' (line: 1, column: 10)`, TasksEnabled)

	expectDump(t, `$a = plan`,
		`(= (var "a") (qn "plan"))`)
}

func TestWorkflowDefinition(t *testing.T) {
	expectDump(t, `workflow foo { }`,
		`(activity {:name "foo" :style "workflow"})`, WorkflowEnabled)

	expectDump(t,
		issue.Unindent(`
      workflow foo {} {
        resource bar {}
      }`),
		`(activity {:name "foo" :style "workflow" :definition (block `+
			`(activity {:name "foo::bar" :style "resource"}))})`,
		WorkflowEnabled)

	expectDump(t,
		issue.Unindent(`
      workflow foo {} {
        resource bar {
          type => Genesis::Aws::Instance
        } {
          x => 2,
          y => {
            a => 'a'
          }
        }
      }`),
		`(activity {:name "foo" :style "workflow" :definition (block `+
			`(activity {:name "foo::bar" :style "resource" :properties (hash (=> (qn "type") (qr "Genesis::Aws::Instance"))) :definition (hash `+
			`(=> (qn "x") 2) `+
			`(=> (qn "y") (hash (=> (qn "a") "a"))))}))})`,
		WorkflowEnabled)

	expectDump(t,
		issue.Unindent(`
      workflow foo {} {
        resource bar {
          type => Genesis::Aws::Instance,
					repeat => {
						each => $y,
						as => $x
					}
        } {
          x => $x,
        }
      }`),
		`(activity {:name "foo" :style "workflow" :definition (block `+
			`(activity {:name "foo::bar" :style "resource" :properties (hash `+
			`(=> (qn "type") (qr "Genesis::Aws::Instance")) `+
			`(=> (qn "repeat") (hash `+
			`(=> (qn "each") (call-method {:functor (. (qr "Deferred") (qn "new")) :args ["$y"]})) `+
			`(=> (qn "as") (array (param {:name "x"})))))) `+
			`:definition (hash (=> (qn "x") (call-method {:functor (. (qr "Deferred") (qn "new")) :args ["$x"]})))}))})`,
		WorkflowEnabled)

	expectDump(t,
		issue.Unindent(`
      workflow foo {} {
        action bar { guard => true } {
          function read {
            true
          }
        }
      }`),
		`(activity {:name "foo" :style "workflow" :definition (block `+
			`(activity {:name "foo::bar" :style "action" :properties (hash (=> (qn "guard") true)) `+
			`:definition (block (function {:name "read" :body [true]}))}))})`,
		WorkflowEnabled)

	expectDump(t,
		issue.Unindent(`
      workflow foo {} {
        action bar {} {
          function delete {
            notice('hello from delete')
          }
          function read {
            notice('hello from read')
          }
          function upsert {
            notice('hello from upsert')
          }
        }
      }`),
		`(activity {:name "foo" :style "workflow" :definition (block `+
			`(activity {:name "foo::bar" :style "action" :definition (block `+
			`(function {:name "delete" :body [(invoke {:functor (qn "notice") :args ["hello from delete"]})]}) `+
			`(function {:name "read" :body [(invoke {:functor (qn "notice") :args ["hello from read"]})]}) `+
			`(function {:name "upsert" :body [(invoke {:functor (qn "notice") :args ["hello from upsert"]})]}))}))})`,
		WorkflowEnabled)
}

func TestNodeDefinition(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      node default {
      }`),
		`(node {:matches [(default)] :body []})`)

	expectDump(t,
		issue.Unindent(`
      node /[a-f].*/ {
      }`),
		`(node {:matches [(regexp "[a-f].*")] :body []})`)

	expectDump(t,
		issue.Unindent(`
      node /[a-f].*/, "example.com" {
      }`),
		`(node {:matches [(regexp "[a-f].*") "example.com"] :body []})`)

	expectDump(t,
		issue.Unindent(`
      node /[a-f].*/, example.com {
      }`),
		`(node {:matches [(regexp "[a-f].*") "example.com"] :body []})`)

	expectDump(t,
		issue.Unindent(`
      node /[a-f].*/, 192.168.0.1, 34, "$x.$y" {
      }`),
		`(node {:matches [(regexp "[a-f].*") "192.168.0.1" "34" (concat (str (var "x")) "." (str (var "y")))] :body []})`)

	expectDump(t,
		issue.Unindent(`
      node /[a-f].*/, 192.168.0.1, 34, 'some.string', {
      }`),
		`(node {:matches [(regexp "[a-f].*") "192.168.0.1" "34" "some.string"] :body []})`)

	expectDump(t,
		issue.Unindent(`
      node /[a-f].*/ inherits 192.168.0.1 {
      }`),
		`(node {:matches [(regexp "[a-f].*")] :parent "192.168.0.1" :body []})`)

	expectDump(t,
		issue.Unindent(`
      node default {
        notify { x: message => 'node default' }
      }`),
		`(node {:matches [(default)] :body [(resource {:type (qn "notify") :bodies [{:title (qn "x") :ops [(=> "message" "node default")]}]})]})`)

	expectError(t,
		issue.Unindent(`
      node [hosta.com, hostb.com] {
      }`),
		issue.Unindent(`hostname expected (line: 1, column: 7)`))

	expectError(t,
		issue.Unindent(`
      node example.* {
      }`),
		issue.Unindent(`expected name or number to follow '.' (line: 1, column: 15)`))
}

func TestSiteDefinition(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      site {
      }`),
		`(site)`)

	expectDump(t,
		issue.Unindent(`
      site {
        notify { x: message => 'node default' }
      }`),
		`(site (resource {:type (qn "notify") :bodies [{:title (qn "x") :ops [(=> "message" "node default")]}]}))`)
}

func TestTypeDefinition(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      type MyType {
        # What statements that can be included here is not yet speced
      }`),
		`(type-definition "MyType" "" (block))`)

	expectDump(t,
		issue.Unindent(`
      type MyType inherits OtherType {
        # What statements that can be included here is not yet speced
      }`),
		`(type-definition "MyType" "OtherType" (block))`)

	expectError(t,
		issue.Unindent(`
      type MyType inherits OtherType [{
        # What statements that can be included here is not yet speced
      }]`),
		`expected token '{', got '[' (line: 1, column: 32)`)

	expectError(t,
		issue.Unindent(`
      type MyType inherits $other {
        # What statements that can be included here is not yet speced
      }`),
		`expected type name to follow 'inherits' (line: 1, column: 28)`)

	expectError(t,
		issue.Unindent(`
      type MyType[a,b] {
        # What statements that can be included here is not yet speced
      }`),
		`expected type name to follow 'type' (line: 1, column: 19)`)

	expectError(t,
		issue.Unindent(`
      type MyType << {
        # What statements that can be included here is not yet speced
      }`),
		`unexpected token '<<' (line: 1, column: 15)`)
}

func TestTypeAlias(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      type MyType = Object[{
        attributes => {
        name => String,
        number => Integer
        }
      }]`),
		`(type-alias "MyType" (access (qr "Object") (hash (=> (qn "attributes") (hash (=> (qn "name") (qr "String")) (=> (qn "number") (qr "Integer")))))))`)

	expectError(t,
		`type Mod::myType[a, b] = Object[{}]`,
		`invalid type name (line: 1, column: 6)`)
}

func TestTypeMapping(t *testing.T) {
	expectDump(t,
		`type Runtime[ruby, 'MyModule::MyObject'] = MyPackage::MyObject`,
		`(type-mapping (access (qr "Runtime") (qn "ruby") "MyModule::MyObject") (qr "MyPackage::MyObject"))`)

	expectDump(t,
		`type Runtime[ruby, [/^MyPackage::(\w+)$/, 'MyModule::\1']] = [/^MyModule::(\w+)$/, 'MyPackage::\1']`,
		`(type-mapping (access (qr "Runtime") (qn "ruby") (array (regexp "^MyPackage::(\\w+)$") "MyModule::\\1")) (array (regexp "^MyModule::(\\w+)$") "MyPackage::\\1"))`)
}

func TestClass(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      class myclass {
      }`),
		`(class {:name "myclass" :body []})`)

	expectDump(t,
		issue.Unindent(`
      class myclass {
        class inner {
        }
      }`),
		`(class {:name "myclass" :body [(class {:name "myclass::inner" :body []})]})`)

	expectDump(t,
		issue.Unindent(`
      class ::myclass {
        class inner {
        }
      }`),
		`(class {:name "myclass" :body [(class {:name "myclass::inner" :body []})]})`)

	expectDump(t,
		issue.Unindent(`
      class ::myclass {
        class ::inner {
        }
      }`),
		`(class {:name "myclass" :body [(class {:name "myclass::inner" :body []})]})`)

	expectDump(t,
		issue.Unindent(`
      class myclass inherits other {
      }`),
		`(class {:name "myclass" :parent "other" :body []})`)

	expectDump(t,
		issue.Unindent(`
      class myclass inherits default {
      }`),
		`(class {:name "myclass" :parent "default" :body []})`)

	expectDump(t,
		issue.Unindent(`
      class myclass($a, $b = 2) {
      }`),
		`(class {:name "myclass" :params {:a {} :b {:value 2}} :body []})`)

	expectDump(t,
		issue.Unindent(`
      class myclass($a, $b = 2) inherits other {
      }`),
		`(class {:name "myclass" :parent "other" :params {:a {} :b {:value 2}} :body []})`)

	expectError(t,
		issue.Unindent(`
      class 'myclass' {
      }`),
		`a quoted string is not valid as a name at this location (line: 1, column: 7)`)

	expectError(t,
		issue.Unindent(`
      class class {
      }`),
		`'class' keyword not allowed at this location (line: 1, column: 7)`)

	expectError(t,
		issue.Unindent(`
      class [a, b] {
      }`),
		`expected name of class (line: 1, column: 7)`)
}

func TestDefinition(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      define apache::vhost (
        Integer $port,
        String[1] $docroot,
        String[1] $servername = $title,
        String $vhost_name = '*',
      ) {
        include apache # contains package['httpd'] and service['httpd']
        include apache::params # contains common config settings

        $vhost_dir = $apache::params::vhost_dir

        # the template used below can access all of the parameters and variable from above.
        file { "${vhost_dir}/${servername}.conf":
          ensure  => file,
          owner   => 'www',
          group   => 'www',
          mode    => '0644',
          content => template('apache/vhost-default.conf.erb'),
          require => Package['httpd'],
          notify  => Service['httpd'],
        }
      }`),
		`(define {`+
			`:name "apache::vhost" `+
			`:params {`+
			`:port {:type (qr "Integer")} `+
			`:docroot {:type (access (qr "String") 1)} `+
			`:servername {:type (access (qr "String") 1) :value (var "title")} `+
			`:vhost_name {:type (qr "String") :value "*"}} `+
			`:body [`+
			`(invoke {:functor (qn "include") :args [(qn "apache")]}) `+
			`(invoke {:functor (qn "include") :args [(qn "apache::params")]}) `+
			`(= (var "vhost_dir") (var "apache::params::vhost_dir")) `+
			`(resource {`+
			`:type (qn "file") `+
			`:bodies [{`+
			`:title (concat (str (var "vhost_dir")) "/" (str (var "servername")) ".conf") `+
			`:ops [`+
			`(=> "ensure" (qn "file")) `+
			`(=> "owner" "www") `+
			`(=> "group" "www") `+
			`(=> "mode" "0644") `+
			`(=> "content" (call {:functor (qn "template") :args ["apache/vhost-default.conf.erb"]})) `+
			`(=> "require" (access (qr "Package") "httpd")) `+
			`(=> "notify" (access (qr "Service") "httpd"))]}]})]})`)
}

func TestCapabilityMapping(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      MyCap produces Cap {
        attr => $value
      }`),
		`(produces (qr "MyCap") ["Cap" (=> "attr" (var "value"))])`)

	expectDump(t,
		issue.Unindent(`
      attr produces Cap {}`),
		`(produces (qn "attr") ["Cap"])`)
}

func TestApplication(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      application lamp(
        String $db_user,
        String $db_password,
        String $docroot = '/var/www/html',
      ){
        lamp::web { $name:
          docroot => $docroot,
          export => Http["lamp-${name}"],
        }

        lamp::app { $name:
          consume => Sql["lamp-${name}"],
          export => Http["lamp-${name}"],
        }

        lamp::db { $name:
          db_user     => $db_user,
          db_name     => $db_name,
          export      => Sql["lamp-${name}"],
        }
      }`),

		`(application {`+
			`:name "lamp" `+
			`:params {`+
			`:db_user {:type (qr "String")} `+
			`:db_password {:type (qr "String")} `+
			`:docroot {:type (qr "String") :value "/var/www/html"}} `+
			`:body [`+
			`(resource {`+
			`:type (qn "lamp::web") `+
			`:bodies [{`+
			`:title (var "name") `+
			`:ops [(=> "docroot" (var "docroot")) (=> "export" (access (qr "Http") (concat "lamp-" (str (var "name")))))]}]}) `+
			`(resource {`+
			`:type (qn "lamp::app") `+
			`:bodies [{`+
			`:title (var "name") `+
			`:ops [(=> "consume" (access (qr "Sql") (concat "lamp-" (str (var "name"))))) (=> "export" (access (qr "Http") (concat "lamp-" (str (var "name")))))]}]}) `+
			`(resource {`+
			`:type (qn "lamp::db") `+
			`:bodies [{`+
			`:title (var "name") `+
			`:ops [(=> "db_user" (var "db_user")) (=> "db_name" (var "db_name")) (=> "export" (access (qr "Sql") (concat "lamp-" (str (var "name")))))]}]})]})`)
}

func TestCallNamed(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = wrap(myFunc(3, 'vx', 'd"x') |Integer $r| >> Integer { $r + 2 })`),
		`(= (var "x") (call {:functor (qn "wrap") :args [(call {:functor (qn "myFunc") :args [3 "vx" "d\"x"] :block (lambda {:params {:r {:type (qr "Integer")}} :returns (qr "Integer") :body [(+ (var "r") 2)]})})]}))`)

	expectDump(t,
		`notice hello()`, `(invoke {:functor (qn "notice") :args [(call {:functor (qn "hello") :args []})]})`)

	expectDump(t,
		`notice hello(), 'world'`, `(invoke {:functor (qn "notice") :args [(call {:functor (qn "hello") :args []}) "world"]})`)

	expectBlock(t,
		issue.Unindent(`
      $x = $y.myFunc
      callIt(*$x)
      (2 + 3).with() |$x| { notice $x }`),
		`(block (= (var "x") (call-method {:functor (. (var "y") (qn "myFunc")) :args []})) (invoke {:functor (qn "callIt") :args [(unfold (var "x"))]}) (call-method {:functor (. (paren (+ 2 3)) (qn "with")) :args [] :block (lambda {:params {:x {}} :body [(invoke {:functor (qn "notice") :args [(var "x")]})]})}))`)

	expectError(t,
		issue.Unindent(`
      $x = myFunc(3`),
		`expected one of ',' or ')', got 'EOF' (line: 1, column: 14)`)

	expectError(t,
		issue.Unindent(`
      $x = myFunc() || $r + 2 }`),
		`expected token '{', got 'variable' (line: 1, column: 18)`)

}

func TestCallNamedNoArgs(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = wrap(myFunc |Integer $r| >> Integer { $r + 2 })`),
		`(= (var "x") (call {:functor (qn "wrap") :args [(call {:functor (qn "myFunc") :args [] :block (lambda {:params {:r {:type (qr "Integer")}} :returns (qr "Integer") :body [(+ (var "r") 2)]})})]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = [myFunc()]`),
		`(= (var "x") (array (call {:functor (qn "myFunc") :args []})))`)

}

func TestCallMethod(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = $y.max(23)`),
		`(= (var "x") (call-method {:functor (. (var "y") (qn "max")) :args [23]}))`)
}

func TestCallMethodArgsLambda(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = $y.max(23) |$x| { $x }`),
		`(= (var "x") (call-method {:functor (. (var "y") (qn "max")) :args [23] :block (lambda {:params {:x {}} :body [(var "x")]})}))`)
}

func TestCallMethodNoArgs(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = $y.max`),
		`(= (var "x") (call-method {:functor (. (var "y") (qn "max")) :args []}))`)

	expectDump(t,
		issue.Unindent(`
      $x == $y.max`),
		`(== (var "x") (call-method {:functor (. (var "y") (qn "max")) :args []}))`)

	expectDump(t,
		issue.Unindent(`
      "${x[3].y}"`),
		`(concat (str (call-method {:functor (. (access (var "x") 3) (qn "y")) :args []})))`)

	expectDump(t,
		issue.Unindent(`
      "${x[3].y.z}"`),
		`(concat (str (call-method {:functor (. (call-method {:functor (. (access (var "x") 3) (qn "y")) :args []}) (qn "z")) :args []})))`)
}

func TestCallMethodNoArgsLambda(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = $y.max |$x| { $x }`),
		`(= (var "x") (call-method {:functor (. (var "y") (qn "max")) :args [] :block (lambda {:params {:x {}} :body [(var "x")]})}))`)
}

func TestCallFuncNoArgsLambdaThenCall(t *testing.T) {
	expectDump(t, `func |$x| { $x }.newfunc`,
		`(call-method {:functor (. (call {:functor (qn "func") :args [] :block (lambda {:params {:x {}} :body [(var "x")]})}) (qn "newfunc")) :args []})`)
}

func TestCallType(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = type(3)`),
		`(= (var "x") (call {:functor (qn "type") :args [3]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = [type(3)]`),
		`(= (var "x") (array (call {:functor (qn "type") :args [3]})))`)

	expectDump(t,
		issue.Unindent(`
      $x = {type(3) => 'v'}`),
		`(= (var "x") (hash (=> (call {:functor (qn "type") :args [3]}) "v")))`)

	expectDump(t,
		issue.Unindent(`
      $x = {'v' => type(3)}`),
		`(= (var "x") (hash (=> "v" (call {:functor (qn "type") :args [3]}))))`)

	expectDump(t, `with |$x,$y=type| {}`,
		`(invoke {:functor (qn "with") :args [] :block (lambda {:params {:x {} :y {:value (qn "type")}} :body []})})`)
}

func TestCallTypeMethod(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = $x.type(3)`),
		`(= (var "x") (call-method {:functor (. (var "x") (qn "type")) :args [3]}))`)
}

func TestImplicitNewWithDot(t *testing.T) {
	expectDump(t, `Foo(3).with |$f| { $f }`,
		`(call-method {:functor (. (call {:functor (qr "Foo") :args [3]}) (qn "with")) :args [] :block (lambda {:params {:f {}} :body [(var "f")]})})`)
}

func TestImplicitNewWithDotDot(t *testing.T) {
	expectDump(t, `Foo(3).type_of.with |$f| { $f }`,
		`(call-method {:functor (. (call-method {:functor (. (call {:functor (qr "Foo") :args [3]}) (qn "type_of")) :args []}) (qn "with")) :args [] :block (lambda {:params {:f {}} :body [(var "f")]})})`)
}

func TestLineComment(t *testing.T) {
	expectBlock(t,
		issue.Unindent(`
      $x = 'y'
      # The above is a variable assignment
      # and here is a notice of the assigned
      # value
      #
      notice($y)`),
		`(block (= (var "x") "y") (invoke {:functor (qn "notice") :args [(var "y")]}))`)
}

func TestIdentifiers(t *testing.T) {
	expectDump(t,
		`name`,
		`(qn "name")`)

	expectDump(t,
		`Name`,
		`(qr "Name")`)

	expectDump(t,
		`Ab::Bc`,
		`(qr "Ab::Bc")`)

	expectDump(t,
		`$x = ::assertType(::TheType, $y)`,
		`(= (var "x") (call {:functor (qn "::assertType") :args [(qr "::TheType") (var "y")]}))`)

	expectError(t,
		`abc:cde`,
		`unexpected token ':' (line: 1, column: 4)`)

	expectError(t,
		`Ab::bc`,
		`invalid type name (line: 1, column: 1)`)

	expectError(t,
		`$x = ::3m`,
		`:: not followed by name segment (line: 1, column: 6)`)
}

func TestRestOfLineComment(t *testing.T) {
	expectBlock(t,
		issue.Unindent(`
      $x = 'y' # A variable assignment
      notice($y)`),
		`(block (= (var "x") "y") (invoke {:functor (qn "notice") :args [(var "y")]}))`)

	expectBlock(t,
		issue.Unindent(`
      # [*version*]
      #   The package version to install, used to set the package name.
      #   Defaults to undefined`),
		`(block)`)
}

func TestMultilineComment(t *testing.T) {
	expectBlock(t,
		issue.Unindent(`
      $x = 'y'
      /* The above is a variable assignment
         and here is a notice of the assigned
         value
      */
      notice($y)`),
		`(block (= (var "x") "y") (invoke {:functor (qn "notice") :args [(var "y")]}))`)
}

func TestSingleQuote(t *testing.T) {
	expectDump(t, `$x = 'a string'`, `(= (var "x") "a string")`)

	expectDump(t, `$x = 'a \'string\' with \\'`, `(= (var "x") "a 'string' with \\")`)

	expectError(t,
		issue.Unindent(`
      $x = 'y
      notice($x)`),
		"unterminated single quoted string (line: 1, column: 6)")
}

func TestUnterminatedQuoteEscapedEnd(t *testing.T) {
	expectError(t,
		issue.Unindent(`
      $x = 'y\`),
		"unterminated single quoted string (line: 1, column: 6)")
}

func TestStrayTilde(t *testing.T) {
	expectError(t,
		issue.Unindent(`
      $x ~ 'y'
      notice($x)`),
		"unexpected token '~' (line: 1, column: 4)")
}

func TestUnknownToken(t *testing.T) {
	expectError(t,
		issue.Unindent(`
      $x ^ 'y'
      notice($x)`),
		"unexpected token '^' (line: 1, column: 4)")
}

func TestUnterminatedComment(t *testing.T) {
	expectError(t,
		issue.Unindent(`
      $x = 'y'
      /* The above is a variable assignment
      notice($y)`),
		"unterminated /* */ comment (line: 2, column: 1)")
}

func TestIf(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = if $y {
        true
      } else {
        false
      }`),
		`(= (var "x") (if {:test (var "y") :then [true] :else [false]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = if $y > 2 {
      } else {
        false
      }`),
		`(= (var "x") (if {:test (> (var "y") 2) :then [] :else [false]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = if $y != 34 {
        true
      } else {
      }`),
		`(= (var "x") (if {:test (!= (var "y") 34) :then [true] :else []}))`)

	expectDump(t,
		issue.Unindent(`
      $x = if $y {
        1
      } elsif $z {
        2
      } else {
        3
      }`),
		`(= (var "x") (if {:test (var "y") :then [1] :else [(if {:test (var "z") :then [2] :else [3]})]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = if $y == if $x {
        true
      } { false }`),
		`(= (var "x") (if {:test (== (var "y") (if {:test (var "x") :then [true]})) :then [false]}))`)

	expectError(t,
		`$x = else { 3 }`,
		`unexpected token 'else' (line: 1, column: 6)`)
}

func TestUnless(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      $x = unless $y {
        true
      } else {
        false
      }`),
		`(= (var "x") (unless {:test (var "y") :then [true] :else [false]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = unless $y {
      } else {
        false
      }`),
		`(= (var "x") (unless {:test (var "y") :then [] :else [false]}))`)

	expectDump(t,
		issue.Unindent(`
      $x = unless $y {
        true
      } else {
      }`),
		`(= (var "x") (unless {:test (var "y") :then [true] :else []}))`)

	expectDump(t,
		issue.Unindent(`
      $x = if $y == unless $x {
        true
      } { false }`),
		`(= (var "x") (if {:test (== (var "y") (unless {:test (var "x") :then [true]})) :then [false]}))`)

	expectError(t,
		issue.Unindent(`
      $x = unless $y {
        1
      } elsif $z {
        2
      } else {
        3
      }`),
		`elsif not supported in unless expression (line: 3, column: 8)`)
}

func TestSelector(t *testing.T) {
	expectDump(t,
		`$rootgroup = $facts['os']['family'] ? 'Solaris' => 'wheel'`,
		`(= (var "rootgroup") (? (access (access (var "facts") "os") "family") [(=> "Solaris" "wheel")]))`)

	expectDump(t,
		issue.Unindent(`
      $rootgroup = $facts['os']['family'] ? {
        'Solaris'          => 'wheel',
        /(Darwin|FreeBSD)/ => 'wheel',
        default            => 'root'
      }`),
		`(= (var "rootgroup") (? (access (access (var "facts") "os") "family") [(=> "Solaris" "wheel") (=> (regexp "(Darwin|FreeBSD)") "wheel") (=> (default) "root")]))`)

	expectDump(t,
		issue.Unindent(`
      $rootgroup = $facts['os']['family'] ? {
        'Solaris'          => 'wheel',
        /(Darwin|FreeBSD)/ => 'wheel',
        default            => 'root',
      }`),
		`(= (var "rootgroup") (? (access (access (var "facts") "os") "family") [(=> "Solaris" "wheel") (=> (regexp "(Darwin|FreeBSD)") "wheel") (=> (default) "root")]))`)
}

func TestCase(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
    case $facts['os']['name'] {
      'Solaris':           { include role::solaris } # Apply the solaris class
      'RedHat', 'CentOS':  { include role::redhat  } # Apply the redhat class
      /^(Debian|Ubuntu)$/: { include role::debian  } # Apply the debian class
      default:             { include role::generic } # Apply the generic class
    }`),
		`(case (access (access (var "facts") "os") "name") [`+
			`{:when ["Solaris"] :then [(invoke {:functor (qn "include") :args [(qn "role::solaris")]})]} `+
			`{:when ["RedHat" "CentOS"] :then [(invoke {:functor (qn "include") :args [(qn "role::redhat")]})]} `+
			`{:when [(regexp "^(Debian|Ubuntu)$")] :then [(invoke {:functor (qn "include") :args [(qn "role::debian")]})]} `+
			`{:when [(default)] :then [(invoke {:functor (qn "include") :args [(qn "role::generic")]})]}])`)
}

func TestAccess(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      Struct[{
        Optional[description] => String,
        Optional[sensitive] => Boolean,
        type => Type}]`),
		`(access (qr "Struct") `+
			`(hash `+
			`(=> (access (qr "Optional") (qn "description")) (qr "String")) `+
			`(=> (access (qr "Optional") (qn "sensitive")) (qr "Boolean")) `+
			`(=> (qn "type") (qr "Type"))))`)

	expectDump(t,
		issue.Unindent(`
      Struct[
        Optional[description] => String,
        Optional[sensitive] => Boolean,
        type => Type]`),
		`(access (qr "Struct") `+
			`(hash `+
			`(=> (access (qr "Optional") (qn "description")) (qr "String")) `+
			`(=> (access (qr "Optional") (qn "sensitive")) (qr "Boolean")) `+
			`(=> (qn "type") (qr "Type"))))`)
}

func TestResource(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      file { '/tmp/foo':
        mode => '0640',
        ensure => present
      }`),
		`(resource {`+
			`:type (qn "file") `+
			`:bodies [{:title "/tmp/foo" :ops [(=> "mode" "0640") (=> "ensure" (qn "present"))]}]})`)

	expectDump(t,
		issue.Unindent(`
      file { '/tmp/foo':
        ensure => file,
        * => $file_ownership
      }`),
		`(resource {`+
			`:type (qn "file") `+
			`:bodies [{:title "/tmp/foo" :ops [(=> "ensure" (qn "file")) (splat-hash (var "file_ownership"))]}]})`)

	expectDump(t,
		issue.Unindent(`
      @file { '/tmp/foo':
        mode => '0640',
        ensure => present
      }`),
		`(resource {`+
			`:type (qn "file") `+
			`:bodies [{:title "/tmp/foo" :ops [(=> "mode" "0640") (=> "ensure" (qn "present"))]}] `+
			`:form "virtual"})`)

	expectDump(t,
		issue.Unindent(`
      @@file { '/tmp/foo':
        mode => '0640',
        ensure => present
      }`),
		`(resource {`+
			`:type (qn "file") `+
			`:bodies [{:title "/tmp/foo" :ops [(=> "mode" "0640") (=> "ensure" (qn "present"))]}] `+
			`:form "exported"})`)

	expectDump(t,
		issue.Unindent(`
      class { some_title: }`),
		`(resource {:type (qn "class") :bodies [{:title (qn "some_title") :ops []}]})`)

	expectDump(t,
		issue.Unindent(`
      file { '/tmp/foo': }`),
		`(resource {`+
			`:type (qn "file") `+
			`:bodies [{:title "/tmp/foo" :ops []}]})`)

	expectDump(t,
		issue.Unindent(`
      package { 'openssh-server':
        ensure => present,
      } -> # and then:
      file { '/etc/ssh/sshd_config':
        ensure => file,
        mode   => '0600',
        source => 'puppet:///modules/sshd/sshd_config',
      } ~> # and then:
      service { 'sshd':
        ensure => running,
        enable => true,
      }`),
		`(~> (-> `+
			`(resource {`+
			`:type (qn "package") `+
			`:bodies [{`+
			`:title "openssh-server" `+
			`:ops [(=> "ensure" (qn "present"))]}]}) `+
			`(resource {`+
			`:type (qn "file") `+
			`:bodies [{`+
			`:title "/etc/ssh/sshd_config" `+
			`:ops [(=> "ensure" (qn "file")) (=> "mode" "0600") (=> "source" "puppet:///modules/sshd/sshd_config")]}]})) `+
			`(resource {`+
			`:type (qn "service") `+
			`:bodies [{`+
			`:title "sshd" `+
			`:ops [(=> "ensure" (qn "running")) (=> "enable" true)]}]}))`)

	expectDump(t,
		issue.Unindent(`
      package { 'openssh-server':
        ensure => present,
      } <- # and then:
      file { '/etc/ssh/sshd_config':
        ensure => file,
        mode   => '0600',
        source => 'puppet:///modules/sshd/sshd_config',
      } <~ # and then:
      service { 'sshd':
        ensure => running,
        enable => true,
      }`),
		`(<~ (<- `+
			`(resource {`+
			`:type (qn "package") `+
			`:bodies [{`+
			`:title "openssh-server" `+
			`:ops [(=> "ensure" (qn "present"))]}]}) `+
			`(resource {`+
			`:type (qn "file") `+
			`:bodies [{`+
			`:title "/etc/ssh/sshd_config" `+
			`:ops [(=> "ensure" (qn "file")) (=> "mode" "0600") (=> "source" "puppet:///modules/sshd/sshd_config")]}]})) `+
			`(resource {`+
			`:type (qn "service") `+
			`:bodies [{`+
			`:title "sshd" `+
			`:ops [(=> "ensure" (qn "running")) (=> "enable" true)]}]}))`)

	expectError(t,
		issue.Unindent(`
      file { '/tmp/foo':
        mode => '0640',
        ensure => present
      `),
		`expected token '}', got 'EOF' (line: 4, column: 1)`)

	expectError(t,
		issue.Unindent(`
      file { '/tmp/foo':
        mode, '0640',
        ensure, present
      }`),
		`invalid attribute operation (line: 2, column: 8)`)

	expectError(t,
		issue.Unindent(`
      file { '/tmp/foo':
        'mode' => '0640',
        'ensure' => present
      }`),
		`expected attribute name (line: 2, column: 3)`)
}

func TestMultipleBodies(t *testing.T) {
	expectDump(t,
		issue.Unindent(`
      file { '/tmp/foo':
        mode => '0640',
        ensure => present;
      '/tmp/bar':
        mode => '0640',
        ensure => present;
      }`),
		`(resource {:type (qn "file") :bodies [`+
			`{:title "/tmp/foo" :ops [(=> "mode" "0640") (=> "ensure" (qn "present"))]} `+
			`{:title "/tmp/bar" :ops [(=> "mode" "0640") (=> "ensure" (qn "present"))]}]})`)

	expectError(t,
		issue.Unindent(`
      file { '/tmp/foo':
        mode => '0640',
        ensure => present;
      '/tmp/bar'
        mode => '0640',
        ensure => present;
      }`),
		`resource title expected (line: 4, column: 1)`)
}

func TestStatmentCallWithUnparameterizedHash(t *testing.T) {
	expectDump(t,
		`warning { message => 'syntax ok' }`,
		`(invoke {:functor (qn "warning") :args [(hash (=> (qn "message") "syntax ok"))]})`)
}

func TestNonStatmentCallWithUnparameterizedHash(t *testing.T) {
	expectError(t,
		`something { message => 'syntax ok' }`,
		`This expression is invalid. Did you try declaring a 'something' resource without a title? (line: 1, column: 1)`)
}

func TestResourceDefaults(t *testing.T) {
	expectDump(t,
		`Something { message => 'syntax ok' }`,
		`(resource-defaults {:type (qr "Something") :ops [(=> "message" "syntax ok")]})`)
}

func TestResourceDefaultsFromAccess(t *testing.T) {
	expectDump(t,
		`Resource[Something] { message => 'syntax ok' }`,
		`(resource-defaults {:type (access (qr "Resource") (qr "Something")) :ops [(=> "message" "syntax ok")]})`)

	expectDump(t,
		`@Resource[Something] { message => 'syntax ok' }`,
		`(resource-defaults {:type (access (qr "Resource") (qr "Something")) :ops [(=> "message" "syntax ok")] :form "virtual"})`)
}

func TestResourceOverride(t *testing.T) {
	expectDump(t,
		`File['/tmp/foo.txt'] { mode => '0644' }`,
		`(resource-override {:resources (access (qr "File") "/tmp/foo.txt") :ops [(=> "mode" "0644")]})`)

	expectDump(t,
		issue.Unindent(`
      Service['apache'] {
        require +> [File['apache.pem'], File['httpd.conf']]
      }`),
		`(resource-override {:resources (access (qr "Service") "apache") :ops [(+> "require" (array (access (qr "File") "apache.pem") (access (qr "File") "httpd.conf")))]})`)

	expectDump(t,
		`@File['/tmp/foo.txt'] { mode => '0644' }`,
		`(resource-override {:resources (access (qr "File") "/tmp/foo.txt") :ops [(=> "mode" "0644")] :form "virtual"})`)

}

func TestInvalidResource(t *testing.T) {
	expectError(t,
		`'File' { mode => '0644' }`,
		`invalid resource expression (line: 1, column: 1)`)
}

func TestVirtualResourceCollector(t *testing.T) {
	expectDump(t,
		`File <| |>`,
		`(collect {:type (qr "File") :query (virtual-query)})`)

	expectDump(t,
		issue.Unindent(`
      File <| mode == '0644' |>`),
		`(collect {:type (qr "File") :query (virtual-query (== (qn "mode") "0644"))})`)

	expectDump(t,
		issue.Unindent(`
      File <| mode == '0644' |> {
        owner => 'root',
        mode => 640
      }`),
		`(collect {:type (qr "File") :query (virtual-query (== (qn "mode") "0644")) :ops [(=> "owner" "root") (=> "mode" 640)]})`)
}

func TestExportedResourceCollector(t *testing.T) {
	expectDump(t,
		`File <<| |>>`,
		`(collect {:type (qr "File") :query (exported-query)})`)

	expectDump(t,
		issue.Unindent(`
      File <<| mode == '0644' |>>`),
		`(collect {:type (qr "File") :query (exported-query (== (qn "mode") "0644"))})`)

	expectDump(t,
		issue.Unindent(`
      File <<| mode == '0644' |>> {
        owner => 'root',
        mode => 640
      }`),
		`(collect {:type (qr "File") :query (exported-query (== (qn "mode") "0644")) :ops [(=> "owner" "root") (=> "mode" 640)]})`)
}

func TestOperators(t *testing.T) {
	expectDump(t,
		`$x = a or b and c < d == e << f + g * -h`,
		`(= (var "x") (or (qn "a") (and (qn "b") (< (qn "c") (== (qn "d") (<< (qn "e") (+ (qn "f") (* (qn "g") (- (qn "h"))))))))))`)

	expectDump(t,
		`$x = -h / g + f << e == d <= c and b or a`,
		`(= (var "x") (or (and (<= (== (<< (+ (/ (- (qn "h")) (qn "g")) (qn "f")) (qn "e")) (qn "d")) (qn "c")) (qn "b")) (qn "a")))`)

	expectDump(t,
		`$x = !a == b`,
		`(= (var "x") (== (! (qn "a")) (qn "b")))`)

	expectDump(t,
		`$x = a > b`,
		`(= (var "x") (> (qn "a") (qn "b")))`)

	expectDump(t,
		`$x = a >= b`,
		`(= (var "x") (>= (qn "a") (qn "b")))`)

	expectDump(t,
		`$x = a +b`,
		`(= (var "x") (+ (qn "a") (qn "b")))`)

	expectDump(t,
		`$x = +4`,
		`(= (var "x") 4)`)

	expectDump(t,
		`$x = 10 - 5 - 3`,
		`(= (var "x") (- (- 10 5) 3))`)

	expectDump(t,
		`$x = 10 - 5 * 3`,
		`(= (var "x") (- 10 (* 5 3)))`)

	expectDump(t,
		`$x = a * (b + c)`,
		`(= (var "x") (* (qn "a") (paren (+ (qn "b") (qn "c")))))`)

	expectDump(t,
		`$x = $y -= $z`,
		`(= (var "x") (-= (var "y") (var "z")))`)

	expectDump(t,
		`$x = $y + $z % 5`,
		`(= (var "x") (+ (var "y") (% (var "z") 5)))`)

	expectDump(t,
		`$x = $y += $z`,
		`(= (var "x") (+= (var "y") (var "z")))`)

	expectError(t,
		`$x = +b`,
		`unexpected token '+' (line: 1, column: 7)`)
}

func TestMatch(t *testing.T) {
	expectDump(t,
		`a =~ /^[a-z]+$/`,
		`(=~ (qn "a") (regexp "^[a-z]+$"))`)

	expectDump(t,
		`a !~ /^[a-z]+$/`,
		`(!~ (qn "a") (regexp "^[a-z]+$"))`)
}

func TestIn(t *testing.T) {
	expectDump(t,
		`'eat' in 'eaten'`,
		`(in "eat" "eaten")`)

	expectDump(t,
		`'eat' in ['eat', 'ate', 'eating']`,
		`(in "eat" (array "eat" "ate" "eating"))`)
}

func dump(e Expression) string {
	result := bytes.NewBufferString(``)
	e.ToPN().Format(result)
	return result.String()
}

func TestEPP(t *testing.T) {
	expectDumpEPP(t,
		``,
		`(lambda {:body [(epp (render-s ""))]})`)

	expectDumpEPP(t,
		issue.Unindent(`
      some arbitrary text
      spanning multiple lines`),
		`(lambda {:body [(epp (render-s "some arbitrary text\nspanning multiple lines"))]})`)

	expectDumpEPP(t,
		issue.Unindent(`
      <%||%> some arbitrary text
      spanning multiple lines`),
		`(lambda {:body [(epp (render-s " some arbitrary text\nspanning multiple lines"))]})`)

	expectDumpEPP(t,
		issue.Unindent(`
      <%||%> some <%#-%>text`),
		`(lambda {:body [(epp (render-s " some text"))]})`)

	expectErrorEPP(t,
		issue.Unindent(`
      <%||%> some <%#-text`),
		`unbalanced epp comment (line: 1, column: 13)`)

	expectDumpEPP(t,
		issue.Unindent(`
      <%||%> some <%%-%%-%%> text`),
		`(lambda {:body [(epp (render-s " some <%-%%-%> text"))]})`)

	expectDumpEPP(t,
		issue.Unindent(`
      <%||-%> some <-% %-> text`),
		`(lambda {:body [(epp (render-s "some <-% %-> text"))]})`)

	expectDumpEPP(t,
		issue.Unindent(`
      <%-||-%> some <%- $x = 3 %> text`),
		`(lambda {:body [(epp (render-s "some") (= (var "x") 3) (render-s " text"))]})`)

	expectErrorEPP(t,
		issue.Unindent(`
      <%-||-%> some <%- $x = 3 -% $y %> text`),
		`invalid operator '-%' (line: 1, column: 28)`)

	expectBlockEPP(t,
		issue.Unindent(`
      vcenter: {
        host: "<%= $host %>"
        user: "<%= $username %>"
        password: "<%= $password %>"
      }`),
		`(lambda {:body [(epp `+
			`(render-s "vcenter: {\n  host: \"") `+
			`(render (var "host")) `+
			`(render-s "\"\n  user: \"") `+
			`(render (var "username")) `+
			`(render-s "\"\n  password: \"") `+
			`(render (var "password")) `+
			`(render-s "\"\n}"))]})`)

	expectDumpEPP(t,
		issue.Unindent(`
      <%- | Boolean $keys_enable,
        String  $keys_file,
        Array   $keys_trusted,
        String  $keys_requestkey,
        String  $keys_controlkey
      | -%>
      <%# Parameter tag ↑ -%>

      <%# Non-printing tag ↓ -%>
      <% if $keys_enable { -%>

      <%# Expression-printing tag ↓ -%>
      keys <%= $keys_file %>
      <% unless $keys_trusted =~ Array[Data,0,0] { -%>
      trustedkey <%= $keys_trusted.join(' ') %>
      <% } -%>
      <% if $keys_requestkey =~ String[1] { -%>
      requestkey <%= $keys_requestkey %>
      <% } -%>
      <% if $keys_controlkey =~ String[1] { -%>
      controlkey <%= $keys_controlkey %>
      <% } -%>

      <% } -%>`),
		`(lambda {`+
			`:params {`+
			`:keys_enable {:type (qr "Boolean")} `+
			`:keys_file {:type (qr "String")} `+
			`:keys_trusted {:type (qr "Array")} `+
			`:keys_requestkey {:type (qr "String")} `+
			`:keys_controlkey {:type (qr "String")}} `+
			`:body [(epp `+
			`(render-s "\n\n\n") `+
			`(if {`+
			`:test (var "keys_enable") `+
			`:then [(render-s "\n\nkeys ") `+
			`(render (var "keys_file")) `+
			`(render-s "\n") `+
			`(unless {`+
			`:test (=~ (var "keys_trusted") (access (qr "Array") (qr "Data") 0 0)) `+
			`:then [`+
			`(render-s "trustedkey ") `+
			`(render (call-method {:functor (. (var "keys_trusted") (qn "join")) :args [" "]})) `+
			`(render-s "\n")]}) `+
			`(if {`+
			`:test (=~ (var "keys_requestkey") (access (qr "String") 1)) `+
			`:then [`+
			`(render-s "requestkey ") `+
			`(render (var "keys_requestkey")) `+
			`(render-s "\n")]}) `+
			`(if {`+
			`:test (=~ (var "keys_controlkey") (access (qr "String") 1)) `+
			`:then [`+
			`(render-s "controlkey ") `+
			`(render (var "keys_controlkey")) `+
			`(render-s "\n")]}) `+
			`(render-s "\n")]}))]})`)

	// Fail on EPP constructs unless EPP is enabled
	expectError(t,
		issue.Unindent(`
      <% $x = 3 %> text`),
		`unexpected token '<' (line: 1, column: 1)`)

	expectError(t,
		issue.Unindent(`
      $x = 3 %> 4`),
		`unexpected token '>' (line: 1, column: 9)`)

	expectError(t,
		issue.Unindent(`
      $x = 3 -%> 4`),
		`unexpected token '%' (line: 1, column: 9)`)

	expectErrorEPP(t,
		"\n<% |String $x| %>\n",
		`Ambiguous EPP parameter expression. Probably missing '<%-' before parameters to remove leading whitespace (line: 2, column: 5)`)
}

func expectDumpEPP(t *testing.T, source string, expected string) {
	expectDump(t, source, expected, EppMode)
}

func expectBlockEPP(t *testing.T, source string, expected string) {
	expectBlock(t, source, expected, EppMode)
}

func expectDump(t *testing.T, source string, expected string, parserOptions ...Option) {
	if expr := parseExpression(t, source, parserOptions...); expr != nil {
		actual := dump(expr)
		if expected != actual {
			t.Errorf("expected '%s', got '%s'", expected, actual)
		}
	}
}

func expectBlock(t *testing.T, source string, expected string, parserOptions ...Option) {
	expr, err := CreateParser(parserOptions...).Parse(``, source, false)
	if err != nil {
		t.Errorf(err.Error())
	} else {
		actual := dump(expr)
		if expected != actual {
			t.Errorf("expected '%s', got '%s'", expected, actual)
		}
	}
}

func expectErrorEPP(t *testing.T, source string, expected string) {
	expectError(t, source, expected, EppMode)
}

func expectError(t *testing.T, source string, expected string, parserOptions ...Option) {
	_, err := CreateParser(parserOptions...).Parse(``, source, false)
	if err == nil {
		t.Errorf("Expected error '%s' but nothing was raised", expected)
	} else {
		actual := err.Error()
		if expected != actual {
			t.Errorf("expected error '%s', got '%s'", expected, actual)
		}
	}
}

func expectHeredoc(t *testing.T, str string, args ...interface{}) {
	expected := args[0].(string)
	expr := parseExpression(t, str)
	if expr == nil {
		return
	}
	if heredoc, ok := expr.(*HeredocExpression); ok {
		if len(args) > 1 && heredoc.syntax != args[1] {
			t.Errorf("Expected syntax '%s', got '%s'", args[1], heredoc.syntax)
		}
		if textExpr, ok := heredoc.text.(*LiteralString); ok {
			if textExpr.value != expected {
				t.Errorf("Expected heredoc '%s', got '%s'", expected, textExpr.value)
			}
			return
		}
		actual := dump(expr)
		if actual != expected {
			t.Errorf("Expected heredoc '%s', got '%s'", expected, actual)
		}
		return
	}
	t.Errorf("'%s' did not result in a heredoc expression", str)
}

func parse(t *testing.T, str string, parserOptions ...Option) Expression {
	expr, err := CreateParser(parserOptions...).Parse(``, str, false)
	if err != nil {
		t.Errorf(err.Error())
		return nil
	}
	program, ok := expr.(*Program)
	if !ok {
		t.Errorf("'%s' did not parse to a program", str)
		return nil
	}
	return program.body
}

func parseExpression(t *testing.T, str string, parserOptions ...Option) Expression {
	expr := parse(t, str, parserOptions...)
	if block, ok := expr.(*BlockExpression); ok {
		if len(block.statements) == 1 {
			return block.statements[0]
		}
		t.Errorf("'%s' did not parse to a block with exactly one expression", str)
		return nil
	}
	return expr
}
