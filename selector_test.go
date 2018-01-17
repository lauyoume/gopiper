package gopiper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"testing"

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
							"filter": "unixtime"
						},
						{
							"name": "nowmill",
							"filter": "unixmill"
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

func TestHtmlDouban(t *testing.T) {

	pb := []byte(`
{
	"type": "map",
	"selector": "",
	"subitem": [
		{
			"type": "string",
			"selector": "title",
			"name": "name",
			"filter": "trimspace|replace((豆瓣))|trim( )"
		},
		{
			"type": "string",
			"selector": "#content .gtleft a.bn-sharing//attr[data-type]",
			"name": "fenlei"
		},
		{
			"type": "string",
			"selector": "#content .gtleft a.bn-sharing//attr[data-pic]",
			"name": "thumbnail"
		},
		{
			"type": "string-array",
			"selector": "#info span.attrs a[rel=v\\:directedBy]",
			"name": "direct"
		},
		{
			"type": "string-array",
			"selector": "#info span a[rel=v\\:starring]",
			"name": "starring"
		},
		{
			"type": "string-array",
			"selector": "#info span[property=v\\:genre]",
			"name": "type"
		},
		{
			"type": "string-array",
			"selector": "#related-pic .related-pic-bd a:not(.related-pic-video) img//attr[src]",
			"name": "imgs",
			"filter": "join($)|replace(albumicon,photo)|split($)"
		},
		{
			"type": "string-array",
			"selector": "#info span[property=v\\:initialReleaseDate]",
			"name": "releasetime"
		},
		{
			"type": "string",
			"selector": "regexp:<span class=\"pl\">单集片长:</span> ([\\w\\W]+?)<br/>",
			"name": "longtime"
		},
		{
			"type": "string",
			"selector": "regexp:<span class=\"pl\">制片国家/地区:</span> ([\\w\\W]+?)<br/>",
			"name": "country",
			"filter": "split(/)|trimspace"
		},
		{
			"type": "string",
			"selector": "regexp:<span class=\"pl\">语言:</span> ([\\w\\W]+?)<br/>",
			"name": "language",
			"filter": "split(/)|trimspace"
		},
		{
			"type": "int",
			"selector": "regexp:<span class=\"pl\">集数:</span> (\\d+)<br/>",
			"name": "episode"
		},
		{
			"type": "string",
			"selector": "regexp:<span class=\"pl\">又名:</span> ([\\w\\W]+?)<br/>",
			"name": "alias",
			"filter": "split(/)|trimspace"
		},
		{
			"type": "string",
			"selector": "#link-report span.hidden, #link-report span[property=v\\:summary]|last",
			"name": "brief",
			"filter": "trimspace|split(\n)|trimspace|wraphtml(p)|join"
		},
		{
			"type": "float",
			"selector": "#interest_sectl .rating_num",
			"name": "score"
		},
		{
			"type": "string",
			"selector": "#content h1 span.year",
			"name": "year",
			"filter": "replace(()|replace())|intval"
		},
		{
			"type": "string",
			"selector": "#comments-section > .mod-hd h2 a",
			"name": "comment",
			"filter": "replace(全部)|replace(条)|trimspace|intval"
		}
	]
}
`)

	log.Println(callFilter("美团他|女神||", `preadd(AAAA)|split(|)|join(,)`))
	if val, err := test_piper("http://movie.douban.com/subject/25850640/", "html", pb); err != nil {
		t.Fatal(err)
	} else {
		showjson(val)
	}

	if val, err := test_piper("http://movie.douban.com/subject/2035218/?from=tag_all", "html", pb); err != nil {
		t.Fatal(err)
	} else {
		showjson(val)
	}

}

func test_piper(u string, tp string, pb []byte, headers ...string) (interface{}, error) {
	pipe := PipeItem{}
	err := json.Unmarshal(pb, &pipe)

	if err != nil {
		return nil, err
	}

	req := gohttp.New()
	req.Get(u)
	for idx := 0; idx < len(headers); idx += 2 {
		req.Set(headers[idx], headers[idx+1])
	}
	req.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36")
	body, _, err := req.Bytes()
	if err != nil {
		return nil, err
	}

	return pipe.PipeBytes(body, tp)
}

func showjson(val interface{}) {
	bd, _ := json.MarshalIndent(val, "", "    ")
	fmt.Println(string(bd))
}

func TestBaidu(t *testing.T) {
	pb := []byte(`
		{
			"type": "jsonparse",
			"selector": "regexp:runtime\\.modsData\\.userData = (\\{(?:.+?)\\});",
			"subitem": [
				{
					"type": "json",
					"selector": "user"
				}
			]
		}
	`)
	val, err := test_piper("https://author.baidu.com/profile?context={%22app_id%22:%221567569757829059%22}&cmdType=&pagelets[]=root&reqID=0&ispeed=1", "text", pb, "Cookie", "BAIDUID=D0FB1501E11F72B20AEC00CED2C220D5:FG=1")
	if err != nil {
		t.Fatal(err)
		return
	} else {
		showjson(val)
	}
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
