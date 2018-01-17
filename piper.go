package gopiper

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/bitly/go-simplejson"
)

const (
	// begin new version
	PT_INT          = "int"
	PT_FLOAT        = "float"
	PT_BOOL         = "bool"
	PT_STRING       = "string"
	PT_INT_ARRAY    = "int-array"
	PT_FLOAT_ARRAY  = "float-array"
	PT_BOOL_ARRAY   = "bool-array"
	PT_STRING_ARRAY = "string-array"
	PT_MAP          = "map"
	PT_ARRAY        = "array"
	PT_JSON_VALUE   = "json"
	PT_JSON_PARSE   = "jsonparse"
	// end new version

	// begin compatible old version
	PT_TEXT       = "text"
	PT_HREF       = "href"
	PT_HTML       = "html"
	PT_ATTR       = `attr\[([\w\W]+)\]`
	PT_ATTR_ARRAY = `attr-array\[([\w\W]+)\]`
	PT_IMG_SRC    = "src"
	PT_IMG_ALT    = "alt"
	PT_TEXT_ARRAY = "text-array"
	PT_HREF_ARRAY = "href-array"
	PT_OUT_HTML   = "outhtml"
	// end compatible old version

	PAGE_JSON = "json"
	PAGE_HTML = "html"
	PAGE_JS   = "js"
	PAGE_XML  = "xml"
	PAGE_TEXT = "text"
)

type PipeItem struct {
	Name     string     `json:"name,omitempty"`
	Selector string     `json:"selector,omitempty"`
	Type     string     `json:"type"`
	Filter   string     `json:"filter,omitempty"`
	SubItem  []PipeItem `json:"subitem,omitempty"`
}

type htmlselector struct {
	*goquery.Selection
	attr     string
	selector string
}

func (p *PipeItem) PipeBytes(body []byte, pagetype string) (interface{}, error) {
	switch pagetype {
	case PAGE_HTML:
		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		return p.pipeSelection(doc.Selection)
	case PAGE_JSON:
		return p.pipeJson(body)
	case PAGE_TEXT:
		return p.pipeText(body)
	}
	return nil, nil
}

func (p *PipeItem) parseRegexp(body string) (interface{}, error) {
	s := p.Selector[7:]
	exp, err := regexp.Compile(s)
	if err != nil {
		return nil, err
	}

	sv := exp.FindStringSubmatch(body)
	rs := ""

	if len(sv) == 1 {
		rs = sv[0]
	} else if len(sv) > 1 {
		rs = sv[1]
		sv = sv[1:]
	}

	switch p.Type {
	case PT_INT, PT_FLOAT, PT_BOOL:
		val, err := parseTextValue(rs, p.Type)
		if err != nil {
			return nil, err
		}
		return callFilter(val, p.Filter)
	case PT_INT_ARRAY, PT_FLOAT_ARRAY, PT_BOOL_ARRAY:
		val, err := parseTextValue(sv, p.Type)
		if err != nil {
			return nil, err
		}
		return callFilter(val, p.Filter)
	case PT_TEXT, PT_STRING:
		return callFilter(rs, p.Filter)
	case PT_TEXT_ARRAY, PT_STRING_ARRAY:
		return callFilter(sv, p.Filter)
	case PT_JSON_PARSE:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type jsonparse need one subItem!")
		}
		body, err := text2jsonbyte(rs)
		if err != nil {
			return nil, errors.New("jsonparse: text is not a json string" + err.Error())
		}
		parse_item := p.SubItem[0]
		res, err := parse_item.pipeJson(body)
		if err != nil {
			return nil, err
		}
		return callFilter(res, p.Filter)
	case PT_JSON_VALUE:
		res, err := text2json(rs)
		if err != nil {
			return nil, err
		}
		return callFilter(res, p.Filter)
	case PT_MAP:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type array need one subItem!")
		}
		res := make(map[string]interface{})
		for _, subitem := range p.SubItem {
			if subitem.Name == "" {
				continue
			}
			res[subitem.Name], _ = subitem.pipeText([]byte(rs))
		}
		return callFilter(res, p.Filter)
	}
	return nil, errors.New("Not support pipe type")
}

