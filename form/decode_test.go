package form_test

import (
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/longkai/encoding/form"
)

func TestUnpack(t *testing.T) {
	type Params struct {
		Q       string  `json:"q"`
		Int     int     `json:"int"`
		Float   float64 `json:"float"`
		Bool    bool    `json:"bool"`
		Array   []int   `json:"array"`
		Default string  `json:"default"`
	}
	testCases := []struct {
		desc        string
		url         string
		method      string
		body        string
		option      form.Option
		contentType string
		params      Params
		want        Params
	}{
		{
			desc:   "GET url encode",
			url:    `http://google.com?q=golang&int=233&float=3.14159&bool=true&array=1&array=2&array=3&default=888`,
			method: http.MethodGet,
			option: form.Query,
			want:   Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "888"},
		},
		{
			desc:   "GET default value",
			url:    `http://google.com`,
			method: http.MethodGet,
			option: form.Query,
			params: Params{Default: "2333"},
			want:   Params{Default: "2333"},
		},
		{
			desc:        "POST body encode",
			url:         `http://google.com?default=888`,
			method:      http.MethodPost,
			option:      form.Body,
			contentType: `application/x-www-form-urlencoded; charset=utf-8`,
			body:        `q=golang&int=233&float=3.14159&bool=true&array=1&array=2&array=3`,
			params:      Params{Default: "2333"},
			want:        Params{Q: "golang", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "2333"},
		},
		{
			desc:        "POST body mixes url encode",
			url:         `http://google.com?q=rust&default=888`,
			method:      http.MethodPost,
			option:      form.Mixed,
			contentType: `application/x-www-form-urlencoded; charset=utf-8`,
			body:        `q=golang&int=233&float=3.14159&bool=true&array=1&array=2&array=3&default=999`,
			params:      Params{Default: "2333"},
			want:        Params{Q: "rust", Int: 233, Float: 3.14159, Bool: true, Array: []int{1, 2, 3}, Default: "888"},
		},
	}
	for _, c := range testCases {
		t.Run(c.desc, func(t *testing.T) {
			req, err := http.NewRequest(c.method, c.url, strings.NewReader(c.body))
			if err != nil {
				t.Errorf("new request: %+v", err)
			}
			req.Header.Set("Content-Type", c.contentType)
			if err := form.UnpackWithOption(req, &c.params, c.option); err != nil {
				t.Errorf("parse: %+v", err)
			}
			if !reflect.DeepEqual(c.params, c.want) {
				t.Errorf("Marshal(%s %s %s) = %+v, want %+v", c.method, c.url, c.body, c.params, c.want)
			}
		})
	}
}

func TestUnpackMultipart(t *testing.T) {
	type model struct {
		Val   string                  `json:"hello"`
		File  *multipart.FileHeader   `json:"file"`
		File2 *multipart.FileHeader   `json:"file2"`
		Files []*multipart.FileHeader `json:"files"`
	}
	cases := []struct {
		desc         string
		UnpackOption form.Option
		body         string
		params       *model
		want         *model
	}{
		{
			desc:   "no file",
			params: &model{},
			want:   &model{Val: "world"},
			body: `------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="hello"

world
------WebKitFormBoundarykhWusB7Rx4ybHQtA--`,
		},
		{
			desc:   "single file",
			params: &model{},
			want:   &model{Val: "world", File: &multipart.FileHeader{Filename: "hello.txt", Size: int64(len("hello, world\n"))}},
			body: `------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="hello"

world
------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="file"; filename="hello.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA--`,
		},
		{
			desc:   "file array",
			params: &model{},
			want: &model{Val: "world", Files: []*multipart.FileHeader{
				&multipart.FileHeader{Filename: "hello.txt", Size: int64(len("hello, world\n"))},
				&multipart.FileHeader{Filename: "hello.txt", Size: int64(len("hello, world\n"))},
			}},
			body: `------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="hello"

world
------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="files"; filename="hello.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="files"; filename="hello.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA--`,
		},
		{
			desc:   "multiple files",
			params: &model{},
			want: &model{
				Val:   "world",
				File:  &multipart.FileHeader{Filename: "hello.txt", Size: int64(len("hello, world\n"))},
				File2: &multipart.FileHeader{Filename: "hello.txt", Size: int64(len("hello, world\n"))},
			},
			body: `------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="hello"

world
------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="file"; filename="hello.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="file2"; filename="world.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA--`,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodPost, "https://google.com/", strings.NewReader(c.body))
			if err != nil {
				t.Errorf("new request fail: %+v", err)
				return
			}
			r.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundarykhWusB7Rx4ybHQtA")
			if err := form.UnpackWithOption(r, c.params, form.Multipart); err != nil {
				t.Errorf("params.UnpackWithType(%s, %d): %+v", c.body, form.Multipart, err)
			}
			if c.params.Val != c.want.Val {
				t.Errorf("field hello got %q, want %q", c.params.Val, c.want.Val)
			}

			if !comparePart(c.params.File, c.want.File) {
				t.Errorf("part file not equal, got %+v, want %+v", c.params.File, c.want.File)
			}

			if !comparePart(c.params.File2, c.want.File2) {
				t.Errorf("part file2 not equal, got %+v, want %+v", c.params.File2, c.want.File2)
			}

			if len(c.params.Files) != len(c.want.Files) {
				t.Errorf("file len got %d, want %d", len(c.params.Files), len(c.want.Files))
				return
			}
			for i, f := range c.params.Files {
				if !comparePart(f, c.want.Files[i]) {
					t.Errorf("files[%d] not equal, got %+v, want %+v", i, f, c.want.Files[i])
				}
			}
		})
	}
}

func comparePart(part1, part2 *multipart.FileHeader) bool {
	if part1 == nil && part2 == nil {
		return true
	}
	// Simply check file name and size, enough.
	if part1.Filename != part1.Filename {
		return false
	}
	if part1.Size != part2.Size {
		return false
	}
	return true
}
