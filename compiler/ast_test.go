package compiler

import (
	"testing"

	// "github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	source := `
let a = 456;
const b = "123";

function logic(txn, gtxn, account) {
	if e == 1 {
		let x = a + b;
		error
	}

	if a == 1 {
		return 0
	}

	return 1
}
`
	result, _ := Parse(source)
	require.NotEmpty(t, result)

	result.Print()
}
