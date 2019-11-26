#!/usr/bin/env bash

THISDIR=$(dirname $0)

cat <<EOM | gofmt > $THISDIR/langspec_gen.go
// Code generated during build process, along with langspec.json. DO NOT EDIT.
package compiler

import "encoding/json"

var langSpecJson []byte

type spec struct {
	EvalMaxVersion  int
	LogicSigVersion int
	Ops             []operation
}

type operation struct {
	Opcode        int
	Name          string
	Cost          int
	Size          int
	Args		  string
	Returns       string
	ArgEnum       []string
	ArgEnumTypes  string
	Doc           string
	ImmediateNote string
	Group         []string
}

var langSpec spec
var langOps map[string]operation

func init() {
	langSpecJson = []byte{
        $(cat $THISDIR/langspec.json | hexdump -v -e '1/1 "0x%02X, "' | fmt)
	}

	err := json.Unmarshal(langSpecJson, &langSpec)
	if err != nil {
		panic("can't load TEAL spec")
	}

	langOps = make(map[string]operation)
	for _, op := range(langSpec.Ops) {
		langOps[op.Name] = op
	}
}

EOM