package stdlib

import (
	"strings"
)

//go:generate bash ./bundle_stdlib_files.sh

// StdLibName constant name
const StdLibName string = "stdlib"

// lib contains source code for all library modules
var lib map[string]string

func init() {
	lib = make(map[string]string)
	lib["const"] = stdlib_const
	lib["templates"] = stdlib_templates
	lib["noop"] = stdlib_noop
}

// LoadModule returns source of a standard library, and a flag indicating success
func LoadModule(name string) (content string, ok bool) {
	prefix := StdLibName + "."
	if strings.HasPrefix(name, prefix) {
		modName := name[len(prefix):]
		content, ok = lib[modName]
	} else {
		content, ok = lib[name]
	}

	return content, ok
}
