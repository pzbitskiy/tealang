package compiler

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"../stdlib"
)

func resolveModule(moduleName string, sourceDir string, currentDir string) (InputDesc, error) {
	// search for module
	var source string
	var sourceFile string
	if strings.HasPrefix(moduleName, stdlib.StdLibName) {
		var ok bool
		source, ok = stdlib.LoadModule(moduleName)
		if !ok {
			return InputDesc{}, fmt.Errorf("standard module %s not found", moduleName)
		}
		sourceFile = moduleName
		sourceDir = currentDir
	} else {
		components := strings.Split(moduleName, ".")
		locations := make([]string, 16)

		// search relative to source file first
		fullPath := path.Join(sourceDir, path.Join(components...))
		locations = append(locations, fullPath)
		locations = append(locations, fullPath+".tl")

		// search relative to current dir as a fallback
		fullPath = path.Join(currentDir, path.Join(components...))
		locations = append(locations, fullPath)
		locations = append(locations, fullPath+".tl")

		for _, loc := range locations {
			if fileExists(loc) {
				sourceFile = path.Base(fullPath)
				sourceDir = path.Dir(fullPath)
				srcBytes, err := ioutil.ReadFile(fullPath)
				if err != nil {
					return InputDesc{}, err
				}
				source = string(srcBytes)
			}
			break
		}

		if source == "" {
			return InputDesc{}, fmt.Errorf("module %s not found", moduleName)
		}
	}
	return InputDesc{source, sourceFile, sourceDir, currentDir}, nil
}
