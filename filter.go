package gopiper

import (
	"errors"
	"fmt"
	"html"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	RegisterFilter("sprintf", sprintf)
	RegisterFilter("sprintfmap", sprintfmap)
	RegisterFilter("unixtime", unixtime)
	RegisterFilter("unixmill", unixmill)
	RegisterFilter("paging", paging)
	RegisterFilter("quote", quote)
	RegisterFilter("unquote", unquote)
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

	exp, _ := regexp.Compile(`([a-zA-Z0-9\-_]+)(?:\(([\w\W]*?)\))?(\||$)`)
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
		return src.Interface(), errors.New("filter trim nil params")
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
	str := strings.TrimSpace(src.String())
	if str == "" {
		return []string{}, nil
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

func sprintf_multi_param(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		return src.Interface(), errors.New("filter split nil params ")
	}

	if src.Type().Kind() == reflect.Array || src.Type().Kind() == reflect.Slice {
		count := strings.Count(params.String(), "%")
		ret := make([]interface{}, 0)
		for i := 0; i < src.Len(); i++ {
			ret = append(ret, src.Index(i).Interface())
		}
		if len(ret) > count {
			return fmt.Sprintf(params.String(), ret[:count]...), nil
		}
		return fmt.Sprintf(params.String(), ret...), nil
	}

	return fmt.Sprintf(params.String(), src.Interface()), nil
}
func sprintf(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		return src.Interface(), errors.New("filter split nil params")
	}
	switch src.Interface().(type) {
	case string:
		return fmt.Sprintf(params.String(), src.String()), nil
	case []string:
		vt, _ := src.Interface().([]string)
		for i := 0; i < len(vt); i++ {
			if len(vt[i]) <= 0 {
				continue
			}
			vt[i] = fmt.Sprintf(params.String(), vt[i])
		}
		return vt, nil
	}

	return fmt.Sprintf(params.String(), src.Interface()), nil
}
func sprintfmap(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	if params == nil {
		return src.Interface(), errors.New("filter split nil params")
	}
	msrc, ok := src.Interface().(map[string]interface{})
	if ok == false {
		return src.Interface(), errors.New("value is not map[string]interface{}")
	}
	vt := strings.Split(params.String(), ",")
	if len(vt) <= 1 {
		return src.Interface(), errors.New("params length must > 1")
	}
	p_array := []interface{}{}
	for _, x := range vt[1:] {
		if vm, ok := msrc[x]; ok {
			p_array = append(p_array, vm)
		}
	}
	return fmt.Sprintf(vt[0], p_array...), nil
}

func unixtime(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return time.Now().Unix(), nil
}

func unixmill(src *reflect.Value, params *reflect.Value) (interface{}, error) {
	return time.Now().UnixNano() / int64(time.Millisecond), nil
}

func paging(src *reflect.Value, params *reflect.Value) (interface{}, error) {

	if params == nil {
		return src.Interface(), errors.New("filter paging nil params")
	}
	src_type := src.Type().Kind()
	if src_type != reflect.Slice && src_type != reflect.Array && src_type != reflect.String {
		return src.Interface(), errors.New("value is not slice ,array or string")
	}
	vt := strings.Split(params.String(), ",")
	if len(vt) < 2 {
		return src.Interface(), errors.New("params length must > 1")
	}

	start, err := strconv.Atoi(vt[0])
	end, err := strconv.Atoi(vt[1])
	if err != nil {
		return src.Interface(), errors.New("params type error:need int." + err.Error())
	}

	offset := -1
	if len(vt) == 3 {
		offset, err = strconv.Atoi(vt[2])
		return src.Interface(), errors.New("params type error:need int." + err.Error())
		if offset < 1 {
			return src.Interface(), errors.New("offset must > 0")
		}
	}

	var result []string
	switch src.Interface().(type) {
	case []interface{}:
		{
			vt, _ := src.Interface().([]interface{})
			for i := start; i <= end; i++ {
				for j := 0; j < len(vt); j++ {
					if offset > 0 {
						result = append(result, sprintf_replace(vt[j].(string), []string{strconv.Itoa(i * offset), strconv.Itoa((i + 1) * offset)}))
					} else {
						result = append(result, sprintf_replace(vt[j].(string), []string{strconv.Itoa(i)}))
					}
				}

			}
			return result, nil
		}
	case []string:
		{
			vt, _ := src.Interface().([]string)
			for i := start; i <= end; i++ {
				for j := 0; j < len(vt); j++ {
					if offset > 0 {
						result = append(result, sprintf_replace(vt[i], []string{strconv.Itoa(i * offset), strconv.Itoa((i + 1) * offset)}))
					} else {
						result = append(result, sprintf_replace(vt[i], []string{strconv.Itoa(i)}))
					}
				}

			}
			return result, nil
		}
	case string:
		{
			msrc1, ok := src.Interface().(string)
			if ok == true {
				for i := start; i <= end; i++ {
					if offset > 0 {
						result = append(result, sprintf_replace(msrc1, []string{strconv.Itoa(i * offset), strconv.Itoa((i + 1) * offset)}))
					} else {
						result = append(result, sprintf_replace(msrc1, []string{strconv.Itoa(i)}))
					}
				}
				return result, nil
			}
		}
	}
	return src.Interface(), errors.New("do nothing,src type not support!")
}

func sprintf_replace(src string, param []string) string {
	for i, _ := range param {
		src = strings.Replace(src, "{"+strconv.Itoa(i)+"}", param[i], -1)
	}
	return src
}

func quote(src *reflect.Value, params *reflect.Value) (interface{}, error) {

	switch src.Interface().(type) {
	case string:
		return strconv.Quote(src.String()), nil
	case []string:
		vt, _ := src.Interface().([]string)
		for i := 0; i < len(vt); i++ {
			vt[i] = strconv.Quote(vt[i])
		}
		return vt, nil
	}

	return src.Interface(), nil
}

func unquote(src *reflect.Value, params *reflect.Value) (interface{}, error) {

	switch src.Interface().(type) {
	case string:
		return strconv.Unquote(`"` + src.String() + `"`)
	case []string:
		vt, _ := src.Interface().([]string)
		for i := 0; i < len(vt); i++ {
			vt[i], _ = strconv.Unquote(`"` + vt[i] + `"`)
		}
		return vt, nil
	}

	return src.Interface(), nil
}
