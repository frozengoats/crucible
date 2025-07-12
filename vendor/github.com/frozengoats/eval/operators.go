package eval

import (
	"fmt"
	"math"
)

func EqualsOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT == bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for == comparison", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return aT == bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for == comparison", a, b)
		}
	case bool:
		switch bT := b.(type) {
		case bool:
			return aT == bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for == comparison", a, b)
		}

	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for == comparison", a, b)
	}
}

func UnequalsOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT != bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for != comparison", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return aT != bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for != comparison", a, b)
		}
	case bool:
		switch bT := b.(type) {
		case bool:
			return aT != bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for != comparison", a, b)
		}

	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for != comparison", a, b)
	}
}

func GreaterThanOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT > bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for > comparison", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return aT > bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for > comparison", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for > comparison", a, b)
	}
}

func GreaterThanEqualsOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT >= bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for >= comparison", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return aT >= bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for >= comparison", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for >= comparison", a, b)
	}
}

func LessThanOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT < bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for < comparison", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return aT < bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for < comparison", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for < comparison", a, b)
	}
}

func LessThanEqualsOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT <= bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for <= comparison", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return aT <= bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for <= comparison", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for <= comparison", a, b)
	}
}

func AndOp(a any, b any) (any, error) {
	var aIsTrue bool
	var bIsTrue bool
	switch aT := a.(type) {
	case string:
		aIsTrue = len(aT) > 0
	case float64:
		aIsTrue = aT != 0
	case bool:
		aIsTrue = a == true
	case []any:
		aIsTrue = len(aT) > 0
	case map[string]any:
		aIsTrue = len(aT) > 0
	default:
		aIsTrue = false
	}

	switch bT := b.(type) {
	case string:
		bIsTrue = len(bT) > 0
	case float64:
		bIsTrue = bT != 0
	case bool:
		bIsTrue = b == true
	case []any:
		aIsTrue = len(bT) > 0
	case map[string]any:
		aIsTrue = len(bT) > 0
	default:
		bIsTrue = false
	}

	if aIsTrue && bIsTrue {
		return b, nil
	}

	if !aIsTrue {
		return a, nil
	}

	return b, nil
}

func OrOp(a any, b any) (any, error) {
	var aIsTrue bool
	var bIsTrue bool
	switch aT := a.(type) {
	case string:
		aIsTrue = len(aT) > 0
	case float64:
		aIsTrue = aT != 0
	case bool:
		aIsTrue = a == true
	case []any:
		aIsTrue = len(aT) > 0
	case map[string]any:
		aIsTrue = len(aT) > 0
	default:
		aIsTrue = false
	}

	switch bT := b.(type) {
	case string:
		bIsTrue = len(bT) > 0
	case float64:
		bIsTrue = bT != 0
	case bool:
		bIsTrue = b == true
	case []any:
		aIsTrue = len(bT) > 0
	case map[string]any:
		aIsTrue = len(bT) > 0
	default:
		bIsTrue = false
	}

	if aIsTrue {
		return a, nil
	}

	if bIsTrue {
		return b, nil
	}

	return b, nil
}

func PlusOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case string:
		switch bT := b.(type) {
		case string:
			return aT + bT, nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for addition/concatenation", a, b)
		}
	case float64:
		switch bT := b.(type) {
		case float64:
			return float64(aT + bT), nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for addition/concatenation", a, b)
		}
	case []any:
		switch bT := b.(type) {
		case []any:
			return append(aT, bT...), nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for addition/concatenation", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for addition/concatenation", a, b)
	}
}

func MinusOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case float64:
		switch bT := b.(type) {
		case float64:
			return float64(aT - bT), nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for subtraction", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for subtraction", a, b)
	}
}

func MultiplyOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case float64:
		switch bT := b.(type) {
		case float64:
			return float64(aT * bT), nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for multiplication", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for multiplication", a, b)
	}
}

func ExponentOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case float64:
		switch bT := b.(type) {
		case float64:
			return math.Pow(aT, bT), nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for multiplication", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for multiplication", a, b)
	}
}

func DivideOp(a any, b any) (any, error) {
	switch aT := a.(type) {
	case float64:
		switch bT := b.(type) {
		case float64:
			if bT == 0 {
				return nil, fmt.Errorf("division by zero error")
			}
			return float64(aT / bT), nil
		default:
			return nil, fmt.Errorf("%v and %v are incompatible types for division", a, b)
		}
	default:
		return nil, fmt.Errorf("%v and %v are incompatible types for division", a, b)
	}
}
