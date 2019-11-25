package mandala

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"

	"mime"

	"github.com/kennygrant/sanitize"
	"github.com/nats-io/nuid"
)

// Email rapresents an email message.
type Email struct {
	From        EmailAddress   `json:"from"`
	To          []EmailAddress `json:"to"`
	Cc          []EmailAddress `json:"cc"`
	Bcc         []EmailAddress `json:"bcc"`
	Subject     string         `json:"subject"`
	Headers     Headers        `json:"headers"` // Extended SMTP headers
	Text        string         `json:"text"`
	HTML        string         `json:"html"`
	AMP         string         `json:"amp"`
	Encoding    string         `json:"encoding"` // "quoted-printable", "base64", "8bit"
	CharSet     string         `json:"charset"`  // "UTF-8", "iso-8859-1", ...
	MessageID   string         `json:"message_id"`
	ReplyTo     EmailAddress   `json:"reply_to"`
	Recipient   string         `json:"recipient"`
	ReturnPath  string         `json:"returnpath"`
	Sender      string         `json:"sender"`
	Attachments []*Part        `json:"attachments"`
	Images      []*Part        `json:"images"`
	Sanitize    bool           `json:"sanitize"`
}

// NewEmail creates a new email message using default settings.
func NewEmail(from EmailAddress, to []EmailAddress, subject, html, text string) *Email {
	e := new(Email)
	e.From = from
	e.To = to
	e.Subject = subject
	e.HTML = html
	e.Text = text
	e.Encoding = "quoted-printable"
	e.CharSet = "utf-8"
	e.Attachments = make([]*Part, 0)
	e.Images = make([]*Part, 0)
	return e
}

// Write the headers for the email to the specified writer.
func (e *Email) writeHeaders(w io.Writer, boundary string) error {
	// check MEssage-Id
	if e.MessageID == "" {
		_, domain := Split(e.From.Address)
		e.MessageID = fmt.Sprintf("%s@%s", nuid.Next(), domain)
	}
	headers := Headers{}
	headers = headers.Add("Message-Id", fmt.Sprintf("<%s>", e.MessageID), false)
	headers = headers.Add("From", e.From.FormatAddress(e.CharSet), false)
	if len(e.To) > 0 {
		headers = headers.Add("To", JoinFormattedAddresses(e.To, e.CharSet), false)
	}
	if len(e.Cc) > 0 {
		headers = headers.Add("Cc", JoinFormattedAddresses(e.Cc, e.CharSet), false)
	}
	if e.ReplyTo.Address != "" {
		headers = headers.Add("Reply-To", e.ReplyTo.FormatAddress(e.CharSet), false)
	}
	if e.Sender != "" {
		headers = headers.Add("Sender", e.Sender, false)
	}
	headers = headers.Add("Subject", e.Subject, true)
	headers = headers.Add("Date", time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"), false)
	headers = headers.Add("MIME-Version", "1.0", false)
	if e.IsMultiPart() {
		headers = headers.Add("Content-Type", fmt.Sprintf("%s; boundary=%s", e.ContentType(), boundary), false)
	} else {
		headers = headers.Add("Content-Type", fmt.Sprintf("%s; charset=\"%s\"", e.ContentType(), e.CharSet), false)
		headers = headers.Add("Content-Transfer-Encoding", e.Encoding, false)
	}
	// add extended headers
	headers = headers.AddHeaders(e.Headers)
	err := headers.Write(w, e.CharSet)
	return err
}

// Write the body of the email to the specified writer.
func (e *Email) writeBody(w *multipart.Writer) error {
	var (
		buff      = &bytes.Buffer{}
		altWriter = multipart.NewWriter(buff)
	)
	p, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type": []string{
			fmt.Sprintf("multipart/alternative; boundary=%s", altWriter.Boundary()),
		},
	})
	if err != nil {
		return err
	}
	if e.Text == "" && e.Sanitize {
		e.Text = sanitize.HTML(e.HTML)
	}
	pt := &Part{
		ContentType: "text/plain",
		CharSet:     e.CharSet,
		Encoding:    e.Encoding,
		Body:        []byte(e.Text),
	}
	if err := pt.WriteMultipart(altWriter); err != nil {
		return err
	}
	if e.HTML == "" {
		e.HTML = e.Text
	}
	ph := &Part{
		ContentType: "text/html",
		CharSet:     e.CharSet,
		Encoding:    e.Encoding,
		Body:        []byte(e.HTML),
	}
	if err := ph.WriteMultipart(altWriter); err != nil {
		return err
	}
	if err := altWriter.Close(); err != nil {
		return err
	}
	if _, err := io.Copy(p, buff); err != nil {
		return err
	}
	return nil
}

