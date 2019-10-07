package main

import (
	"fmt"
	"strconv"
)

// parseStringLiteral unquotes string and returns []byte
func parseStringLiteral(input string) (result []byte, err error) {
	if input[0] != '"' || input[len(input)-1] != '"' {
		err = fmt.Errorf("no quotes")
		return
	}

	var char byte
	escapeSeq := false
	hexSeq := false
	result = make([]byte, 0, len(input)-2)

	// skip first and last quotes
	pos := 1
	for pos < len(input)-1 {
		char = input[pos]
		if char == '\\' && !escapeSeq {
			if hexSeq {
				return nil, fmt.Errorf("escape seq inside hex number")
			}
			escapeSeq = true
			pos++
			continue
		}
		if escapeSeq {
			escapeSeq = false
			switch char {
			case 'n':
				char = '\n'
			case 'r':
				char = '\r'
			case 't':
				char = '\t'
			case '\\':
				char = '\\'
			case '"':
				char = '"'
			case 'x':
				hexSeq = true
				pos++
				continue
			default:
				return nil, fmt.Errorf("invalid escape seq \\%c", char)
			}
		}
		if hexSeq {
			hexSeq = false
			if pos >= len(input)-2 { // count a closing quote
				return nil, fmt.Errorf("non-terminated hex seq")
			}
			num, err := strconv.ParseUint(input[pos:pos+2], 16, 8)
			if err != nil {
				return nil, err
			}
			char = uint8(num)
			pos++
		}

		result = append(result, char)
		pos++
	}
	if escapeSeq || hexSeq {
		return nil, fmt.Errorf("non-terminated escape seq")
	}

	return
}
