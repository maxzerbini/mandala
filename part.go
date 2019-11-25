package mandala

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
)

// Part is used for including file and embedded image.
type Part struct {
	Filename           string `json:"filename"`
	ContentType        string `json:"content_type"`
	ContentDisposition string `json:"content_disposition"` // "attachment"
	Encoding           string `json:"encoding"`            // "quoted-printable", "base64", "8bit"
	CharSet            string `json:"charset"`             // "utf-8", "iso-8859-1", ...
	ContentID          string `json:"content_id"`
	Body               []byte `json:"body"`
}

// WriteMultipart writes the attachment to the specified multipart writer.
func (a *Part) WriteMultipart(w *multipart.Writer) error {
	a.Encoding = a.contentType()
	headers := make(textproto.MIMEHeader)
	if a.Filename != "" {
		headers.Add("Content-Type", fmt.Sprintf("%s; name=%s", a.ContentType, a.Filename))
	} else {
		headers.Add("Content-Type", a.ContentType)
	}
	if a.CharSet != "" {
		headers.Set("Content-Type", fmt.Sprintf("%s; charset=\"%s\"", headers.Get("Content-Type"), a.CharSet))
	}
	if a.ContentDisposition != "" {
		headers.Add("Content-Disposition", fmt.Sprintf("%s; filename=%s; size=%d", a.ContentDisposition, a.Filename, len(a.Body)))
	}
	headers.Add("Content-Transfer-Encoding", a.Encoding)
	if a.ContentID != "" {
		headers.Add("Content-ID", fmt.Sprintf("<%s>", a.ContentID))
	}
	p, err := w.CreatePart(headers)
	if err != nil {
		return err
	}
	return WriteEncodedBody(p, a.Body, a.Encoding)
}

// Get one of the available content-type.
func (a *Part) contentType() string {
	switch strings.ToLower(a.Encoding) {
	case "base64":
		return "base64"
	case "8bit":
		return "8bit"
	default:
		return "quoted-printable"
	}
}

// WriteEncodedBody writes the body in encoded format.
func WriteEncodedBody(p io.Writer, body []byte, encoding string) (err error) {
	switch encoding {
	case "base64":
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(body)))
		base64.StdEncoding.Encode(encoded, body)
		if _, err := p.Write(encoded); err != nil {
			return err
		}
	case "8bit":
		if _, err := p.Write(body); err != nil {
			return err
		}
	case "quoted-printable":
		fallthrough
	default: // "quoted-printable"
		q := quotedprintable.NewWriter(p)
		if _, err := q.Write(body); err != nil {
			return err
		}
		if err := q.Close(); err != nil {
			return err
		}
	}
	if _, err := p.Write([]byte("\r\n")); err != nil {
		return err
	}
	return nil
}
