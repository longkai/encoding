// Package form implements decoding HTTP form data and file upload as Golang struct.
package form

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

// Option pack option.
type Option int

const (
	// Body only parses the request body.
	Body Option = iota
	// Query only parses the request URL query.
	Query
	// Multipart like Body but counts multipart files in.
	Multipart
	// Mixed mixes the request body and URL query, note the Query has higher priority if same key found.
	// It's existed for compatability only.
	Mixed
	// MixedMultipart mixes Multipart and Query, Query has higher priority.
	// It's existed for compatability only.
	MixedMultipart
)

// MultipartMaxMemory the up to a total of maxMemory bytes of its file parts are stored in memory.
// See http.Request.ParseMultipartForm for more information.
// Default to 10M.
var MultipartMaxMemory int64 = 10 * 1024

// FieldTag is the default tag key.
var FieldTag = "json"

var fileHeaderPtrType = reflect.TypeOf(&multipart.FileHeader{})

// Unpack populates the fields of the struct pointed to by ptr
// from the HTTP request body in r.
func Unpack(r *http.Request, ptr interface{}) error {
	return UnpackWithOption(r, ptr, Body)
}

// UnpackWithOption populates the fields of the struct pointed to by ptr
// from the HTTP request parameters in r with the given unpack option.
func UnpackWithOption(r *http.Request, ptr interface{}, option Option) error {
	var err error
	if option == Multipart || option == MixedMultipart {
		err = r.ParseMultipartForm(MultipartMaxMemory)
	} else { // Otherwise treat all as application/x-www-form-urlencoded type.
		err = r.ParseForm()
	}
	if err != nil {
		return err
	}
	// Build map of fields keyed by effective name.
	fields := make(map[string]reflect.Value)
	v := reflect.ValueOf(ptr).Elem() // the struct variable
	for i := 0; i < v.NumField(); i++ {
		fieldInfo := v.Type().Field(i) // a reflect.StructField
		tag := fieldInfo.Tag           // a reflect.StructTag
		name := tag.Get(FieldTag)
		if name == "" {
			// First letter to lower since most languages will style that way.
			for i := range fieldInfo.Name {
				name = strings.ToLower(fieldInfo.Name[:i+1]) + fieldInfo.Name[i+1:]
				break
			}
		}
		fields[name] = v.Field(i)
	}

	switch option {
	case Query:
		return unpack(fields, r.URL.Query())
	default:
		fallthrough
	case Body:
		return unpack(fields, r.PostForm)
	case Mixed:
		return unpack(fields, r.Form)
	case Multipart:
		err = unpack(fields, r.PostForm)
	case MixedMultipart:
		err = unpack(fields, r.Form)
	}
	// Contine handle parsing multipart.
	if err != nil {
		return err
	}
	return unpackMultipart(fields, r.MultipartForm.File)
}

func unpack(fields map[string]reflect.Value, form map[string][]string) error {
	// Update struct field for each parameter in the request.
	for name, values := range form {
		f := fields[name]
		if !f.IsValid() {
			continue // ignore unrecognized HTTP parameters
		}
		for _, value := range values {
			if f.Kind() == reflect.Slice {
				elem := reflect.New(f.Type().Elem()).Elem()
				if err := populate(elem, value); err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}
				f.Set(reflect.Append(f, elem))
			} else {
				if err := populate(f, value); err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}
			}
		}
	}
	return nil
}

func unpackMultipart(fields map[string]reflect.Value, m map[string][]*multipart.FileHeader) error {
	for name, parts := range m {
		f := fields[name]
		if !f.IsValid() {
			continue // ignore unrecognized HTTP parameters
		}
		for _, part := range parts {
			if f.Kind() == reflect.Slice {
				elem := reflect.New(f.Type().Elem()).Elem()
				if err := populatePart(elem, part); err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}
				f.Set(reflect.Append(f, elem))
			} else {
				if err := populatePart(f, part); err != nil {
					return fmt.Errorf("%s: %v", name, err)
				}
			}
		}
	}
	return nil
}

func populatePart(v reflect.Value, part *multipart.FileHeader) error {
	if fileHeaderPtrType != v.Type() {
		return fmt.Errorf("unsupported multipart kind %s", v.Kind())
	}
	v.Set(reflect.ValueOf(part))
	return nil
}

func populate(v reflect.Value, value string) error {
	switch v.Kind() {
	case reflect.String:
		v.SetString(value)
	case reflect.Int:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		v.SetInt(i)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		v.SetBool(b)
	case reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		v.SetFloat(f)
	default:
		return fmt.Errorf("unsupported kind %s", v.Type())
	}
	return nil
}
