package gopiper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/axgle/mahonia"
	simplejson "github.com/bitly/go-simplejson"
	"github.com/lauyoume/gohttp"
)

func TestSelector(t *testing.T) {
	js, err := simplejson.NewJson([]byte(`{"value": ["1","2",{"data": ["3", "2", "1"]}]}`))
	if err != nil {
		log.Println(err)
		return
	}
	js, err = parseJsonSelector(js, "this.value[2].data[1]")
	log.Println(js, err)
}

func TestJsonPipe(t *testing.T) {
	req := gohttp.New()

	resp, _ := req.Get("http://s.m.taobao.com/search?&q=qq&atype=b&searchfrom=1&from=1&sst=1&m=api4h5").End()

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	pipe := PipeItem{}
	json.Unmarshal([]byte(`
		{
			"type": "array",
			"selector": "this.listItem",
			"subitem": [
				{
					"type": "map",
					"subitem": [
						{
							"name": "now",
							"filter": "unixtime()"
						},
						{
							"name": "nowmill",
							"filter": "unixmill()"
						},
						{
							"type": "text",
							"selector": "nick",
							"name": "nickname"
						},
						{
							"type": "text",
							"selector": "name",
							"name": "title"
						}
					]
				}
			]
		}
	`), &pipe)

	fmt.Println(pipe.PipeBytes(body, "json"))
}

func TestHtmlPipe(t *testing.T) {

	log.Println(callFilter("美团他|女神||", `preadd(AAAA)|split(|)|join(,)`))
	req := gohttp.New()

	resp, _ := req.Get("http://movie.douban.com/subject/25850640/").End()

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	//resp2, errs := req.Get("http://movie.douban.com/subject/1306784/?from=tag_all").End()
	//resp2, errs := req.Get("http://movie.douban.com/subject/26087497/").End()
	//resp2, errs := req.Get("http://movie.douban.com/subject/25723907/").End()
	//resp2, errs := req.Get("http://movie.douban.com/subject/4014396/").End()
	resp2, errs := req.Get("http://movie.douban.com/subject/2035218/?from=tag_all").End()

	if errs != nil {
		log.Println("hahah")
		log.Println(errs)
		return
	}

	defer resp2.Body.Close()
	timer := time.AfterFunc(time.Millisecond*100, func() {
		resp2.Body.Close()
	})
	body2, err := ioutil.ReadAll(resp2.Body)

	if err != nil {
		log.Println(err)
		return
	}
	timer.Stop()

	pipe := PipeItem{}
	err = json.Unmarshal([]byte(`
		{
			"type": "map",
			"selector": "",
			"subitem": [
						{
							"type": "text",
							"selector": "title",
							"name": "name",
							"filter": "trim(\n)|replace((豆瓣))|trim( )"
						},
						{
							"type": "attr[data-type]",
							"selector": "#content .gtleft a.bn-sharing",
							"name": "fenlei"
						},
						{
							"type": "attr[data-pic]",
							"selector": "#content .gtleft a.bn-sharing",
							"name": "thumbnail"
						},
						{
							"type": "text-array",
							"selector": "#info span.attrs a[rel=v\\:directedBy]",
							"name": "direct"
						},
						{
							"type": "text-array",
							"selector": "#info span a[rel=v\\:starring]",
							"name": "starring"
						},
						{
							"type": "text-array",
							"selector": "#info span[property=v\\:genre]",
							"name": "type"
						},
						{
							"type": "attr-array[src]",
							"selector": "#related-pic .related-pic-bd a:not(.related-pic-video) img",
							"name": "imgs",
							"filter": "join($)|replace(albumicon,photo)|split($)"
						},
						{
							"type": "text-array",
							"selector": "#info span[property=v\\:initialReleaseDate]",
							"name": "releasetime"
						},
						{
							"type": "text",
							"selector": "#info span[property=v\\:runtime]",
							"name": "longtime"
						},
						{
							"type": "text",
							"selector": "regexp:<span class=\"pl\">制片国家/地区:</span> ([\\w\\W]+?)<br/>",
							"name": "country",
							"filter": "split(/)|trim( )"
						},
						{
							"type": "text",
							"selector": "regexp:<span class=\"pl\">语言:</span> ([\\w\\W]+?)<br/>",
							"name": "language",
							"filter": "split(/)|trim( )"
						},
						{
							"type": "text",
							"selector": "regexp:<span class=\"pl\">集数:</span> (\\d+)<br/>",
							"name": "episode",
							"filter": "intval()"
						},
						{
							"type": "text",
							"selector": "regexp:<span class=\"pl\">又名:</span> ([\\w\\W]+?)<br/>",
							"name": "alias",
							"filter": "split(/)|trim( )"
						},
			    		{
			    			"type": "text",
							"selector": "#link-report span.hidden, #link-report span[property=v\\:summary]|last",
			    			"name": "brief",
			    			"filter": "trim(\n )|split(\n)|trim( )|wraphtml(p)|join()"
			    		},
						{
							"type": "text",
							"selector": "#interest_sectl .rating_num",
							"name": "score",
							"filter": "floatval()"
						},
						{
							"type": "text",
							"selector": "#content h1 span.year",
							"name": "year",
							"filter": "replace(()|replace())|intval()"
						},
						{
							"type": "text",
							"selector": "#comments-section > .mod-hd h2 a",
							"name": "comment",
							"filter": "replace(全部)|replace(条)|trim( )|intval()"
						}
			]
		}
	`), &pipe)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println(pipe.PipeBytes(body, "html"))
	v, _ := (pipe.PipeBytes(body2, "html"))
	bd, _ := json.Marshal(v)
	fmt.Println(string(bd))
}