// Write the body of the email to the specified writer.
func (e *Email) writeAMPBody(w *multipart.Writer) error {
	if e.Text == "" && e.Sanitize {
		if e.HTML != "" {
			e.Text = sanitize.HTML(e.HTML)
		}
	}
	if e.Text != "" {
		pt := &Part{
			ContentType: "text/plain",
			CharSet:     e.CharSet,
			Encoding:    e.Encoding,
			Body:        []byte(e.Text),
		}
		if err := pt.WriteMultipart(w); err != nil {
			return err
		}
	}
	pamp := &Part{
		ContentType: "text/x-amp-html",
		CharSet:     e.CharSet,
		Encoding:    e.Encoding,
		Body:        []byte(e.AMP),
	}
	if err := pamp.WriteMultipart(w); err != nil {
		return err
	}
	if e.HTML == "" {
		e.HTML = e.Text
	}
	ph := &Part{
		ContentType: "text/html",
		CharSet:     e.CharSet,
		Encoding:    e.Encoding,
		Body:        []byte(e.HTML),
	}
	if err := ph.WriteMultipart(w); err != nil {
		return err
	}
	return nil
}

func (e *Email) Write(w io.Writer) error {
	if e.IsMultiPart() {
		if e.AMP != "" {
			// Multipart alternative AMP message
			mpWriter := multipart.NewWriter(w)
			if err := e.writeHeaders(w, mpWriter.Boundary()); err != nil {
				return err
			}
			if err := e.writeAMPBody(mpWriter); err != nil {
				return err
			}
			if err := mpWriter.Close(); err != nil {
				return err
			}
		} else {
			// Multipart mixed message
			mpWriter := multipart.NewWriter(w)
			if err := e.writeHeaders(w, mpWriter.Boundary()); err != nil {
				return err
			}
			if err := e.writeBody(mpWriter); err != nil {
				return err
			}
			for _, att := range e.Attachments {
				if err := att.WriteMultipart(mpWriter); err != nil {
					return err
				}
			}
			for _, img := range e.Images {
				if err := img.WriteMultipart(mpWriter); err != nil {
					return err
				}
			}
			if err := mpWriter.Close(); err != nil {
				return err
			}
		}
	} else {
		// Simple message
		if err := e.writeHeaders(w, ""); err != nil {
			return err
		}
		var body []byte
		if e.Text != "" {
			body = []byte(e.Text)
		} else {
			body = []byte(e.HTML)
		}
		if err := WriteEncodedBody(w, body, e.Encoding); err != nil {
			return err
		}
	}
	return nil
}

// IsMultiPart detects if the message is multipart.
func (e *Email) IsMultiPart() bool {
	if e.AMP != "" {
		return true
	}
	if len(e.Attachments) == 0 && len(e.Images) == 0 {
		if e.Text == "" && e.HTML != "" {
			return false
		}
		if e.HTML == "" && e.Text != "" {
			return false
		}
	}
	return true
}

// ContentType detects the message content type.
func (e *Email) ContentType() string {
	if e.IsMultiPart() {
		if e.AMP != "" {
			return "multipart/alternative"
		}
		return "multipart/mixed"
	} else if e.HTML == "" {
		return "text/plain"
	}
	return "text/html"
}

// AddAttachment adds an attachment to the message.
func (e *Email) AddAttachment(filename, contentType string, body []byte) {
	part := new(Part)
	part.Body = body
	part.ContentDisposition = "attachment"
	part.Encoding = "base64"
	part.ContentType = contentType
	part.Filename = filename
	e.Attachments = append(e.Attachments, part)
}

// AddEmbeddedImage adds an embedded image to the message.
func (e *Email) AddEmbeddedImage(filename, contentType, contentID string, body []byte) {
	part := new(Part)
	part.Body = body
	part.Encoding = "base64"
	part.ContentType = contentType
	part.ContentID = contentID
	part.Filename = filename
	e.Images = append(e.Images, part)
}

// LoadAttachment attachs a file to the message.
func (e *Email) LoadAttachment(path string) error {
	//os.op
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	name := filepath.Base(path)
	ext := filepath.Ext(path)
	if ext != "" {
		e.AddAttachment(name, mime.TypeByExtension(ext), file)
	} else {
		e.AddAttachment(name, "application/octet-stream", file)
	}
	return nil
}

// Split an user@domain address into user and domain.
func Split(addr string) (string, string) {
	ps := strings.SplitN(addr, "@", 2)
	if len(ps) != 2 {
		return addr, ""
	}
	return ps[0], ps[1]
}