func (p *PipeItem) pipeSelection(s *goquery.Selection) (interface{}, error) {

	var (
		sel = htmlselector{s, "", p.Selector}
		err error
	)

	if strings.HasPrefix(p.Selector, "regexp:") {
		body, _ := sel.Html()
		return p.parseRegexp(body)
	}

	selector := p.Selector
	if selector != "" {
		sel, err = parseHtmlSelector(s, selector)
		if err != nil {
			return nil, err
		}
		selector = sel.selector
	}

	if sel.Size() == 0 {
		return nil, errors.New("Selector can't Find node!: " + selector)
	}

	attr_exp, _ := regexp.Compile(PT_ATTR)
	attr_array_exp, _ := regexp.Compile(PT_ATTR_ARRAY)

	if attr_exp.MatchString(p.Type) {
		vt := attr_exp.FindStringSubmatch(p.Type)
		res, has := sel.Attr(vt[1])
		if !has {
			return nil, errors.New("Can't Find attribute: " + p.Type + " selector: " + selector)
		}
		return callFilter(res, p.Filter)
	} else if attr_array_exp.MatchString(p.Type) {
		vt := attr_array_exp.FindStringSubmatch(p.Type)
		res := make([]string, 0)
		sel.Each(func(index int, child *goquery.Selection) {
			href, has := child.Attr(vt[1])
			if has {
				res = append(res, href)
			}
		})
		return callFilter(res, p.Filter)
	}

	switch p.Type {
	case PT_INT, PT_FLOAT, PT_BOOL, PT_STRING, PT_TEXT, PT_INT_ARRAY, PT_FLOAT_ARRAY, PT_BOOL_ARRAY, PT_STRING_ARRAY:
		val, err := parseHtmlAttr(sel, p.Type)
		if err != nil {
			return nil, err
		}
		return callFilter(val, p.Filter)
	case PT_HTML:
		html := ""
		sel.Each(func(idx int, s1 *goquery.Selection) {
			str, _ := s1.Html()
			html += str
		})
		return callFilter(html, p.Filter)
	case PT_OUT_HTML:
		html := ""
		sel.Each(func(idx int, s1 *goquery.Selection) {
			str, _ := goquery.OuterHtml(s1)
			html += str
		})
		return callFilter(html, p.Filter)
	case PT_HREF, PT_IMG_SRC, PT_IMG_ALT:
		res, has := sel.Attr(p.Type)
		if !has {
			return nil, errors.New("Can't Find attribute: " + p.Type + " selector: " + selector)
		}
		return callFilter(res, p.Filter)
	case PT_TEXT_ARRAY:
		res := make([]string, 0)
		sel.Each(func(index int, child *goquery.Selection) {
			res = append(res, child.Text())
		})
		return callFilter(res, p.Filter)
	case PT_HREF_ARRAY:
		res := make([]string, 0)
		sel.Each(func(index int, child *goquery.Selection) {
			href, has := child.Attr("href")
			if has {
				res = append(res, href)
			}
		})
		return callFilter(res, p.Filter)
	case PT_ARRAY:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type array need one subItem!")
		}
		array_item := p.SubItem[0]
		res := make([]interface{}, 0)
		sel.Each(func(index int, child *goquery.Selection) {
			v, _ := array_item.pipeSelection(child)
			res = append(res, v)
		})
		return callFilter(res, p.Filter)
	case PT_MAP:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type array need one subItem!")
		}
		res := make(map[string]interface{})
		for _, subitem := range p.SubItem {
			if subitem.Name == "" {
				continue
			}
			res[subitem.Name], _ = subitem.pipeSelection(sel.Selection)
		}

		return callFilter(res, p.Filter)
	default:
		return callFilter(0, p.Filter)
	}

	return nil, errors.New("Not support pipe type")
}

