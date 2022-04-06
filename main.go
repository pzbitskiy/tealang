package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/algorand/go-algorand/data/transactions/logic"
	"github.com/spf13/cobra"

	"github.com/pzbitskiy/tealang/compiler"
	dr "github.com/pzbitskiy/tealang/dryrun"
)

var outFile string
var inFile string
var source string
var compileOnly bool
var verbose bool
var oneliner string
var stdout bool
var raw bool
var dryrun string

var currentDir string
var sourceDir string
var sourceFile string

const defaultInFile string = "a.tl"

var rootCmd = &cobra.Command{
	Use:   "tealang [flags] source-file",
	Short: "Tealang compiler for Algorand Smart Contract (ASC1)",
	Long: `Tealang compiler (ASC1)
Documentation: https://github.com/pzbitskiy/tealang
Syntax highlighter for vscode: https://github.com/pzbitskiy/tealang-syntax-highlighter`,
	DisableFlagsInUseLine: true,
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if len(oneliner) > 0 {
			source = oneliner
			inFile = defaultInFile
			return nil
		}

		if len(args) < 1 {
			return errors.New("requires a source file name or - for stdin")
		}
		inFile = args[0]
		if inFile == "-" {
			data, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			source = string(data)
			inFile = defaultInFile
			return nil
		}

		currentDir, err := os.Getwd()
		if err != nil {
			return err
		}

		fullPath := path.Join(currentDir, inFile)
		srcBytes, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return err
		}
		sourceDir = path.Dir(fullPath)
		sourceFile = path.Base(fullPath)

		source = string(srcBytes)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("stdout") && cmd.Flags().Changed("output") {
			fmt.Printf("only one of [--stdout] or [--output] can be specified")
			os.Exit(1)
		}
		if cmd.Flags().Changed("raw") && !cmd.Flags().Changed("raw") {
			fmt.Printf("[--raw] might be only used with [--stdout]")
			os.Exit(1)
		}

		var prog compiler.TreeNodeIf
		var parseErrors []compiler.ParserError
		var teal string
		var bytecode []byte
		var err error
		var op *logic.OpStream
		if len(oneliner) > 0 {
			prog, parseErrors = compiler.ParseOneLineCond(source)
			if len(parseErrors) > 0 {
				for _, e := range parseErrors {
					fmt.Printf("%s\n", e.String())
				}
				os.Exit(1)
			}
		} else {
			input := compiler.InputDesc{
				Source:     source,
				SourceFile: sourceFile,
				SourceDir:  sourceDir,
				CurrentDir: currentDir,
			}
			prog, parseErrors = compiler.ParseProgram(input)
			if len(parseErrors) > 0 {
				for _, e := range parseErrors {
					fmt.Printf("%s\n", e.String())
				}
				os.Exit(1)
			}
		}
		teal = compiler.Codegen(prog)

		if !compileOnly {
			op, err = logic.AssembleString(teal)
			if err != nil {
				for _, err := range op.Errors {
					fmt.Println(err)
				}
				fmt.Println(err.Error())
				os.Exit(1)
			}
			bytecode = op.Program
		}

		if stdout {
			output := teal
			if bytecode != nil {
				if raw {
					output = string(bytecode)
				} else {
					output = hex.Dump(bytecode)
				}
			}
			fmt.Print(output)
		} else {
			ext := path.Ext(inFile)
			if outFile == "" {
				if compileOnly {
					outFile = inFile[0:len(inFile)-len(ext)] + ".teal"
				} else {
					outFile = inFile[0:len(inFile)-len(ext)] + ".tok"
				}
			}
			if verbose {
				fmt.Printf("Writing result to %s\n", outFile)
			}

			output := []byte(teal)
			if bytecode != nil {
				output = bytecode
			}
			ioutil.WriteFile(outFile, output, 0644)
		}

		if cmd.Flags().Changed("dryrun") {
			if bytecode == nil {
				op, err = logic.AssembleString(teal)
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
				bytecode = op.Program
			}
			sb := strings.Builder{}
			pass, err := dr.Run(bytecode, dryrun, &sb)
			fmt.Printf("trace:\n%s\n", sb.String())
			if pass {
				fmt.Printf(" - pass -\n")
			} else {
				fmt.Printf("REJECT\n")
			}
			if err != nil {
				fmt.Printf("ERROR: %s\n", err.Error())
			}

		}
	},
}

func setRootCmdFlags() {
	rootCmd.Flags().StringVarP(&outFile, "output", "o", "", "write output to this file")
	rootCmd.Flags().BoolVarP(&compileOnly, "compile", "c", false, "compile to TEAL assembler, do not produce bytecode")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().StringVarP(&oneliner, "oneliner", "l", "", "compile logic one-liner like '(txn.Sender == \"abc\") && (1+2) >= 3'")
	rootCmd.Flags().BoolVarP(&stdout, "stdout", "s", false, "write output to stdout instead of a file")
	rootCmd.Flags().BoolVarP(&raw, "raw", "r", false, "do not hex-encode bytecode when outputting to stdout")
	rootCmd.Flags().StringVarP(&dryrun, "dryrun", "d", "", "dry run program with transaction data from the file provided")
}

func main() {
	setRootCmdFlags()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
