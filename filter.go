package gopiper

import (
	"errors"
	"fmt"
	"html"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	RegisterFilter("preadd", preadd)
	RegisterFilter("postadd", postadd)
	RegisterFilter("replace", replace)
	RegisterFilter("split", split)
	RegisterFilter("join", join)
	RegisterFilter("trim", trim)
	RegisterFilter("trimspace", trimspace)
	RegisterFilter("substr", substr)
	RegisterFilter("intval", intval)
	RegisterFilter("floatval", floatval)
	RegisterFilter("hrefreplace", hrefreplace)
	RegisterFilter("wraphtml", wraphtml)
	RegisterFilter("tosbc", tosbc)
	RegisterFilter("unescape", unescape)
	RegisterFilter("escape", escape)
}

type FilterFunction func(src *reflect.Value, params *reflect.Value) (interface{}, error)

var filters = make(map[string]FilterFunction)

func RegisterFilter(name string, fn FilterFunction) {
	_, existing := filters[name]
	if existing {
		panic(fmt.Sprintf("Filter with name '%s' is already registered.", name))
	}
	filters[name] = fn
}

func ReplaceFilter(name string, fn FilterFunction) {
	_, existing := filters[name]
	if !existing {
		panic(fmt.Sprintf("Filter with name '%s' does not exist (therefore cannot be overridden).", name))
	}
	filters[name] = fn
}

func applyFilter(name string, src *reflect.Value, params *reflect.Value) (interface{}, error) {
	fn, existing := filters[name]
	if !existing {
		return nil, errors.New(fmt.Sprintf("Filter with name '%s' not found.", name))
	}
	return fn(src, params)
}

func callFilter(src interface{}, value string) (interface{}, error) {

	if src == nil || len(value) == 0 {
		return src, nil
	}

	exp, _ := regexp.Compile(`([a-zA-Z0-9\-_]+)\(([\w\W]*?)\)(\||$)`)
	vt := exp.FindAllStringSubmatch(value, -1)

	for _, v := range vt {
		if len(v) < 3 {
			continue
		}
		name := v[1]
		params := v[2]

		src_value := reflect.ValueOf(src)
		param_value := reflect.ValueOf(params)
		next, err := applyFilter(name, &src_value, &param_value)
		if err != nil {
			continue
		}
		src = next

	}

	return src, nil
}

func preadd(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return params.String() + src.String(), nil
}
func postadd(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return src.String() + params.String(), nil
}
func substr(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	vt := strings.Split(params.String(), ",")
	if len(vt) == 1 {
		start, _ := strconv.Atoi(vt[0])
		return src.String()[start:], nil
	} else if len(vt) == 2 {
		start, _ := strconv.Atoi(vt[0])
		end, _ := strconv.Atoi(vt[1])
		return src.String()[start:end], nil
	}
	return src.Interface(), nil
}
func replace(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	vt := strings.Split(params.String(), ",")
	if len(vt) == 1 {
		return strings.Replace(src.String(), vt[0], "", -1), nil
	} else if len(vt) == 2 {
		return strings.Replace(src.String(), vt[0], vt[1], -1), nil
	} else if len(vt) == 3 {
		n, _ := strconv.Atoi(vt[2])
		return strings.Replace(src.String(), vt[0], vt[1], n), nil
	}
	return src.Interface(), nil
}
func trim(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		log.Println(src.String())
		return src.Interface(), errors.New("filter split nil params")
	}

	switch src.Interface().(type) {
	case string:
		return strings.Trim(src.String(), params.String()), nil
	case []string:
		vt, _ := src.Interface().([]string)
		for i := 0; i < len(vt); i++ {
			vt[i] = strings.Trim(vt[i], params.String())
		}
		return vt, nil
	}

	return src.Interface(), nil
}

func trimspace(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		log.Println(src.String())
		return src.Interface(), errors.New("filter split nil params")
	}

	switch src.Interface().(type) {
	case string:
		return strings.TrimSpace(src.String()), nil
	case []string:
		vt, _ := src.Interface().([]string)
		for i := 0; i < len(vt); i++ {
			vt[i] = strings.TrimSpace(vt[i])
		}
		return vt, nil
	}

	return src.Interface(), nil
}

func split(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		return src.Interface(), errors.New("filter split nil params")
	}
	return strings.Split(src.String(), params.String()), nil
}

func join(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		return src.Interface(), errors.New("filter split nil params")
	}
	switch src.Interface().(type) {
	case []string:
		vt, _ := src.Interface().([]string)
		rs := make([]string, 0)
		for _, v := range vt {
			if len(v) > 0 {
				rs = append(rs, v)
			}
		}
		return strings.Join(rs, params.String()), nil
	}

	return src.Interface(), nil
}

func intval(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if src.Interface() == nil {
		return 0, nil
	}
	v, _ := strconv.Atoi(src.String())
	return v, nil
}

func floatval(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if src.Interface() == nil {
		return 0.0, nil
	}
	v, _ := strconv.ParseFloat(src.String(), 64)
	return v, nil
}

func hrefreplace(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	href_filter_regexp, _ := regexp.Compile(`href(\s*)=(\s*)([\w\W]+?)"`)
	return href_filter_regexp.ReplaceAllString(src.String(), params.String()), nil
}

func regexpreplace(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return src.Interface(), nil
}

func tosbc(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	res := ""
	for _, t := range src.String() {
		if t == 12288 {
			t = 32
		} else if t > 65280 && t < 65375 {
			t = t - 65248
		}
		res += string(t)
	}
	return res, nil
}

func unescape(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return html.UnescapeString(src.String()), nil
}

func escape(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return html.EscapeString(src.String()), nil
}

func wraphtml(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		return src.Interface(), errors.New("filter wraphtml nil params")
	}

	switch src.Interface().(type) {
	case string:
		return fmt.Sprintf("<%s>%s</%s>", params.String(), src.String(), params.String()), nil
	case []string:
		vt, _ := src.Interface().([]string)
		for i := 0; i < len(vt); i++ {
			if len(vt[i]) <= 0 {
				continue
			}
			vt[i] = fmt.Sprintf("<%s>%s</%s>", params.String(), vt[i], params.String())
		}
		return vt, nil
	}

	return src.Interface(), nil
}
