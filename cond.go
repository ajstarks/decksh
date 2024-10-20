// Package decksh is a little language that generates deck markup
// conditional processing
package decksh

import (
	"fmt"
	"strconv"
)

const relfmt = `line %d: %v

if v1 relation v2

where relation is:
== or eq   equals                 if x == y
!= or ne   not equals             if x != y
<  or lt   less than              if x > y
>  or gt   greater than           if x < y
>= or ge   greater than or equal  if x >= y
<= or ge   less than or equal     if x <= y
>< or bt   between                if x >< y z
`

const quote = '"'

// isString tests of a string is bounded by quotes
func isString(s string) bool {
	l := len(s)
	return l > 1 && s[0] == quote && s[l-1] == quote
}

// condition evaluates string and number conditional
func condition(s []string, linenumber int) (bool, error) {
	var left, right, upper float64
	var err error

	l := len(s)
	if !(l == 4 || l == 5) {
		return false, fmt.Errorf(relfmt, linenumber, s)
	}
	relation := s[2]

	// evaluate the arguments for the condition
	// if x [rel] y
	if l >= 4 {
		sleft := eval(s[1])
		sright := eval(s[3])

		// string conditionals
		if isString(sleft) && isString(sright) {
			switch relation {
			case "eq", "==":
				return sleft == sright, nil
			case "neq", "!=":
				return sleft != sright, nil
			default:
				return false, fmt.Errorf(relfmt, linenumber, s)
			}
		}

		// numberic conditionals
		left, err = strconv.ParseFloat(sleft, 64)
		if err != nil {
			return false, err
		}
		right, err = strconv.ParseFloat(sright, 64)
		if err != nil {
			return false, err
		}
	}
	// get the last argument for the between condition
	// if x >< y z  // if x is between y and z
	if l == 5 {
		upper, err = strconv.ParseFloat(eval(s[4]), 64)
		if err != nil {
			return false, err
		}
	}
	// return result of the condition
	switch relation {
	case "eq", "==": // equal
		return left == right, nil
	case "ne", "!=": // not equal
		return left != right, nil
	case "le", "<=": // less than or equal to
		return left <= right, nil
	case "ge", ">=": // greater than or equal to
		return left >= right, nil
	case "lt", "<": // less than
		return left < right, nil
	case "gt", ">": // greater than
		return left > right, nil
	case "bt", "><": // between
		return left >= right && left <= upper, nil
	default:
		return false, fmt.Errorf(relfmt, linenumber, s)
	}
}
