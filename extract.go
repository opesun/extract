// Package validator implements validation and sanitization of input.
// Written to be used in web applications, where one must make sure that the input is well behaving before
// inserting to database.
//
// Possible gotcha: encoding/json.Unmarshal uses float64 for all numbers.
//
// For now, the design is to do NOT handle the incoming data gracefully, and say no at the slightest problem.
// Eg: quit if a slice is larger than the expected instead of truncating it, etc...
// Later there will be a flag to do the opposite.
//
// I am really unhappy about the solutions in this code though (too much redundant typing). A better solution must exists.
// János Dobronszki @ Opesun
package extract

import(
	"net/url"
	"strconv"
	"math"
)

const(
	min 		=	"min"
	max 		=	"max"
	min_amt 	=	"min_amt"
	max_amt 	=	"max_amt"
)

type Rules struct {
	R 		map[string]interface{}
}

func (r *Rules) ExtractForm(dat url.Values) (map[string]interface{}, bool) {
	return r.Extract(map[string][]string(dat))
}

func minMax(i int64, rules map[string]interface{}) bool {
	if min, hasmin := rules[min]; hasmin {
		if i < int64(min.(float64)) {
			return false
		}
	}
	if max, hasmax := rules[max]; hasmax {
		if i > int64(max.(float64)) {
			return false
		}
	}
	return true
}

func handleString(val string, rules map[string]interface{}) (string, bool) {
	len_ok := minMax(int64(len(val)), rules)
	if !len_ok {
		return val, false
	}
	return val, true
}

func handleInt(val string, rules map[string]interface{}) (int64, bool) {
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}
	size_ok := minMax(i, rules)	// This is so uncorrect. TODO: rethink
	if !size_ok {
		return i, false
	}
	return i, true
}

func handleFloat(val string, rules map[string]interface{}) (float64, bool) {
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}
	size_ok := minMax(int64(math.Ceil(f)), rules)	// This is so uncorrect. TODO: rethink
	if !size_ok {
		return f, false
	}
	return f, true
}

// TODO: rethink
func handleBool(val string, rules map[string]interface{}) (bool, bool) {
	b, err := strconv.ParseBool(val)
	if err != nil {
		return false, false
	}
	return b, true
}

// Slices

func minMaxS(l int, rules map[string]interface{}) bool {
	if min_amt, has_min := rules[min_amt]; has_min {
		if l < int(min_amt.(float64)) {
			return false
		}
	}
	if max_amt, has_max := rules[max_amt]; has_max {
		if l > int(max_amt.(float64)) {
			return false
		}
	}
	return true
}

func allOk(val []string, rules map[string]interface{}, f func(int) bool) bool {
	slen_ok := minMaxS(len(val), rules)
	if !slen_ok {
		return false
	}
	for i, _ := range val {
		if !f(i) {
			return false
		}
	}
	return true
}

func handleStringS(val []string, rules map[string]interface{}) ([]string, bool) {
	ret := []string{}
	return ret, allOk(val, rules,
	func(i int) bool {
		if str, ok := handleString(val[i], rules); ok {
			ret = append(ret, str)
			return true
		}
		return false
	})
}

func handleIntS(val []string, rules map[string]interface{}) ([]int64, bool) {
	ret := []int64{}
	return ret, allOk(val, rules,
	func(i int) bool {
		if fl, ok := handleInt(val[i], rules); ok {
			ret = append(ret, fl)
			return true
		}
		return false
	})
}

func handleFloatS(val []string, rules map[string]interface{}) ([]float64, bool) {
	ret := []float64{}
	return ret, allOk(val, rules,
	func(i int) bool {
		if fl, ok := handleFloat(val[i], rules); ok {
			ret = append(ret, fl)
			return true
		}
		return false
	})
}

func handleBoolS(val []string, rules map[string]interface{}) ([]bool, bool) {
	ret := []bool{}
	return ret, allOk(val, rules,
	func(i int) bool {
		if b, ok := handleBool(val[i], rules); ok {
			ret = append(ret, b)
			return true
		}
		return false
	})
}

func (r *Rules) Extract(dat map[string][]string) (map[string]interface{}, bool) {
	ret := map[string]interface{}{}
	// missing := false
	for i, v := range r.R {
		obj, is_obj := v.(map[string]interface{})
		val, hasval := dat[i]
		if !is_obj && hasval {	// Without any check
			ret[i] = val
		} else {
			_, must := obj["must"];
			if must && (!hasval || len(val) == 0) {
				return ret, false
			} else if !hasval || len(val) == 0 {
				continue
			}
			typ, hastype := obj["type"]
			if !hastype {
				if len(val) > 1 {
					return ret, false
				}
				s, passed := handleString(val[0], obj)
				if passed {
					ret[i] = s
				} else if must {
					return ret, false
				}
			} else {
				// passed := false
				switch typ {
					case "bools":
						s, pass := handleBoolS(val, obj)
						if !pass { return ret, false } else { ret[i] = s }
					case "strings":
						s, pass := handleStringS(val, obj)
						if !pass { return ret, false } else { ret[i] = s }
					case "ints":
						s, pass := handleIntS(val, obj)
						if !pass { return ret, false } else { ret[i] = s }
					case "floats":
						s, pass := handleFloatS(val, obj)
						if !pass { return ret, false } else { ret[i] = s }
					default:
						if len(val) > 1 {
							return ret, false
						}
						switch typ {
						case "bool":
							s, pass := handleBool(val[0], obj)
							if !pass { return ret, false } else { ret[i] = s }
						case "string":
							s, pass := handleString(val[0], obj)
							if !pass { return ret, false } else { ret[i] = s }
						case "int":
							s, pass := handleInt(val[0], obj)
							if !pass { return ret, false } else { ret[i] = s }
						case "float":
							s, pass := handleFloat(val[0], obj)
							if !pass { return ret, false } else { ret[i] = s }
						default:
							return ret, false
						}
				}
			}
		}
	}
	return ret, true
}

func (r *Rules) ResetRules(templ map[string]interface{}) {
	r.R = templ
}

func New(templ map[string]interface{}) *Rules {
	r := &Rules{templ}
	return r
}