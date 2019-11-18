package form_test

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/longkai/encoding/form"
)

func ExampleUnpack() {
	form.FieldTag = "http" // the Default key is `json`, you can change it.
	// Populate a request, grabbing one via ServerHTTP in server side usually.
	r, err := http.NewRequest(http.MethodGet, "https://google.com/search?q=golang&int=1&float=3.14&bool=true&array=1&array=2&array=3&camelCase=123", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	var params struct {
		Q         string  `http:"q"`
		Int       string  `http:"int"`
		Float     float64 `http:"float"`
		Bool      bool    `http:"bool"`
		Array     []int   `http:"array"`
		CamelCase int     // camelCase=xxx
	}
	params.Q = "hello" // Set default value for fields.
	if err := form.UnpackWithOption(r, &params, form.Query); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", params)
	// Output: {Q:golang Int:1 Float:3.14 Bool:true Array:[1 2 3] CamelCase:123}
}

func ExampleUnpackWithOption() {
	form.FieldTag = "json"
	fileUploadBody := `------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="hello"

world
------WebKitFormBoundarykhWusB7Rx4ybHQtA
Content-Disposition: form-data; name="file"; filename="hello.txt"
Content-Type: text/plain

hello, world

------WebKitFormBoundarykhWusB7Rx4ybHQtA--`

	// Simulate a multipart fileupload request.
	r, err := http.NewRequest(http.MethodPost, "https://google.com/upload?q=golang", strings.NewReader(fileUploadBody))
	if err != nil {
		fmt.Println(err)
		return
	}
	r.Header.Set("Content-Type", "multipart/form-data; boundary=----WebKitFormBoundarykhWusB7Rx4ybHQtA")

	var params struct {
		Val  string                `json:"hello"`
		File *multipart.FileHeader `json:"file"`
		// Other key/val, file or file array.
	}
	if err := form.UnpackWithOption(r, &params, form.Multipart); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Val: %v, file name: %v, file size: %d", params.Val, params.File.Filename, params.File.Size)
	// Output: Val: world, file name: hello.txt, file size: 13
}
