// Package decksh is a little language that generates deck markup
// assignments
package decksh

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
)

// ftoa returns the string value of a floating point value
func ftoa(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// assign creates an assignment by filling in the global id map
// assignments are either
// simple (x=10),
// binary op (x=a+b),
// coordinate (p=(100,50))
// or built-ins (random, polar, polarx, polary, vmap, sprint, substr, format, sqrt)
func assign(s []string, linenumber int) error {
	ls := len(s)
	if ls < 3 {
		return fmt.Errorf("line %d: %v is an incorrect assignment", linenumber, s)
	}
	if ls == 3 {
		return simpleassign(s, linenumber) // v=10
	}
	switch s[2] {
	case "area":
		return areafunc(s, linenumber) // v=area x, v=area a+b
	case "sqrt", "sine", "cosine", "tangent":
		return mathfunc(s, linenumber)
	case "random":
		return random(s, linenumber) // x=random min max
	case "sprint", "format":
		return sprint(s, linenumber) // v=format fmt x, v=format fmt a+b
	case "substr", "slice":
		return substr(s, linenumber) // x=substr s begin end
	case "polar", "polarx", "polary":
		return polarfunc(s, linenumber) // x=polar[x|y] cx cy r theta
	case "vmap":
		return vmapfunc(s, linenumber) // x=vmap d min1 max1 min2 max2
	case "(":
		return coordfunc(s, linenumber) // p=(100,100), p=(a,a+b), p=(a+b,b), p=(a+b,a+c)
	default:
		return binop(s, linenumber) // v=a+b
	}
}

// simpleassign creates an simple assignment id=number
func simpleassign(s []string, linenumber int) error {
	if len(s) != 3 {
		return fmt.Errorf("line %d: use: id=number or other id", linenumber)
	}
	emap[s[0]] = s[2]
	return nil
}

// opval returns the value of a binary operation
func opval(s []string, linenumber int) (float64, error) {
	es := fmt.Errorf("line %d: %v is not a valid operation", linenumber, s)
	if len(s) != 3 {
		return 0, es
	}
	ls := s[0]
	op := s[1]
	rs := s[2]
	lv, err := strconv.ParseFloat(eval(ls), 64)
	if err != nil {
		return 0, fmt.Errorf("line %d: %v is not a number", linenumber, ls)
	}
	rv, err := strconv.ParseFloat(eval(rs), 64)
	if err != nil {
		return 0, fmt.Errorf("line %d: %v is not a number", linenumber, rs)
	}
	var v float64
	switch op {
	case "+":
		v = lv + rv
	case "-":
		v = lv - rv
	case "*":
		v = lv * rv
	case "%":
		if rv == 0 {
			return 0, fmt.Errorf("line %d: you cannot modulo zero (%v modulo %v)", linenumber, lv, rv)
		}
		v = math.Mod(lv, rv)
	case "/":
		if rv == 0 {
			return 0, fmt.Errorf("line %d: you cannot divide by zero (%v / %v)", linenumber, lv, rv)
		}
		v = lv / rv
	default:
		return 0, fmt.Errorf("line %d: %s is not a valid operation", linenumber, op)
	}
	return v, nil
}

// binop processes a binary expression: id=id op number
func binop(s []string, linenumber int) error {
	es := fmt.Errorf("line %d: %v is not a valid operation", linenumber, s)
	if len(s) < 5 {
		return es
	}
	if s[1] != "=" {
		return es
	}

	v, err := opval(s[2:5], linenumber)
	if err != nil {
		return err
	}
	emap[s[0]] = ftoa(v)
	return nil
}

// assignop creates an assignment by computing an addition or substraction on an identifier
func assignop(s []string, linenumber int) error {
	operr := fmt.Errorf("line %d: %s is not a valid operation", linenumber, s)
	if len(s) < 4 {
		return operr
	}
	e, err := strconv.ParseFloat(eval(s[0]), 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a number", linenumber, s[0])
	}
	v, err := strconv.ParseFloat(s[3], 64)
	if err != nil {
		return fmt.Errorf("line %d: %v is not a number", linenumber, s[3])
	}
	switch s[1] {
	case "+":
		emap[s[0]] = ftoa(e + v)
	case "-":
		emap[s[0]] = ftoa(e - v)
	case "*":
		emap[s[0]] = ftoa(e * v)
	case "/":
		if v == 0 {
			return fmt.Errorf("line %d: you cannot divide by zero (%v / %v)", linenumber, e, v)
		}
		emap[s[0]] = ftoa(e / v)
	default:
		return operr
	}
	return nil
}

// area computes the diameter from a given area
// Area = Pi * (r*r), so given an area the diameter = sqrt ( area / pi) * 2
// usecase:
// x=somevalue
// d=area x
// circle x y d
func area(v float64) float64 {
	return math.Sqrt((v / math.Pi)) * 2
}

// polarfunc assigns polar coordinates
func polarfunc(s []string, linenumber int) error {
	e := fmt.Errorf("line %d use: v = [polar|polarx|polary] cx cy r theta", linenumber)
	if len(s) != 7 {
		return e
	}
	goodkey := s[2] == "polarx" || s[2] == "polary" || s[2] == "polar"
	if !goodkey {
		return e
	}
	var cx, cy, r, theta float64
	var err error

	cx, err = strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return err
	}
	cy, err = strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return err
	}
	r, err = strconv.ParseFloat(eval(s[5]), 64)
	if err != nil {
		return err
	}
	theta, err = strconv.ParseFloat(eval(s[6]), 64)
	if err != nil {
		return err
	}
	x, y := polar(cx, cy, r, theta*(math.Pi/180))
	switch s[2] {
	case "polarx":
		emap[s[0]] = ftoa(x)
	case "polary":
		emap[s[0]] = ftoa(y)
	case "polar":
		emap[s[0]+"_x"] = ftoa(x)
		emap[s[0]+"_y"] = ftoa(y)
	default:
		return e
	}
	return nil
}

