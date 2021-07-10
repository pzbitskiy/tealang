package compiler

import (
	"fmt"
)

var builtinFun = map[string]bool{
	"sha256":            true,
	"keccak256":         true,
	"sha512_256":        true,
	"ed25519verify":     true,
	"len":               true,
	"itob":              true,
	"btoi":              true,
	"concat":            true,
	"substring":         true,
	"substring3":        false, // not a tealang builtin but TEAL func
	"mulw":              true,
	"addw":              true,
	"expw":              true,
	"divmodw":           true,
	"exp":               true,
	"balance":           true,
	"min_balance":       true,
	"app_opted_in":      true,
	"app_local_get":     true,
	"app_local_get_ex":  true,
	"app_global_get":    true,
	"app_global_get_ex": true,
	"app_local_put":     true, // accounts[x].put
	"app_global_put":    true, // apps[0].put
	"app_local_del":     true, // accounts[x].del
	"app_global_del":    true, // apps[0].del
	"asset_holding_get": true,
	"asset_params_get":  true,
	"assert":            true,
	"getbit":            true,
	"getbyte":           true,
	"setbit":            true,
	"setbyte":           true,
	"shl":               true,
	"shr":               true,
	"sqrt":              true,
	"bitlen":            true,
	"bzero":             true,
	"badd":              true,
	"bsub":              true,
	"bdiv":              true,
	"bmul":              true,
	"blt":               true,
	"bgt":               true,
	"ble":               true,
	"bge":               true,
	"beq":               true,
	"bne":               true,
	"bmod":              true,
	"bor":               true,
	"band":              true,
	"bxor":              true,
	"bnot":              true,
}

var builtinFunDependantTypes = map[string]int{
	"setbit": 0, // op type matches to first arg type
}

type remapper func(*funCallNode) (int, error)

var builtinFunRemap = map[string]remapper{
	"substring": remapSubsring,
	"badd":      makeByteArithRemapper("b+"),
	"bsub":      makeByteArithRemapper("b-"),
	"bdiv":      makeByteArithRemapper("b/"),
	"bmul":      makeByteArithRemapper("b*"),
	"blt":       makeByteArithRemapper("b<"),
	"bgt":       makeByteArithRemapper("b>"),
	"ble":       makeByteArithRemapper("b<="),
	"bge":       makeByteArithRemapper("b>="),
	"beq":       makeByteArithRemapper("b=="),
	"bne":       makeByteArithRemapper("b!="),
	"bmod":      makeByteArithRemapper("b%"),
	"bor":       makeByteArithRemapper("b|"),
	"band":      makeByteArithRemapper("b&"),
	"bxor":      makeByteArithRemapper("b^"),
	"bnot":      makeByteArithRemapper("b!"),
}

func makeByteArithRemapper(name string) remapper {
	// save remapped name for later codegen
	builtinFun[name] = false

	remapper := func(exprNode *funCallNode) (argErrorPos int, err error) {
		exprNode.name = name
		return 0, nil
	}
	return remapper
}

func remapSubsring(exprNode *funCallNode) (argErrorPos int, err error) {
	var arg1Val, arg2Val string
	switch arg1 := exprNode.childrenNodes[1].(type) {
	case *constNode:
		if arg1.exprType != intType {
			argErrorPos = 1
			err = fmt.Errorf("arg #1 must be int")
			return
		} else {
			arg1Val = arg1.value
		}
	case *exprLiteralNode:
		if arg1.exprType != intType {
			argErrorPos = 1
			err = fmt.Errorf("arg #1 must be int")
			return
		} else {
			arg1Val = arg1.value
		}
	}
	switch arg2 := exprNode.childrenNodes[2].(type) {
	case *constNode:
		if arg2.exprType != intType {
			argErrorPos = 2
			err = fmt.Errorf("arg #2 must be int")
			return
		} else {
			arg2Val = arg2.value
		}
	case *exprLiteralNode:
		if arg2.exprType != intType {
			argErrorPos = 2
			err = fmt.Errorf("arg #2 must be int")
			return
		} else {
			arg2Val = arg2.value
		}
	}
	if len(arg1Val) > 0 && len(arg2Val) > 0 {
		exprNode.childrenNodes = exprNode.childrenNodes[:1]
		exprNode.index1 = arg1Val
		exprNode.index2 = arg2Val
	} else {
		exprNode.name = "substring3"
	}
	return
}
