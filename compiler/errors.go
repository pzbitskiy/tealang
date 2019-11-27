package compiler

import (
	"fmt"
	"strings"

	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type parserErrorType int

const (
	syntaxError    parserErrorType = 1
	ambiguityError parserErrorType = 2
	semanticError  parserErrorType = 3
)

type TypeError struct {
	msg string
}

// ParserError provides generic info about the error
type ParserError struct {
	errorType parserErrorType
	start     int
	end       int
	line      int
	column    int
	msg       string
	token     string
	excerpt   []string
}

type errorCollector struct {
	errors []ParserError
	source string
}

type tealBaseRecognitionException struct {
	message        string
	recognizer     antlr.Recognizer
	offendingToken antlr.Token
	offendingState int
	ctx            antlr.RuleContext
	input          antlr.IntStream
}

// copy of ANTLR's NewBaseRecognitionException
func newTealBaseRecognitionException(message string, parser antlr.Parser, token antlr.Token, rule antlr.RuleContext) *tealBaseRecognitionException {
	t := new(tealBaseRecognitionException)

	t.message = message
	t.recognizer = parser
	t.input = parser.GetInputStream()
	t.ctx = rule
	t.offendingToken = token
	t.offendingState = -1

	return t
}

func (e *tealBaseRecognitionException) GetOffendingToken() antlr.Token {
	return e.offendingToken
}

func (e *tealBaseRecognitionException) GetMessage() string {
	return e.message
}

func (e *tealBaseRecognitionException) GetInputStream() antlr.IntStream {
	return e.input
}

func newErrorCollector(source string) (er *errorCollector) {
	er = new(errorCollector)
	er.errors = make([]ParserError, 0, 16)
	er.source = source
	return
}

func (er *errorCollector) formatExcerpt(start, end int) []string {
	maxExcerptOffset := 50
	src := er.source
	excerptStart := strings.LastIndexByte(er.source[0:start], '\n') + 1
	if excerptStart == -1 {
		excerptStart = 0
	}
	diff := start - excerptStart
	if diff > maxExcerptOffset {
		excerptStart = start - maxExcerptOffset
	}
	excerptEnd := strings.IndexByte(src[end:], '\n')
	if excerptEnd == -1 {
		excerptEnd = len(src[end:])
	}
	diff = excerptEnd
	if diff > maxExcerptOffset {
		excerptEnd = end + maxExcerptOffset
	} else {
		excerptEnd = end + excerptEnd
	}

	trueEnd := end + 1
	if excerptEnd > trueEnd {
		trueEnd = excerptEnd
	}

	excerptStartFixed := excerptStart
	excerptEndFixed := excerptEnd
	trueEndFixed := trueEnd
	startFixed := start

	src = ""
	const tabSize = 4
	for _, ch := range er.source[excerptStart:trueEnd] {
		if ch == '\t' {
			src += strings.Repeat(" ", tabSize)
			// excerptStartFixed += tabSize
			excerptEndFixed += tabSize - 1 // tab symbol length + 3 rest replacing symbols
			trueEndFixed += tabSize - 1
			startFixed += tabSize - 1
		} else {
			src += string(ch)
		}
	}

	excerpt := make([]string, 2)
	excerpt[0] = src

	emphasizeLeftLength := 5
	emphasizeRightLength := 5
	spaces := startFixed - excerptStartFixed - emphasizeLeftLength
	if spaces < 0 {
		emphasizeLeftLength += spaces
		spaces = 0
	}

	excerpt[1] = fmt.Sprintf(
		"%s%s^%s",
		strings.Repeat(" ", spaces),
		strings.Repeat("-", emphasizeLeftLength),
		strings.Repeat("-", emphasizeRightLength),
	)
	return excerpt
}

func (er *errorCollector) filterAmbiguity() {
	var filtered []ParserError
	for _, err := range er.errors {
		if err.errorType != ambiguityError {
			filtered = append(filtered, err)
		}
	}
	er.errors = filtered
}

func (er *errorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	var start, end int
	var token string

	errorType := syntaxError
	cast := false
	if offendingSymbol != nil {
		if symbol, ok := offendingSymbol.(*antlr.CommonToken); ok {
			start = symbol.GetStart()
			end = symbol.GetStop()
			token = symbol.GetText()
			cast = true
		}
	}
	if e != nil {
		if e.GetOffendingToken() != nil && !cast {
			start = e.GetOffendingToken().GetStart()
			end = e.GetOffendingToken().GetStop()
			token = e.GetOffendingToken().GetText()
		}
		if e.GetMessage() != "" {
			errorType = semanticError
		}
	}

	info := ParserError{
		errorType,
		start,
		end,
		line,
		column,
		msg,
		token,
		er.formatExcerpt(start, end),
	}
	er.errors = append(er.errors, info)
}

func (er *errorCollector) ReportAmbiguity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, exact bool, ambigAlts *antlr.BitSet, configs antlr.ATNConfigSet) {
	info := ParserError{
		ambiguityError,
		startIndex,
		stopIndex,
		0,
		0,
		"",
		"",
		er.formatExcerpt(startIndex, stopIndex),
	}
	er.errors = append(er.errors, info)
}

func (er *errorCollector) ReportAttemptingFullContext(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex int, conflictingAlts *antlr.BitSet, configs antlr.ATNConfigSet) {
	recognizer.GetCurrentToken().GetLine()
	info := ParserError{
		ambiguityError,
		startIndex,
		stopIndex,
		recognizer.GetCurrentToken().GetLine(),
		recognizer.GetCurrentToken().GetColumn(),
		"",
		"",
		er.formatExcerpt(startIndex, stopIndex),
	}
	er.errors = append(er.errors, info)
}

func (er *errorCollector) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs antlr.ATNConfigSet) {
	info := ParserError{
		ambiguityError,
		startIndex,
		stopIndex,
		recognizer.GetCurrentToken().GetLine(),
		recognizer.GetCurrentToken().GetColumn(),
		"",
		"",
		er.formatExcerpt(startIndex, stopIndex),
	}
	er.errors = append(er.errors, info)
}

func (err *ParserError) String() (msg string) {
	switch err.errorType {
	case semanticError:
		msg = fmt.Sprintf("error at line %d, col %d near token \"%s\"", err.line, err.column, err.token)
		lines := append([]string{msg}, err.excerpt...)
		lines = append(lines, err.msg)
		msg = strings.Join(lines, "\n")
	case syntaxError:
		msg = fmt.Sprintf("syntax error at line %d, col %d near token \"%s\"", err.line, err.column, err.token)
		lines := append([]string{msg}, err.excerpt...)
		if err.token == "<EOF>" {
			lines = append(lines, "Missing logic function?")
		}
		msg = strings.Join(lines, "\n")
	case ambiguityError:
		msg = fmt.Sprintf("ambiguity error at offset %d", err.start)
		lines := append([]string{msg}, err.excerpt...)
		msg = strings.Join(lines, "\n")
	default:
		msg = fmt.Sprintf("unknown error at offset %d", err.start)
		lines := append([]string{msg}, err.excerpt...)
		msg = strings.Join(lines, "\n")
	}
	return msg
}
