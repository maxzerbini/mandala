package mandala

/*
	Mandala is a SMTP client that supports STARTTLS authentication, SMTPUTF8 recipient and sender addresses and AMP email dynamic content.
	This work is based on the library go-smtp https://github.com/emersion/go-smtp
*/

import (
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

// Session represents a client connection to an SMTP server.
type Session struct {
	// Text is the textproto.Conn used by the Client. It is exported to allow for
	// clients to add extensions.
	Text *textproto.Conn
	// keep a reference to the connection so it can be used to create a TLS
	// connection later
	conn net.Conn
	// whether the Client is using TLS
	tls        bool
	serverName string
	// map of supported extensions
	ext map[string]string
	// supported auth mechanisms
	auth       []string
	localName  string // the name to use in HELO/EHLO
	didHello   bool   // whether we've said HELO/EHLO
	helloError error  // the error from the hello
	a          smtp.Auth
}

// SendBulkReportItem represents the outcome of a single sending.
type SendBulkReportItem struct {
	MessageID string
	Sent      bool
	Err       error
}

// SendBulkReport contains the list of the report items.
type SendBulkReport []SendBulkReportItem

// NewSession returns a new client Session connected to an SMTP server at host.
// The host must include a port, as in "mail.example.com:smtp".
func NewSession(host string, a smtp.Auth) (*Session, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}
	soloHost, _, _ := net.SplitHostPort(host)
	c, err := NewSessionUsingConnection(conn, soloHost, a)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewSessionUsingConnection returns a new Session using an existing connection and host as a
// server name to be used when authenticating.
func NewSessionUsingConnection(conn net.Conn, host string, auth smtp.Auth) (*Session, error) {
	text := textproto.NewConn(conn)
	_, _, err := text.ReadResponse(220)
	if err != nil {
		text.Close()
		return nil, err
	}
	c := &Session{Text: text, conn: conn, serverName: host, localName: "localhost", a: auth}
	return c, nil
}

// Close closes the connection.
func (c *Session) Close() error {
	return c.Text.Close()
}

// hello runs a hello exchange if needed.
func (c *Session) hello() error {
	if !c.didHello {
		c.didHello = true
		err := c.ehlo()
		if err != nil {
			c.helloError = c.helo()
		}
	}
	return c.helloError
}

// Hello sends a HELO or EHLO to the server as the given host name.
// Calling this method is only necessary if the client needs control
// over the host name used. The client will introduce itself as "localhost"
// automatically otherwise. If Hello is called, it must be called before
// any of the other methods.
func (c *Session) Hello(localName string) error {
	if c.didHello {
		return errors.New("smtp: Hello called after other methods")
	}
	c.localName = localName
	return c.hello()
}

// cmd is a convenience function that sends a command and returns the response
func (c *Session) cmd(expectCode int, format string, args ...interface{}) (int, string, error) {
	id, err := c.Text.Cmd(format, args...)
	if err != nil {
		return 0, "", err
	}
	c.Text.StartResponse(id)
	defer c.Text.EndResponse(id)
	code, msg, err := c.Text.ReadResponse(expectCode)
	return code, msg, err
}

// helo sends the HELO greeting to the server. It should be used only when the
// server does not support ehlo.
func (c *Session) helo() error {
	c.ext = nil
	_, _, err := c.cmd(250, "HELO %s", c.localName)
	return err
}

// ehlo sends the EHLO (extended hello) greeting to the server. It
// should be the preferred greeting for servers that support it.
func (c *Session) ehlo() error {
	_, msg, err := c.cmd(250, "EHLO %s", c.localName)
	if err != nil {
		return err
	}
	ext := make(map[string]string)
	extList := strings.Split(msg, "\n")
	if len(extList) > 1 {
		extList = extList[1:]
		for _, line := range extList {
			args := strings.SplitN(line, " ", 2)
			if len(args) > 1 {
				ext[args[0]] = args[1]
			} else {
				ext[args[0]] = ""
			}
		}
	}
	if mechs, ok := ext["AUTH"]; ok {
		c.auth = strings.Split(mechs, " ")
	}
	c.ext = ext
	return err
}

// StartTLS sends the STARTTLS command and encrypts all further communication.
// Only servers that advertise the STARTTLS extension support this function.
func (c *Session) StartTLS(config *tls.Config) error {
	if err := c.hello(); err != nil {
		return err
	}
	_, _, err := c.cmd(220, "STARTTLS")
	if err != nil {
		return err
	}
	c.conn = tls.Client(c.conn, config)
	c.Text = textproto.NewConn(c.conn)
	c.tls = true
	return c.ehlo()
}

// TLSConnectionState returns the client's TLS connection state.
// The return values are their zero values if StartTLS did
// not succeed.
func (c *Session) TLSConnectionState() (state tls.ConnectionState, ok bool) {
	tc, ok := c.conn.(*tls.Conn)
	if !ok {
		return
	}
	return tc.ConnectionState(), true
}

// Verify checks the validity of an email address on the server.
// If Verify returns nil, the address is valid. A non-nil return
// does not necessarily indicate an invalid address. Many servers
// will not verify addresses for security reasons.
func (c *Session) Verify(addr string) error {
	if err := c.hello(); err != nil {
		return err
	}
	_, _, err := c.cmd(250, "VRFY %s", addr)
	return err
}

// Auth authenticates a client using the provided authentication mechanism.
// A failed authentication closes the connection.
// Only servers that advertise the AUTH extension support this function.
func (c *Session) Auth() error {
	if err := c.hello(); err != nil {
		return err
	}
	encoding := base64.StdEncoding
	mech, resp, err := c.a.Start(&smtp.ServerInfo{c.serverName, c.tls, c.auth})
	if err != nil {
		c.Quit()
		return err
	}
	resp64 := make([]byte, encoding.EncodedLen(len(resp)))
	encoding.Encode(resp64, resp)
	code, msg64, err := c.cmd(0, strings.TrimSpace(fmt.Sprintf("AUTH %s %s", mech, resp64)))
	for err == nil {
		var msg []byte
		switch code {
		case 334:
			msg, err = encoding.DecodeString(msg64)
		case 235:
			// the last message isn't base64 because it isn't a challenge
			msg = []byte(msg64)
		default:
			err = &textproto.Error{Code: code, Msg: msg64}
		}
		if err == nil {
			resp, err = c.a.Next(msg, code == 334)
		}
		if err != nil {
			// abort the AUTH
			c.cmd(501, "*")
			c.Quit()
			break
		}
		if resp == nil {
			break
		}
		resp64 = make([]byte, encoding.EncodedLen(len(resp)))
		encoding.Encode(resp64, resp)
		code, msg64, err = c.cmd(0, string(resp64))
	}
	return err
}

// Mail issues a MAIL command to the server using the provided email address.
// If the server supports the 8BITMIME extension, Mail adds the BODY=8BITMIME
// parameter.
// This initiates a mail transaction and is followed by one or more Rcpt calls.
func (c *Session) Mail(from string) error {
	if err := c.hello(); err != nil {
		return err
	}
	cmdStr := "MAIL FROM:<%s>"
	if c.ext != nil {
		if _, ok := c.ext["8BITMIME"]; ok {
			cmdStr += " BODY=8BITMIME"
		}
	}
	_, _, err := c.cmd(250, cmdStr, from)
	return err
}

// Rcpt issues a RCPT command to the server using the provided email address.
// A call to Rcpt must be preceded by a call to Mail and may be followed by
// a Data call or another Rcpt call.
func (c *Session) Rcpt(to string) error {
	_, _, err := c.cmd(25, "RCPT TO:<%s>", to)
	return err
}

type dataCloser struct {
	c *Session
	io.WriteCloser
}

func (d *dataCloser) Close() error {
	d.WriteCloser.Close()
	_, _, err := d.c.Text.ReadResponse(250)
	return err
}

// Data issues a DATA command to the server and returns a writer that
// can be used to write the mail headers and body. The caller should
// close the writer before calling any more methods on c. A call to
// Data must be preceded by one or more calls to Rcpt.
func (c *Session) Data() (io.WriteCloser, error) {
	_, _, err := c.cmd(354, "DATA")
	if err != nil {
		return nil, err
	}
	return &dataCloser{c, c.Text.DotWriter()}, nil
}

var testHookStartTLS func(*tls.Config) // nil, except for tests

// Extension reports whether an extension is support by the server.
// The extension name is case-insensitive. If the extension is supported,
// Extension also returns a string that contains any parameters the
// server specifies for the extension.
func (c *Session) Extension(ext string) (bool, string) {
	if err := c.hello(); err != nil {
		return false, ""
	}
	if c.ext == nil {
		return false, ""
	}
	ext = strings.ToUpper(ext)
	param, ok := c.ext[ext]
	return ok, param
}

// Reset sends the RSET command to the server, aborting the current mail
// transaction.
func (c *Session) Reset() error {
	if err := c.hello(); err != nil {
		return err
	}
	_, _, err := c.cmd(250, "RSET")
	return err
}

// Quit sends the QUIT command and closes the connection to the server.
func (c *Session) Quit() error {
	if err := c.hello(); err != nil {
		return err
	}
	_, _, err := c.cmd(221, "QUIT")
	if err != nil {
		return err
	}
	return c.Text.Close()
}

// MailAndRcpt issues MAIL FROM and RCPT TO commands, in sequence.
// It will check the addresses, decide if SMTPUTF8 is needed, and apply the
// necessary transformations.
// If the message ReturnPath is setted, it will be used as MAIL FROM address.
// If the message Recipient is setted, it will be used ad the only RCPT TO address.
func (c *Session) MailAndRcpt(msg *Email) error {
	recipients := make([]string, 0)
	var fromNeeds bool
	var from string
	var err error
	var toNeeds bool
	// prepare MAIL FROM
	if msg.ReturnPath != "" {
		from, fromNeeds, err = c.prepareForSMTPUTF8(msg.ReturnPath)
		if err != nil {
			return err
		}
	} else {
		from, fromNeeds, err = c.prepareForSMTPUTF8(msg.From.Address)
		if err != nil {
			return err
		}
	}
	// prepare RCPT TO
	if msg.Recipient != "" {
		// if the Recipient is setted than use only it
		to, needs, err := c.prepareForSMTPUTF8(msg.Recipient)
		if err != nil {
			return err
		}
		recipients = append(recipients, to)
		toNeeds = toNeeds || needs
	} else {
		for _, rec := range msg.To {
			to, needs, err := c.prepareForSMTPUTF8(rec.Address)
			if err != nil {
				return err
			}
			recipients = append(recipients, to)
			toNeeds = toNeeds || needs
		}
		for _, rec := range msg.Cc {
			to, needs, err := c.prepareForSMTPUTF8(rec.Address)
			if err != nil {
				return err
			}
			recipients = append(recipients, to)
			toNeeds = toNeeds || needs
		}
		for _, rec := range msg.Bcc {
			to, needs, err := c.prepareForSMTPUTF8(rec.Address)
			if err != nil {
				return err
			}
			recipients = append(recipients, to)
			toNeeds = toNeeds || needs
		}
	}

	smtputf8Needed := fromNeeds || toNeeds

	cmdStr := "MAIL FROM:<%s>"
	if ok, _ := c.Extension("8BITMIME"); ok {
		cmdStr += " BODY=8BITMIME"
	}
	if smtputf8Needed {
		cmdStr += " SMTPUTF8"
	}
	_, _, err = c.cmd(250, cmdStr, from)
	if err != nil {
		return err
	}
	for _, to := range recipients {
		_, _, err = c.cmd(25, "RCPT TO:<%s>", to)
		if err != nil {
			return err
		}
	}
	return nil
}

// StartSession opens an SMTP session.
func (c *Session) StartSession() error {
	if err := c.hello(); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: c.serverName}
		if testHookStartTLS != nil {
			testHookStartTLS(config)
		}
		if err := c.StartTLS(config); err != nil {
			return err
		}
	}
	if c.a != nil && c.ext != nil {
		if _, ok := c.ext["AUTH"]; ok {
			if err := c.Auth(); err != nil {
				return err
			}
		}
	}
	return nil
}

