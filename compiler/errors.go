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
)

// ParserError provides generic info about the error
type ParserError struct {
	errorType parserErrorType
	start     int
	end       int
	line      int
	column    int
	msg       string
	token     string
	excerpt   string
}

type errorCollector struct {
	errors []ParserError
	source string
}

func newErrorCollector(source string) (er *errorCollector) {
	er = new(errorCollector)
	er.errors = make([]ParserError, 0, 16)
	er.source = source
	return
}

func (er *errorCollector) formatExcerpt(start, end int) string {
	maxExcerptOffset := 20
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

	excerpt := fmt.Sprintf("%s ==> %s <== ", er.source[excerptStart:start], er.source[start:end+1])
	if excerptEnd > end+1 {
		excerpt = fmt.Sprintf("%s%s", excerpt, er.source[end+1:excerptEnd])
	}

	return excerpt
}

func (er *errorCollector) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	start := e.GetOffendingToken().GetStart()
	end := e.GetOffendingToken().GetStop()

	info := ParserError{
		syntaxError,
		start,
		end,
		line,
		column,
		msg,
		e.GetOffendingToken().GetText(),
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

func (er *errorCollector) ReportContextSensitivity(recognizer antlr.Parser, dfa *antlr.DFA, startIndex, stopIndex, prediction int, configs antlr.ATNConfigSet) {
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

func (err *ParserError) String() (msg string) {
	switch err.errorType {
	case syntaxError:
		msg = fmt.Sprintf("syntax error at token \"%s\" at line %d, col %d: %s", err.token, err.line, err.column, err.excerpt)
	case ambiguityError:
		msg = fmt.Sprintf("ambiguity error at offset %d: %s", err.start, err.excerpt)
	default:
		msg = fmt.Sprintf("unknown error at offset %d: %s", err.start, err.excerpt)
	}
	return msg
}
