package compiler

import "fmt"

func opTypeFromSpec(name string, ret int) (exprType, error) {
	if op, ok := langOps[name]; ok && len(op.Returns) != 0 {
		switch op.Returns[ret] {
		case 'U':
			return intType, nil
		case 'B':
			return bytesType, nil
		case '.':
			return unknownType, nil
		}
	}
	return invalidType, fmt.Errorf("can't get type for %s ret #%d", name, ret+1)
}

func argOpTypeFromSpec(name string, arg int) (exprType, error) {
	if op, ok := langOps[name]; ok && len(op.Args) > arg {
		switch op.Args[arg] {
		case 'U':
			return intType, nil
		case 'B':
			return bytesType, nil
		case '.':
			return unknownType, nil
		}
	}
	return invalidType, fmt.Errorf("can't get type for %s arg #%d", name, arg+1)
}

func runtimeFieldTypeFromSpec(name string, field string) (exprType, error) {
	if op, ok := langOps[name]; ok && len(op.ArgEnum) != 0 {
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
	} else {
		// gtxns does not have ArgEnum
		return opTypeFromSpec(name, 0)
	}
	return invalidType, fmt.Errorf("can't get type for %s.%s", name, field)
}
