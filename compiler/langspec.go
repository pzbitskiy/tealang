package compiler

import "fmt"

func opTypeFromSpec(op string, ret int) (exprType, error) {
	if op, ok := langOps[op]; ok && len(op.Returns) != 0 {
		switch op.Returns[ret] {
		case 'U':
			return intType, nil
		case 'B':
			return bytesType, nil
		case '.':
			return unknownType, nil
		}
	}
	return invalidType, fmt.Errorf("can't get type for %s ret #%d", op, ret+1)
}

func argOpTypeFromSpec(op string, arg int) (exprType, error) {
	if op, ok := langOps[op]; ok && len(op.Args) > arg {
		switch op.Args[arg] {
		case 'U':
			return intType, nil
		case 'B':
			return bytesType, nil
		case '.':
			return unknownType, nil
		}
	}
	return invalidType, fmt.Errorf("can't get type for %s arg #%d", op, arg+1)
}

func runtimeFieldTypeFromSpec(op string, field string) (exprType, error) {
	if op, ok := langOps[op]; ok && len(op.ArgEnum) != 0 {
		for idx, entry := range op.ArgEnum {
			if entry == field {
				switch op.ArgEnumTypes[idx] {
				case 'U':
					return intType, nil
				case 'B':
					return bytesType, nil
				default:
					break
				}
			}
		}
	}
	return invalidType, fmt.Errorf("can't get type for %s.%s", op, field)
}
