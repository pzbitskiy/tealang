package compiler

import (
	"bytes"
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

func TestStringEncodingPrefixes(t *testing.T) {
	a := require.New(t)

	s := `b64"MTIz"`
	e := []byte(`123`)
	result, err := parseStringLiteral(s)
	a.NoError(err)
	a.Equal(e, result)

	s = `b64"MTIzCg=="`
	e = []byte("123\n")
	result, err = parseStringLiteral(s)
	a.NoError(err)
	a.Equal(e, result)

	s = `b64"123"`
	result, err = parseStringLiteral(s)
	a.Error(err)

	s = `b32"GEZDGCQ="`
	e = []byte("123\n")
	result, err = parseStringLiteral(s)
	a.NoError(err)
	a.Equal(e, result)

	s = `b32"GEZDG==="`
	e = []byte("123")
	result, err = parseStringLiteral(s)
	a.NoError(err)
	a.Equal(e, result)

	s = `b32"123"`
	result, err = parseStringLiteral(s)
	a.Error(err)

	s = `addr"J5YDZLPOHWB5O6MVRHNFGY4JXIQAYYM6NUJWPBSYBBIXH5ENQ4Z5LTJELU"`
	result, err = parseStringLiteral(s)
	a.NoError(err)

	s = `addr"J5YDZLPOHWB5O6MVRHNFGY4JXIQAYYM6NUJWPBSYBBIXH5ENQ4Z5LTJELY"`
	result, err = parseStringLiteral(s)
	a.Error(err)

	s = `addr"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAY5HFKQ"`
	e = bytes.Repeat([]byte("\x00"), 32)
	result, err = parseStringLiteral(s)
	a.NoError(err)
	a.Equal(e, result)
}
