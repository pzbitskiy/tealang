package compiler

import "fmt"

func typeFromSpec(op string) (exprType, error) {
	if op, ok := langOps[op]; ok && len(op.Returns) != 0 {
		switch op.Returns[0] {
		case 'U':
			return intType, nil
		case 'B':
			return bytesType, nil
		}
	}
	return invalidType, fmt.Errorf("can't get type for %s", op)
}
