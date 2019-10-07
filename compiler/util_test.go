package compiler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringDecoding(t *testing.T) {
	s := `"test"`
	e := []byte(`test`)
	result, err := parseStringLiteral(s)
	require.NoError(t, err)
	require.Equal(t, e, result)

	s = `"test\n"`
	e = []byte(`test
`)
	result, err = parseStringLiteral(s)
	require.NoError(t, err)
	require.Equal(t, e, result)

	s = `"test\x0a"`
	e = []byte(`test
`)
	result, err = parseStringLiteral(s)
	require.NoError(t, err)
	require.Equal(t, e, result)

	s = `"test\n\t\""`
	e = []byte(`test
	"`)
	result, err = parseStringLiteral(s)
	require.NoError(t, err)
	require.Equal(t, e, result)

	s = `"\x74\x65\x73\x74\x31\x32\x33"`
	e = []byte(`test123`)
	result, err = parseStringLiteral(s)
	require.NoError(t, err)
	require.Equal(t, e, result)

	s = `"test`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "no quotes")

	s = `test`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "no quotes")

	s = `test"`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "no quotes")

	s = `"test\"`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "non-terminated escape seq")

	s = `"test\x\"`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "escape seq inside hex number")

	s = `"test\a"`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "invalid escape seq \\a")

	s = `"test\x10\x1"`
	result, err = parseStringLiteral(s)
	require.EqualError(t, err, "non-terminated hex seq")
}
