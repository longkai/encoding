package reqconv_test

import (
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/longkai/encoding/form"
	"github.com/longkai/encoding/reqconv"
)

func TestUnmarshal(t *testing.T) {
	defer func(tag string) { form.FieldTag = tag }(form.FieldTag)
	form.FieldTag = "form"

	type Params struct {
		Q       string  `json:"q" xml:"q" form:"q"`
		Int     int     `json:"int" xml:"int" form:"int"`
		Float   float64 `http:"float" xml:"float" form:"float"`
		Bool    bool    `http:"bool" xml:"bool" form:"bool"`
		Array   []int   `http:"array" xml:"array" form:"array"`
		Default string  `json:"default" xml:"default" form:"default"`
	}
	testCases := []struct {
		desc        string
		url         string
		method      string
		body        string
		contentType string
		params      Params
		want        Params
	}{
		{
			desc:   "GET url encode",
			url:    `http://google.com?q=golang&int=233&float=3.14159&bool=true&array=1&array=2&array=3&default=888`,
			method: http.MethodGet,
			want:   Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "888"},
		},
		{
			desc:   "GET default value",
			url:    `http://google.com`,
			method: http.MethodGet,
			params: Params{Default: "2333"},
			want:   Params{Default: "2333"},
		},
		{
			desc:        "POST body encode",
			url:         `http://google.com`,
			method:      http.MethodPost,
			contentType: `application/x-www-form-urlencoded; charset=utf-8`,
			body:        `q=golang&int=233&float=3.14159&bool=true&array=1&array=2&array=3&default=888`,
			params:      Params{Default: "2333"},
			want:        Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "888"},
		},
		{
			desc:        "POST body+url encode",
			url:         `http://google.com?q=rust&default=888`,
			method:      http.MethodPost,
			contentType: `application/x-www-form-urlencoded; charset=utf-8`,
			body:        `q=golang&int=233&float=3.14159&bool=true&array=1&array=2&array=3`,
			params:      Params{Default: "2333"},
			want:        Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "2333"},
		},
		{
			desc:        "POST json",
			url:         `http://google.com`,
			method:      http.MethodPost,
			contentType: `application/json; charset=utf-8`,
			body:        `{"q": "golang", "int": 233, "float": 3.14159, "bool": true, "array": [1, 2, 3], "default": "2333"}`,
			params:      Params{},
			want:        Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "2333"},
		},
		{
			desc:        "POST xml",
			url:         `http://google.com`,
			method:      http.MethodPost,
			contentType: `application/xml; charset=utf-8`,
			body: `<xml>
	<q>golang</q>
	<int>233</int>
	<float>3.14159</float>
	<bool>true</bool>
	<array>1</array>
	<array>2</array>
	<array>3</array>
	<default>2333</default>
</xml>`,
			params: Params{},
			want:   Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "2333"},
		},
		{
			desc:        "POST json with default value",
			url:         `http://google.com`,
			method:      http.MethodPost,
			contentType: `application/json; charset=utf-8`,
			body:        `{}`,
			params:      Params{Default: "2333"},
			want:        Params{Default: "2333"},
		},
		{
			desc:        "non utf-8 encoding",
			url:         `http://google.com?q=golang`,
			method:      http.MethodPost,
			contentType: `application/json; charset=gbk`,
			body:        `{"q": "ÄãºÃ, hello"}`, // 你好, hello
			params:      Params{},
			want:        Params{Q: "ÄãºÃ, hello"}, // golang assume input encoding is utf-8.
		},
	}
	for _, c := range testCases {
		t.Run(c.desc, func(t *testing.T) {
			req, err := http.NewRequest(c.method, c.url, strings.NewReader(c.body))
			if err != nil {
				t.Errorf("new request: %+v", err)
			}
			req.Header.Set("Content-Type", c.contentType)
			if err := reqconv.Unmarshal(req, &c.params); err != nil {
				t.Errorf("parse: %+v", err)
			}
			if !reflect.DeepEqual(c.params, c.want) {
				t.Errorf("Marshal(%s %s %s) = %+v, want %+v", c.method, c.url, c.body, c.params, c.want)
			}
		})
	}
}

func TestMultiparUnmarshal(t *testing.T) {
	type params struct {
		Val  string                `json:"hello"`
		File *multipart.FileHeader `json:"file"`
	}
	cases := []struct {
		desc        string
		body        string
		contentType string
		ptr, want   *params
	}{
		{
			desc: "normal",
			body: `------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="hello"

world
------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="file"; filename="hello.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA--`,
			contentType: "multipart/form-data; boundary=----WebKitFormBoundarykhWusB7Rx4ybHQtA",
			ptr:         &params{},
			want:        &params{Val: "world", File: &multipart.FileHeader{Filename: "hello.txt", Size: int64(len("hello, world\n"))}},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodPost, "https://google.com", strings.NewReader(c.body))
			if err != nil {
				t.Errorf("new request: %+v", err)
				return
			}
			r.Header.Set("Content-Type", c.contentType)
			if err := reqconv.Unmarshal(r, c.ptr); err != nil {
				t.Errorf("reqconv.Unmarshal: %+v", err)
				return
			}
			if c.ptr.Val != c.want.Val || c.ptr.File.Filename != c.want.File.Filename || c.ptr.File.Size != c.want.File.Size {
				t.Errorf("got %+v, want %+v", c.ptr, c.want)
			}
		})
	}
}

func TestUnmarshalUnsupportedType(t *testing.T) {
	cases := []struct {
		desc        string
		contentType string
	}{
		{
			desc:        "tencent image",
			contentType: "image/vnd.tencent.tap",
		},
		{
			desc:        "js",
			contentType: "application/javascript",
		},
		{
			desc:        "empty",
			contentType: "",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "https://google.com/", strings.NewReader(`...`))
			if err != nil {
				t.Errorf("new request: %+v", err)
				return
			}
			req.Header.Set("Content-Type", c.contentType)
			var ptr interface{}
			if err := reqconv.Unmarshal(req, ptr); err == nil {
				t.Errorf("Unmarshal content type %s err != nil", c.contentType)
			}
		})
	}
}