// delim returns the index of sep in slice s.
// if not found return -1
func delim(s []string, sep string) int {
	for i, t := range s {
		if t == sep {
			return i
		}
	}
	return -1
}

// coordfunc assigns a coordinate pair
func coordfunc(s []string, linenumber int) error {
	e := fmt.Errorf("line %d use: p=(xexpr, yexpr)", linenumber)
	ls := len(s)
	goodarg := ls == 7 || ls == 9 || ls == 11
	if !goodarg {
		return e
	}
	xcoord := s[0] + "_x"
	ycoord := s[0] + "_y"
	end := ls - 1
	ci := delim(s, ",")
	if s[end] == ")" && ci != -1 && end > ci {
		left := s[3:ci]
		right := s[ci+1 : end]
		ll := len(left)
		lr := len(right)
		switch {
		case ll == 1 && lr == 1: // p=(a,b)
			emap[xcoord] = left[0]
			emap[ycoord] = right[0]
			return nil
		case ll == 1 && lr == 3: // p=(a,a+b)
			v, err := opval(right, linenumber)
			if err != nil {
				return err
			}
			emap[xcoord] = left[0]
			emap[ycoord] = ftoa(v)
			return nil
		case ll == 3 && lr == 1: // p=(a+b,b)
			v, err := opval(left, linenumber)
			if err != nil {
				return err
			}
			emap[xcoord] = ftoa(v)
			emap[ycoord] = right[0]
			return nil
		case ll == 3 && lr == 3: // p=(a+b,b+c)
			vl, lerr := opval(left, linenumber)
			if lerr != nil {
				return lerr
			}
			vr, rerr := opval(right, linenumber)
			if rerr != nil {
				return rerr
			}
			emap[xcoord] = ftoa(vl)
			emap[ycoord] = ftoa(vr)
			return nil
		default:
			return e
		}
	}
	return e
}

// vmap maps one interval to another
func vmap(value float64, low1 float64, high1 float64, low2 float64, high2 float64) float64 {
	return low2 + (high2-low2)*(value-low1)/(high1-low1)
}

// sprint makes a string assignment using formatted text
func sprint(s []string, linenumber int) error {
	var err error
	switch len(s) {
	case 5: // x=sprint fmt a
		var v1 float64
		v1, err = strconv.ParseFloat(eval(s[4]), 64) // make evaluated number (string) to number
		if err != nil {
			return err
		}
		emap[s[0]] = fmt.Sprintf(s[3], v1)
		return nil

	case 6: // x=sprint fmt a b
		var v1, v2 float64
		v1, err = strconv.ParseFloat(eval(s[4]), 64)
		if err != nil {
			return err
		}
		v2, err = strconv.ParseFloat(eval(s[5]), 64)
		if err != nil {
			return err
		}
		emap[s[0]] = fmt.Sprintf(s[3], v1, v2)
		return nil

	case 7: // x=sprint fmt a op b, or x=sprint fmt a b c
		var v1, v2, v3 float64
		op := s[5]
		if op == "+" || op == "-" || op == "*" || op == "/" || op == "%" {
			var v1 float64
			v1, err = opval(s[4:7], linenumber) // note that opval does eval
			if err != nil {
				return err
			}
			emap[s[0]] = fmt.Sprintf(s[3], v1)
			return nil
		}
		v1, err = strconv.ParseFloat(eval(s[4]), 64)
		if err != nil {
			return err
		}
		v2, err = strconv.ParseFloat(eval(s[5]), 64)
		if err != nil {
			return err
		}
		v3, err = strconv.ParseFloat(eval(s[6]), 64)
		if err != nil {
			return err
		}
		emap[s[0]] = fmt.Sprintf(s[3], v1, v2, v3)
		return nil

	case 8: // x=sprint fmt a b c d
		var v1, v2, v3, v4 float64
		v1, err = strconv.ParseFloat(eval(s[4]), 64)
		if err != nil {
			return err
		}
		v2, err = strconv.ParseFloat(eval(s[5]), 64)
		if err != nil {
			return err
		}
		v3, err = strconv.ParseFloat(eval(s[6]), 64)
		if err != nil {
			return err
		}
		v4, err = strconv.ParseFloat(eval(s[7]), 64)
		if err != nil {
			return err
		}
		emap[s[0]] = fmt.Sprintf(s[3], v1, v2, v3, v4)
		return nil

	case 9: // x=sprint fmt a b c d e
		var v1, v2, v3, v4, v5 float64
		v1, err = strconv.ParseFloat(eval(s[4]), 64)
		if err != nil {
			return err
		}
		v2, err = strconv.ParseFloat(eval(s[5]), 64)
		if err != nil {
			return err
		}
		v3, err = strconv.ParseFloat(eval(s[6]), 64)
		if err != nil {
			return err
		}
		v4, err = strconv.ParseFloat(eval(s[7]), 64)
		if err != nil {
			return err
		}
		v5, err = strconv.ParseFloat(eval(s[8]), 64)
		if err != nil {
			return err
		}
		emap[s[0]] = fmt.Sprintf(s[3], v1, v2, v3, v4, v5)
		return nil

	default:
		return fmt.Errorf("line %d: 1-5 arguments or one expression: %v %s", linenumber, s[3], s[4:])
	}
}

