package mandala

import (
	"fmt"
	"io"
	"mime"
	"strings"
)

// Header contains SMTP or MIME header.
type Header struct {
	Name    string
	Value   string
	Encoded bool
}

// Headers is a list of email headers.
type Headers []*Header

// Write the headers to the specified io.Writer following RFC 2047
func (h Headers) Write(w io.Writer, charset string) error {
	for _, v := range h {
		var header string
		if v.Encoded {
			header = fmt.Sprintf("%s: %s\r\n", v.Name, mime.QEncoding.Encode(charset, v.Value))
		} else {
			header = fmt.Sprintf("%s: %s\r\n", v.Name, v.Value)
		}
		if _, err := w.Write([]byte(header)); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("\r\n"))
	return err
}

// Add an header to the list
func (h Headers) Add(name string, value string, encoded bool) Headers {
	return append(h, &Header{Name: name, Value: value, Encoded: encoded})
}

// AddHeaders adds a list of headers
func (h Headers) AddHeaders(headers Headers) Headers {
	if headers != nil {
		return append(h, headers...)
	}
	return h
}

// GetHeader returns the first occurrance of the header with this name or nil if not found.
func (h Headers) GetHeader(name string) *Header {
	for _, val := range h {
		if strings.ToLower(val.Name) == strings.ToLower(name) {
			return val
		}
	}
	return nil
}

// GetHeaders returns all the occurrence of the header with this name.
func (h Headers) GetHeaders(name string) []*Header {
	headers := make([]*Header, 0)
	for _, val := range h {
		if strings.ToLower(val.Name) == strings.ToLower(name) {
			headers = append(headers, val)
		}
	}
	return headers
}