func parseHtmlSelector(s *goquery.Selection, selector string) (htmlselector, error) {
	attr := ""
	if selector == "" {
		return htmlselector{s, attr, selector}, nil
	}

	if idx := strings.Index(selector, "//"); idx > 0 {
		attr = strings.TrimSpace(selector[idx+2:])
		selector = strings.TrimSpace(selector[:idx])
	}

	subs := strings.Split(selector, "|")
	if len(subs) < 1 {
		return htmlselector{s.Find(selector), attr, selector}, nil
	}

	s = s.Find(subs[0])
	exp, _ := regexp.Compile(`([a-z_]+)(\(([\w\W+]+)\))?`)
	for i := 1; i < len(subs); i++ {
		if !exp.MatchString(subs[i]) {
			return htmlselector{s, attr, selector}, errors.New("error parse html selector: " + subs[i])
		}

		vt := exp.FindStringSubmatch(subs[i])
		fn := vt[1]
		params := ""
		if len(vt) > 3 {
			params = strings.TrimSpace(vt[3])
		}

		switch fn {
		case "eq":
			pm, _ := strconv.Atoi(params)
			s = s.Eq(pm)
		case "next":
			s = s.Next()
		case "prev":
			s = s.Prev()
		case "first":
			s = s.First()
		case "last":
			s = s.Last()
		case "siblings":
			s = s.Siblings()
		case "nextall":
			s = s.NextAll()
		case "children":
			s = s.Children()
		case "parent":
			s = s.Parent()
		case "parents":
			s = s.Parents()
		case "not":
			if params != "" {
				s = s.Not(params)
			}
		case "filter":
			if params != "" {
				s = s.Filter(params)
			}
		case "prevfilter":
			if params != "" {
				s = s.PrevFiltered(params)
			}
		case "prevallfilter":
			if params != "" {
				s = s.PrevAllFiltered(params)
			}
		case "nextfilter":
			if params != "" {
				s = s.NextFiltered(params)
			}
		case "nextallfilter":
			if params != "" {
				s = s.NextAllFiltered(params)
			}
		case "parentfilter":
			if params != "" {
				s = s.ParentFiltered(params)
			}
		case "parentsfilter":
			if params != "" {
				s = s.ParentsFiltered(params)
			}
		case "childrenfilter":
			if params != "" {
				s = s.ChildrenFiltered(params)
			}
		case "siblingsfilter":
			if params != "" {
				s = s.SiblingsFiltered(params)
			}
		case "rm":
			if params != "" {
				s.Find(params).Remove()
			}
		}
	}
	return htmlselector{s, attr, selector}, nil
}

func parseTextValue(text interface{}, tp string) (interface{}, error) {
	switch tp {
	case PT_INT, PT_INT_ARRAY:
		return text2int(text)
	case PT_FLOAT, PT_FLOAT_ARRAY:
		return text2float(text)
	case PT_BOOL, PT_BOOL_ARRAY:
		return text2bool(text)
	}
	return text, nil
}

func parseHtmlAttr(sel htmlselector, tp string) (interface{}, error) {
	switch tp {
	case PT_INT, PT_FLOAT, PT_BOOL, PT_TEXT, PT_STRING:
		text, err := gethtmlattr(sel.Selection, sel.attr, sel.selector)
		if err != nil {
			return nil, err
		}
		return parseTextValue(text, tp)
	case PT_INT_ARRAY, PT_FLOAT_ARRAY, PT_BOOL_ARRAY, PT_STRING_ARRAY:
		text, err := gethtmlattr_array(sel.Selection, sel.attr, sel.selector)
		if err != nil {
			return nil, err
		}
		return parseTextValue(text, tp)
	}

	return nil, errors.New("unknow html attr")
}

func gethtmlattr_array(sel *goquery.Selection, attr, selector string) ([]string, error) {
	res := make([]string, 0)
	if attr == "" {
		sel.Each(func(index int, child *goquery.Selection) {
			res = append(res, child.Text())
		})
		return res, nil
	}

	attr_exp, _ := regexp.Compile(PT_ATTR)

	if attr_exp.MatchString(attr) {
		vt := attr_exp.FindStringSubmatch(attr)
		sel.Each(func(index int, child *goquery.Selection) {
			text, has := child.Attr(vt[1])
			if has {
				res = append(res, text)
			}
		})
		return res, nil
	} else if attr == "html" {
		sel.Each(func(idx int, s1 *goquery.Selection) {
			str, _ := s1.Html()
			res = append(res, str)
		})
		return res, nil
	} else if attr == "outhtml" {
		sel.Each(func(idx int, s1 *goquery.Selection) {
			str, _ := goquery.OuterHtml(s1)
			res = append(res, str)
		})
		return res, nil
	}

	return res, nil
}