func TestHtmlJingJiang(t *testing.T) {

	req := gohttp.New()

	resp, _ := req.Get("http://www.jjwxc.net/bookbase_slave.php?submit=&booktype=&opt=&page=3&endstr=&orderstr=4").End()

	defer resp.Body.Close()
	body_gbk, _ := ioutil.ReadAll(resp.Body)

	body_utf8 := mahonia.NewDecoder("gb18030").ConvertString(string(body_gbk))
	body := []byte(body_utf8)

	pipe := PipeItem{}
	err := json.Unmarshal([]byte(`
		{
			"type": "array",
			"selector": ".cytable tr:not(:nth-child(1))",
			"subitem": [
						{
							"type": "map",
							"subitem": [
								{
									"type": "text",
									"selector": "td:nth-child(1) a",
									"name":"author"
								},
								{
									"type": "text",
									"selector": "td:nth-child(2) a",
									"name": "name"
								},
								{
									"type": "attr[href]",
									"selector": "td:nth-child(2) a",
									"name": "source",
									"filter": "preadd(http://www.jjwxc.net/)"
								}
							]
						}
				]
		}
	`), &pipe)
	if err != nil {
		log.Println(err)
		return
	}

	v, _ := (pipe.PipeBytes(body, "html"))
	bd, _ := json.Marshal(v)
	fmt.Println(string(bd))
}

func TestBaiduLocal(t *testing.T) {
	req := gohttp.New()

	resp, _ := req.Get("http://www.baidu.com/s?wd=采集关键词&rn=50&tn=baidulocal").End()

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	pipe := PipeItem{}
	err := json.Unmarshal([]byte(`
		{
                "type" : "array",
                "selector": "table td ol table",
                "subitem": [
                    {
                        "type": "map",
                        "subitem": [
                            {
                                "name": "title",
                                "type" : "text",
                                "selector" : "td > a"
                            },
                            {
                                "name" : "url",
                                "type": "href",
                                "selector": "td a"
                            },
                            {
                                "name" : "desc",
                                "selector": "td > font|rm(font[color=\\#008000], font > a)",
                                "type" : "text"
                            }
                        ]
                    }
                ]
		}
	`), &pipe)
	if err != nil {
		log.Println(err)
		return
	}

	v, _ := (pipe.PipeBytes(body, "html"))
	bd, _ := json.Marshal(v)
	fmt.Println(string(bd))
}

func TestTTKBPaging(t *testing.T) {

	req := gohttp.New()

	resp, _ := req.Get("http://r.cnews.qq.com/getSubChannels").End()

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	pipe := PipeItem{}

	err := json.Unmarshal([]byte(`{
		"selector": "channellist",
		"type": "array",
		"name": "ROOT",
		"filter": "paging(1,2)",
		"subitem": [
			{
				"name": "child",
				"selector": "chlid",
				"type": "text",
				"filter": "sprintf(http://r.cnews.qq.com/getSubNewsChlidInterest?devid=860046037899335&appver=25_areading_3.3.1&chlid=%s&qn-sig=b07fe23c165c858d38f32ba972f3ccc1&qn-rid=b3f1c889-8a27-402c-9339-f87a9333546c&page={0})"
			}
		]
	}`), &pipe)

	if err != nil {
		log.Println(err)
		return
	}

	v, _ := (pipe.PipeBytes(body, "json"))
	bd, _ := json.Marshal(v)
	fmt.Println(string(bd))
}
