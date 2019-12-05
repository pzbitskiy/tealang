package compiler

import (
	"bytes"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const prefixBase32 = "b32"
const prefixBase64 = "b64"
const prefixAddr = "addr"

var decoders = map[string]func(string, int, int) ([]byte, error){
	prefixBase32: b32String,
	prefixBase64: b64String,
	prefixAddr:   addrString,
}

// parseStringLiteral unquotes string and returns []byte
func parseStringLiteral(input string) (result []byte, err error) {
	start := 0
	end := len(input) - 1
	if input[start] != '"' {
		// check encoding prefixes
		for prefix, decoder := range decoders {
			if strings.HasPrefix(input, prefix) {
				start = len(prefix)
				return decoder(input, start+1, end)
			}
		}
	}

	if input[start] != '"' || input[end] != '"' {
		return nil, fmt.Errorf("no quotes")
	}

	return rawString(input, start+1, end)
}

func b32String(input string, start int, end int) (result []byte, err error) {
	return base32.StdEncoding.DecodeString(input[start:end])
}

func b64String(input string, start int, end int) (result []byte, err error) {
	return base64.StdEncoding.DecodeString(input[start:end])
}

func addrString(input string, start int, end int) (result []byte, err error) {
	const checksumLength = 4
	type digest [sha512.Size256]byte

	checksum := func(data digest) []byte {
		shortAddressHash := sha512.Sum512_256(data[:])
		checksum := shortAddressHash[len(shortAddressHash)-checksumLength:]
		return checksum
	}
	canonical := func(data digest) string {
		var addrWithChecksum []byte
		addrWithChecksum = append(data[:], checksum(data)...)
		return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(addrWithChecksum)
	}

	address := input[start:end]
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(address)
	if err != nil {
		return nil, fmt.Errorf("failed to decode address %s to base 32", address)
	}
	var short digest
	if len(decoded) < len(short) {
		return nil, fmt.Errorf("decoded bad addr: %s", address)
	}

	copy(short[:], decoded[:len(short)])
	incomingchecksum := decoded[len(decoded)-checksumLength:]

	calculatedchecksum := checksum(short)
	isValid := bytes.Equal(incomingchecksum, calculatedchecksum)

	if !isValid {
		return nil, fmt.Errorf("address %s is malformed, checksum verification failed", address)
	}

	// Validate that we had a canonical string representation
	if canonical(short) != address {
		return nil, fmt.Errorf("address %s is non-canonical", address)
	}

	return short[:], nil
}

func rawString(input string, start int, end int) (result []byte, err error) {
	escapeSeq := false
	hexSeq := false
	result = make([]byte, 0, end-start+1)

	// skip first and last quotes
	pos := start
	for pos < end {
		char := input[pos]
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
