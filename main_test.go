package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMainBasic(t *testing.T) {
	setRootCmdFlags()

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	rootCmd.SetArgs([]string{"-c", "examples/basic.tl", "-s"})

	err := rootCmd.Execute()
	require.NoError(t, err)
	w.Close()

	out, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	require.Contains(t, string(out), "end_main")
}