// substr makes a substring assignment
func substr(s []string, linenumber int) error {
	if len(s) != 6 {
		return fmt.Errorf("line %d: x=substr s begin end", linenumber)
	}
	src := unquote(eval(s[3]))
	bs := eval(s[4])
	es := eval(s[5])
	var b, e int
	var err error

	// beginning
	if bs == "beg" || bs == "-" {
		b = 0
	} else {
		b, err = strconv.Atoi(bs)
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
		if b < 0 {
			b = 0
		}
	}
	// end
	if es == "end" || es == "-" {
		e = len(src)
	} else {
		e, err = strconv.Atoi(es)
		e++
		if err != nil {
			return fmt.Errorf("line %d: %v", linenumber, err)
		}
		if e > len(src) {
			e = len(src)
		}
	}

	if b > e {
		return fmt.Errorf("line %d: s[%d:%d] is out of range", linenumber, b, e)
	}
	emap[s[0]] = `"` + src[b:e] + `"`
	return nil
}

// areafunc returns the diameter, given the area measure
func areafunc(s []string, linenumber int) error {
	var v float64
	var err error
	switch len(s) {
	case 4: // v = area x
		v, err = strconv.ParseFloat(eval(s[3]), 64)
		if err != nil {
			return err
		}
	case 6: // v = area a op b
		v, err = opval(s[3:6], linenumber)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d use: v = area x or v = area expression", linenumber)
	}
	emap[s[0]] = ftoa(area(v))
	return nil
}

// mathfunc returns the {sine|cosine|tangent|sqrt} of a number or binary operation
func mathfunc(s []string, linenumber int) error {
	var v float64
	var err error
	funcname := s[2]
	switch len(s) {
	case 4: // y = sqrt x
		v, err = strconv.ParseFloat(eval(s[3]), 64)
		if err != nil {
			return err
		}
	case 6: // y = sqrt a op b
		v, err = opval(s[3:6], linenumber)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d use: v = %s x or v = %s expression", linenumber, funcname, funcname)
	}
	var result float64
	switch funcname {
	case "sine":
		result = math.Sin(v)
	case "cosine":
		result = math.Cos(v)
	case "tangent":
		result = math.Tan(v)
	case "sqrt":
		if v < 0 {
			return fmt.Errorf("line %d: cannot take the square root of %g", linenumber, v)
		}
		result = math.Sqrt(v)
	}
	emap[s[0]] = ftoa(result)
	return nil
}

// random returns a bounded random number
func random(s []string, linenumber int) error {
	if len(s) != 5 {
		return fmt.Errorf("line %d use: v = random min max", linenumber)
	}
	var min, max float64
	var err error
	min, err = strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return err
	}
	max, err = strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return err
	}
	emap[s[0]] = ftoa(vmap(rand.Float64(), 0, 1, min, max))
	return nil
}

// vmapfunc translates a value given two ranges
func vmapfunc(s []string, linenumber int) error {
	n := len(s)
	if n < 8 {
		return fmt.Errorf("line %d: use: v = vmap data min1 max1 min2 max2", linenumber)
	}
	var data, min1, max1, min2, max2 float64
	var err error
	data, err = strconv.ParseFloat(eval(s[3]), 64)
	if err != nil {
		return err
	}
	min1, err = strconv.ParseFloat(eval(s[4]), 64)
	if err != nil {
		return err
	}
	max1, err = strconv.ParseFloat(eval(s[5]), 64)
	if err != nil {
		return err
	}
	min2, err = strconv.ParseFloat(eval(s[6]), 64)
	if err != nil {
		return err
	}
	max2, err = strconv.ParseFloat(eval(s[7]), 64)
	if err != nil {
		return err
	}
	emap[s[0]] = ftoa(vmap(data, min1, max1, min2, max2))
	return nil
}