// SendSingleMessage sends a single email to the recipient.
// The method requires that the session is open and leaves it open.
func (c *Session) SendSingleMessage(msg *Email) error {
	if msg.From.Address == "" {
		return errors.New("From address can not be empty")
	}
	if msg.Recipient == "" && len(msg.To) == 0 && len(msg.Cc) == 0 && len(msg.Bcc) == 0 {
		return errors.New("Recipient addresses can not be empty")
	}
	if err := c.MailAndRcpt(msg); err != nil {
		c.Reset()
		return err
	}
	w, err := c.Data()
	if err != nil {
		c.Reset()
		return err
	}
	err = msg.Write(w)
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return nil
}

// SendMessageBulk sends a list of messages to an SMTP server using the same connection and at the end closes the session and the connection.
func (c *Session) SendMessageBulk(messages []*Email) (SendBulkReport, error) {
	report := make(SendBulkReport, 0)
	defer c.Close()
	err := c.StartSession()
	if err != nil {
		return report, err
	}
	for _, msg := range messages {
		err := c.SendSingleMessage(msg)
		report = append(report, SendBulkReportItem{MessageID: msg.MessageID, Sent: err == nil, Err: err})
	}
	return report, c.Quit()
}

// prepareForSMTPUTF8 prepares the address for SMTPUTF8.
// It returns:
//  - The address to use. It is based on addr, and possibly modified to make
//    it not need the extension, if the server does not support it.
//  - Whether the address needs the extension or not.
//  - An error if the address needs the extension, but the client does not
//    support it.
func (c *Session) prepareForSMTPUTF8(addr string) (string, bool, error) {
	// ASCII address pass through.
	if isASCII(addr) {
		return addr, false, nil
	}

	// Non-ASCII address also pass through if the server supports the
	// extension.
	// Note there's a chance the server wants the domain in IDNA anyway, but
	// it could also require it to be UTF8. We assume that if it supports
	// SMTPUTF8 then it knows what its doing.
	if ok, _ := c.Extension("SMTPUTF8"); ok {
		return addr, true, nil
	}

	// Something is not ASCII, and the server does not support SMTPUTF8:
	//  - If it's the local part, there's no way out and is required.
	//  - If it's the domain, use IDNA.
	user, domain := Split(addr)

	if !isASCII(user) {
		return addr, true, &textproto.Error{Code: 599,
			Msg: "local part is not ASCII but server does not support SMTPUTF8"}
	}

	// If it's only the domain, convert to IDNA and move on.
	domain, err := idna.ToASCII(domain)
	if err != nil {
		// The domain is not IDNA compliant, which is odd.
		// Fail with a permanent error, not ideal but this should not
		// happen.
		return addr, true, &textproto.Error{
			Code: 599, Msg: "non-ASCII domain is not IDNA safe"}
	}

	return user + "@" + domain, false, nil
}

// isASCII returns true if all the characters in s are ASCII, false otherwise.
func isASCII(s string) bool {
	for _, c := range s {
		if c > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// IsPermanent returns true if the error is permanent, and false otherwise.
// If it can't tell, it returns false.
func IsPermanent(err error) bool {
	terr, ok := err.(*textproto.Error)
	if !ok {
		return false
	}
	// Error codes 5yz are permanent.
	// https://tools.ietf.org/html/rfc5321#section-4.2.1
	if terr.Code >= 500 && terr.Code < 600 {
		return true
	}
	return false
}

// SendMail connects to the server at addr, switches to TLS if
// possible, authenticates with the optional mechanism a if possible,
// and then sends an email from address from, to addresses to, with
// message msg.
// The addr must include a port, as in "mail.example.com:smtp".
func SendMail(addr string, a smtp.Auth, msg *Email) error {
	c, err := NewSession(addr, a)
	if err != nil {
		return err
	}
	defer c.Close()
	err = c.StartSession()
	if err != nil {
		return err
	}
	c.SendSingleMessage(msg)
	return c.Quit()
}