func gethtmlattr(sel *goquery.Selection, attr, selector string) (string, error) {
	if attr == "" {
		return sel.Text(), nil
	}

	attr_exp, _ := regexp.Compile(PT_ATTR)

	if attr_exp.MatchString(attr) {
		vt := attr_exp.FindStringSubmatch(attr)
		res, has := sel.Attr(vt[1])
		if !has {
			return "", errors.New("Can't Find attribute: " + attr + " selector: " + selector)
		}
		return res, nil
	} else if attr == "html" {
		html := ""
		sel.Each(func(idx int, s1 *goquery.Selection) {
			str, _ := s1.Html()
			html += str
		})
		return html, nil
	} else if attr == "outhtml" {
		html := ""
		sel.Each(func(idx int, s1 *goquery.Selection) {
			str, _ := goquery.OuterHtml(s1)
			html += str
		})
		return html, nil
	}

	return sel.Text(), nil
}

func parseJsonSelector(js *simplejson.Json, selector string) (*simplejson.Json, error) {
	subs := strings.Split(selector, ".")

	for _, s := range subs {
		if index := strings.Index(s, "["); index >= 0 {
			if index > 0 {
				k := s[:index]
				if k != "this" {
					js = js.Get(k)
				}
			}
			s = s[index:]
			exp, _ := regexp.Compile(`^\[(\d+)\]$`)
			if !exp.MatchString(s) {
				return nil, errors.New("parse json selector error:  " + selector)
			}
			v := exp.FindStringSubmatch(s)
			int_v, err := strconv.Atoi(v[1])
			if err != nil {
				return nil, err
			}
			js = js.GetIndex(int_v)
		} else {
			if s == "this" {
				continue
			}
			js = js.Get(s)
		}
	}
	return js, nil
}

func (p *PipeItem) pipeJson(body []byte) (interface{}, error) {

	js, err := simplejson.NewJson(body)
	if err != nil {
		return nil, err
	}

	if p.Selector != "" {
		js, err = parseJsonSelector(js, p.Selector)
		if err != nil {
			return nil, err
		}
	}

	switch p.Type {
	case PT_INT:
		return callFilter(js.MustInt64(0), p.Filter)
	case PT_FLOAT:
		return callFilter(js.MustFloat64(0.0), p.Filter)
	case PT_BOOL:
		return callFilter(js.MustBool(false), p.Filter)
	case PT_TEXT, PT_STRING:
		return callFilter(js.MustString(""), p.Filter)
	case PT_TEXT_ARRAY, PT_STRING_ARRAY:
		v, err := js.StringArray()
		if err != nil {
			return nil, err
		}
		return callFilter(v, p.Filter)
	case PT_JSON_VALUE:
		return callFilter(js.Interface(), p.Filter)
	case PT_JSON_PARSE:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type jsonparse need one subItem!")
		}
		body_str := strings.TrimSpace(js.MustString(""))
		if body_str == "" {
			return nil, nil
		}
		body, err := text2jsonbyte(body_str)
		if err != nil {
			return nil, errors.New("jsonparse: text is not a json string" + err.Error())
		}
		parse_item := p.SubItem[0]
		res, err := parse_item.pipeJson(body)
		if err != nil {
			return nil, err
		}
		return callFilter(res, p.Filter)
	case PT_ARRAY:
		v, err := js.Array()
		if err != nil {
			return nil, err
		}

		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type array need one subItem!")
		}
		array_item := p.SubItem[0]
		res := make([]interface{}, 0)
		for _, r := range v {
			data, _ := json.Marshal(r)
			vl, _ := array_item.pipeJson(data)
			res = append(res, vl)
		}
		return callFilter(res, p.Filter)
	case PT_MAP:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type array need one subItem!")
		}
		data, _ := json.Marshal(js)
		res := make(map[string]interface{})
		for _, subitem := range p.SubItem {
			if subitem.Name == "" {
				continue
			}
			res[subitem.Name], _ = subitem.pipeJson(data)
		}

		return callFilter(res, p.Filter)
	default:
		return callFilter(0, p.Filter)
	}

	return nil, nil
}

