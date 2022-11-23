package record3

import (
	"discovery/errors"
	"discovery/fmt"

	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

/* XXX: lexer regex token 스트링 순서 유지
 */
var RcmdLexer = lexer.Must(lexer.Regexp(`(?m)` +
	`(\s+)` +
	`|(^[#].*$)` +
	`|(?P<COMMENT>;.*$)` +
	`|(?P<RCMD>(?i)\b(BASHSETENV|BP|CHECK|CLOSE|CONNECT|DEBUG|DEFER|ENVIRONMENT|EOL|ERROR|EXPECT|FOR|GET|IF|LOAD|LOG|PUT|REQUIRE|SCRIPT|SEND|SET|SETA|SLEEP|SPAWN|TABLE|UNLOAD|UNSET|VERSION)\b)` +
	`|(?P<KEYWORD>(?i)\b(CR|LF|CRLF|INI|RANGE|ON|OFF|CSV|ROW|IN|TRUE|FALSE|NULL|NIL|NONE|AND|OR|NOT|ELSEIF|ELSE|ENDIF|ENDDEFER|ENDFOR|ENDTABLE|BREAK|CONTINUE|RETURN|STEP|BOTH_VARIABLE_NAME|IGNORE_SECTION_NAME|COMPAT_INI|LOGIN|LOGOUT|RFC2544|NORMAL|REQ)\b)` +
	`|(?P<FUNCTION>\b(len|num|str|exist|expr|split|join|trim|filter|type|append|isdefined)\b)` +
	`|(?P<IDENT>[a-zA-Z_][a-zA-Z0-9_:]*)` +
	`|(?P<OPERATORS>[-+*/%,.()=<>!~:;])` +
	`|(?P<NUMBER>\d+(\.\d+)?)` +
	"|(?P<STRING>'(?:[^'\\\\]|\\\\.)*'|\"(?:[^\"\\\\]|\\\\.)*\"|`(?:[^`\\\\]|\\\\.)*`)" +
	`|(?P<BRACKET>[\[\]\{\}])`,
))

func RcmdParser(recordText string, grammar interface{}, target interface{}) *errors.Error {
	parser, oserr := participle.Build(
		grammar,
		participle.Lexer(RcmdLexer),
		participle.UseLookahead(10))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	oserr = parser.ParseString(recordText, target)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func NewStruct(text string, grammar Void) (Void, *errors.Error) {
	if len(text) == 0 || grammar == nil {
		return nil, errors.New("invalid arguments")
	}

	target := grammar
	err := RcmdParser(text, grammar, target)
	if err != nil {
		return nil, err
	}

	return target, nil
}
