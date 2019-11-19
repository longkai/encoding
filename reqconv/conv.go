/*
Package reqconv implements auto parsing a HTTP request as golang struct according to its content type.
The supported content type are:

	- application/json
	- application/xml
	- multipart/form-data
	- application/x-www-form-urlencoded

For a request without body, e.g., GET, DELETE, HEAD, TRACE, it will parse the URL query into given pointer.

It returns a error when other types incoming.

As of Golang struct, the supported types are:

	- int
	- bool
	- string
	- float64
	- *multipart.FileHeader
	- slice of above

For example, a file upload request:

	var params struct {
		Int   int                   `json:"int"`
		Bool  bool                  `json:"bool"`
		File  *multipart.FileHeader `json:"file"`
		Array []string              `json:"array"`
	}

	params.Int = 233 // default value if any.
	err := reqconv.Unmarshal(req, &params)

By default, the form data key uses `json`, same as json for consistency.
You can change it by:

	form.FieldTag = "form"

If no tag specified, it will use cammel case of the field name since most languages fields start with lower case.

As of xml, however, you must use the `xml` tag.
*/
package reqconv

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"

	"github.com/longkai/encoding/form"
)

// Unmarshal auto parses a HTTP request r into ptr according to its content type.
func Unmarshal(r *http.Request, ptr interface{}) error {
	// If the request has no body, we could only parse the URL query.
	switch r.Method {
	// Which method MUST NOT have body? See https://tools.ietf.org/html/rfc7231#section-4.3
	case http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodTrace:
		return form.UnpackWithOption(r, ptr, form.Query)
	}

	ct := r.Header.Get("Content-Type")
	if ct == "" {
		// RFC 7231, section 3.1.1.5 - empty type
		//   MAY be treated as application/octet-stream
		ct = "application/octet-stream"
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return fmt.Errorf("parse request media type: %v", err)
	}

	switch mediaType {
	case "application/json":
		err = unmarshal(r, ptr, json.Unmarshal)
	case "application/xml":
		err = unmarshal(r, ptr, xml.Unmarshal)
	case "multipart/form-data":
		err = form.UnpackWithOption(r, ptr, form.Multipart)
	case "application/x-www-form-urlencoded":
		err = form.UnpackWithOption(r, ptr, form.Body)
	default:
		return fmt.Errorf("unsupported content type: %s", ct)
	}
	// Register other types parser? Unlikely, since almost commom media types are above.

	if err != nil {
		return fmt.Errorf("parse request body as %s: %v", mediaType, err)
	}
	return nil
}

func unmarshal(r *http.Request, ptr interface{}, unmarshaler func(b []byte, ptr interface{}) error) error {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body.Close()
	// Reset body since caller may read it for some reasons later.
	r.Body = ioutil.NopCloser(bytes.NewReader(b))
	return unmarshaler(b, ptr)
}