func (p *PipeItem) pipeText(body []byte) (interface{}, error) {
	body_str := string(body)
	if strings.HasPrefix(p.Selector, "regexp:") {
		return p.parseRegexp(body_str)
	}

	switch p.Type {
	case PT_INT, PT_FLOAT, PT_BOOL:
		val, err := parseTextValue(body_str, p.Type)
		if err != nil {
			return nil, err
		}
		return callFilter(val, p.Filter)
	case PT_TEXT, PT_STRING:
		return callFilter(body_str, p.Filter)
	case PT_JSON_PARSE:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type jsonparse need one subItem!")
		}
		body, err := text2jsonbyte(body_str)
		if err != nil {
			return nil, errors.New("jsonparse: text is not a json string" + err.Error())
		}
		parse_item := p.SubItem[0]
		res, err := parse_item.pipeJson(body)
		if err != nil {
			return nil, err
		}
		return callFilter(res, p.Filter)
	case PT_JSON_VALUE:
		res, err := text2json(string(body))
		if err != nil {
			return nil, err
		}
		return callFilter(res, p.Filter)
	case PT_MAP:
		if p.SubItem == nil || len(p.SubItem) <= 0 {
			return nil, errors.New("Pipe type array need one subItem!")
		}
		res := make(map[string]interface{})
		for _, subitem := range p.SubItem {
			if subitem.Name == "" {
				continue
			}
			res[subitem.Name], _ = subitem.pipeText(body)
		}
		return callFilter(res, p.Filter)
	default:
		return callFilter(0, p.Filter)
	}

	return nil, errors.New("Not support pipe type")
}

func text2int(text interface{}) (interface{}, error) {
	switch val := text.(type) {
	case string:
		return strconv.ParseInt(val, 10, 64)
	case []string:
		vs := make([]int64, 0)
		for _, v := range val {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}
			vs = append(vs, n)
		}
		return vs, nil
	}
	return nil, errors.New("unsupport text2int type")
}

func text2float(text interface{}) (interface{}, error) {
	switch val := text.(type) {
	case string:
		return strconv.ParseFloat(val, 64)
	case []string:
		vs := make([]float64, 0)
		for _, v := range val {
			n, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return nil, err
			}
			vs = append(vs, n)
		}
		return vs, nil
	}
	return nil, errors.New("unsupport text2float type")
}

func text2bool(text interface{}) (interface{}, error) {
	switch val := text.(type) {
	case string:
		return strconv.ParseBool(val)
	case []string:
		vs := make([]bool, 0)
		for _, v := range val {
			n, err := strconv.ParseBool(v)
			if err != nil {
				return nil, err
			}
			vs = append(vs, n)
		}
		return vs, nil
	}
	return nil, errors.New("unsupport text2bool type")
}

func text2json(text string) (interface{}, error) {
	res, err := textJsonValue(text)
	if err != nil {
		return untextJsonValue(text)
	}
	return res, nil
}

func text2jsonbyte(text string) ([]byte, error) {
	val, err := text2json(text)
	if err != nil {
		return nil, err
	}
	return json.Marshal(val)
}

func textJsonValue(text string) (interface{}, error) {
	res := map[string]interface{}{}
	if err := json.Unmarshal([]byte(text), &res); err != nil {
		resarray := make([]interface{}, 0)
		if err = json.Unmarshal([]byte(text), &resarray); err != nil {
			return nil, errors.New("parse json value error, text is not json value: " + err.Error())
		}
		return resarray, nil
	}

	return res, nil
}

func untextJsonValue(text string) (interface{}, error) {
	text, err := strconv.Unquote(`"` + text + `"`)
	if err != nil {
		return nil, err
	}
	return textJsonValue(text)
}
