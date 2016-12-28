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
	PT_TEXT       = "text"
	PT_HREF       = "href"
	PT_HTML       = "html"
	PT_ATTR       = `attr\[([\w\W]+)\]`
	PT_ATTR_ARRAY = `attr-array\[([\w\W]+)\]`
	PT_IMG_SRC    = "src"
	PT_IMG_ALT    = "alt"
	PT_TEXT_ARRAY = "text-array"
	PT_HREF_ARRAY = "href-array"
	PT_MAP        = "map"
	PT_ARRAY      = "array"
	PT_JSON_VALUE = "json"
	PT_OUT_HTML   = "outhtml"

	PAGE_JSON = "json"
	PAGE_HTML = "html"
	PAGE_JS   = "js"
	PAGE_XML  = "xml"
)

type PipeItem struct {
	Name     string     `json:"name,omitempty"`
	Selector string     `json:"selector,omitempty"`
	Type     string     `json:"type"`
	Filter   string     `json:"filter,omitempty"`
	SubItem  []PipeItem `json:"subitem,omitempty"`
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
	}

	return nil, nil
}

func (p *PipeItem) parseRegexp(body string) (interface{}, error) {
	if p.Type != PT_TEXT {
		return nil, errors.New("not text type")
	}

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
	}

	return callFilter(rs, p.Filter)
}

func (p *PipeItem) pipeSelection(s *goquery.Selection) (interface{}, error) {

	var (
		sel *goquery.Selection = s
		err error
	)

	if strings.HasPrefix(p.Selector, "regexp:") && p.Type == PT_TEXT {
		body, _ := sel.Html()
		return p.parseRegexp(body)
	}

	if p.Selector != "" {
		sel, err = parseHtmlSelector(s, p.Selector)
		if err != nil {
			return nil, err
		}
	}

	if sel.Size() == 0 {
		return nil, errors.New("Selector can't Find node!: " + p.Selector)
	}

	attr_exp, _ := regexp.Compile(PT_ATTR)
	attr_array_exp, _ := regexp.Compile(PT_ATTR_ARRAY)

	if attr_exp.MatchString(p.Type) {
		vt := attr_exp.FindStringSubmatch(p.Type)
		res, has := sel.Attr(vt[1])
		if !has {
			return nil, errors.New("Can't Find attribute: " + p.Type + " selector: " + p.Selector)
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
	case PT_TEXT:
		return callFilter(sel.Text(), p.Filter)
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
			return nil, errors.New("Can't Find attribute: " + p.Type + " selector: " + p.Selector)
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
			res[subitem.Name], _ = subitem.pipeSelection(sel)
		}

		return callFilter(res, p.Filter)
	}

	return nil, errors.New("Not support pipe type")
}

func parseHtmlSelector(s *goquery.Selection, selector string) (*goquery.Selection, error) {
	if selector == "" {
		return s, nil
	}

	subs := strings.Split(selector, "|")
	if len(subs) < 1 {
		return s.Find(selector), nil
	}

	s = s.Find(subs[0])
	exp, _ := regexp.Compile(`([a-z_]+)(\(([\w\W+]+)\))?`)
	for i := 1; i < len(subs); i++ {
		if !exp.MatchString(subs[i]) {
			return s, errors.New("error parse html selector: " + subs[i])
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
	return s, nil
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
	case PT_TEXT:
		return callFilter(js.MustString(""), p.Filter)
	case PT_TEXT_ARRAY:
		v, err := js.StringArray()
		if err != nil {
			return nil, err
		}
		return callFilter(v, p.Filter)
	case PT_JSON_VALUE:
		return callFilter(js.Interface(), p.Filter)
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
	}

	return nil, nil
}
