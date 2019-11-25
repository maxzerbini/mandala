package mandala

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

const (
	amp_message = `<!doctype html><html amp4email>
<head>
<meta charset="utf-8">
<style amp4email-boilerplate>body{visibility:hidden}</style>
<script async src="https://cdn.ampproject.org/v0.js"></script>
</head>
<body>
Hello World in AMP!
</body>
</html>`
	amp_message2 = `<!doctype html>
	<html ⚡4email>
	<head>
	  <meta charset="utf-8">
	  <script async src="https://cdn.ampproject.org/v0.js"></script>
	  <script async custom-element="amp-list" src="https://cdn.ampproject.org/v0/amp-list-0.1.js"></script>
	  <script async custom-template="amp-mustache" src="https://cdn.ampproject.org/v0/amp-mustache-0.2.js"></script>
	  <style amp4email-boilerplate>body{visibility:hidden}</style>
	  <style amp-custom>
		.products {
		  display: block;
		  height: 100px;
		  box-shadow: 0 10px 20px rgba(0,0,0,0.19), 0 6px 6px rgba(0,0,0,0.23);
		  background: #fff;
		  border-radius: 2px;
		  margin-bottom: 15px;
		  position: relative;
		}
	
		.products amp-img {
		  float: left;
		  margin-right: 16px;
		}
	  </style>
	</head>
	<body>
	  You should see <b>6</b> fruits with pictures, names, stars, and prices.
	  <amp-list id="amp-list-placeholder" noloading width="auto"
				height="1000"
				layout="fixed-height" src="https://amp.gmail.dev/playground/public/ssr_amp_list">
		<div placeholder>
		  <ul class="results">
			 <li></li><li></li><li></li><li></li><li></li>
		  </ul>
		</div>
		<template type="amp-mustache">
			<div class="products">
				<amp-img width="150"
					   height="100"
					   alt="{{name}}"
					   src="{{img}}"></amp-img>
				<p class="name">{{name}}</p>
				<p class="star">{{{stars}}}</p>
				<p class="price">{{price}}</p>
			</div>
		</template>
	  </amp-list>
	
	  You should now only see <b>2</b> fruits with pictures, names, stars, and prices because I specify "max-items" = 2.
	
	  <amp-list id="amp-list-placeholder" noloading width="auto"
				height="600"
				max-items=2
				layout="fixed-height" src="https://amp.gmail.dev/playground/public/ssr_amp_list">
		<div placeholder>
		  <ul class="results">
			 <li></li><li></li><li></li><li></li><li></li>
		  </ul>
		</div>
		<template type="amp-mustache">
			<div class="products">
				<amp-img width="150"
					   height="100"
					   alt="{{name}}"
					   src="{{img}}"></amp-img>
				<p class="name">{{name}}</p>
				<p class="star">{{{stars}}}</p>
				<p class="price">{{price}}</p>
			</div>
		</template>
	  </amp-list>
	
	  You should now only see <b>pear and banana</b> with pictures, names, stars, and prices because I specify a different path by setting "items".
	
	  <amp-list id="amp-list-placeholder" noloading width="auto"
				height="600"
				items="part_of_them.pear_and_banana"
				layout="fixed-height" src="https://amp.gmail.dev/playground/public/ssr_amp_list">
		<div placeholder>
		  <ul class="results">
			 <li></li><li></li><li></li><li></li><li></li>
		  </ul>
		</div>
		<template type="amp-mustache">
			<div class="products">
				<amp-img width="150"
					   height="100"
					   alt="{{name}}"
					   src="{{img}}"></amp-img>
				<p class="name">{{name}}</p>
				<p class="star">{{{stars}}}</p>
				<p class="price">{{price}}</p>
			</div>
		</template>
	  </amp-list>
	</body>
	</html>`
)

func TestEmailMessage_Write(t *testing.T) {
	msg1 := &Email{
		MessageID: "1",
		From:      EmailAddress{Address: "sender@gmail.com", Name: "Jack Sender"},
		To:        []EmailAddress{EmailAddress{Address: "recipient@gmail.com", Name: "John Receiver"}, EmailAddress{Address: "recipient@yahoo.com", Name: "Luke Yahoo"}},
		Subject:   "Hello world 编程",
		Text:      "Hello world!!!\nGo编程语言\n",
		CharSet:   "utf-8",
		Encoding:  "quoted-printable",
		ReplyTo:   EmailAddress{Address: "replyto@gmail.com", Name: "Jack ReplyTo"},
	}
	msg2 := &Email{
		MessageID: "2",
		From:      EmailAddress{Address: "sender@gmail.com", Name: "Jack Sender"},
		To:        []EmailAddress{EmailAddress{Address: "recipient@gmail.com", Name: "John Receiver"}, EmailAddress{Address: "recipient@yahoo.com", Name: "Luke Yahoo"}},
		Subject:   "Hello world 编程 पाठ या कोई वेबसाइट के पते के अनुवाद या किसी दस्तावेज़ का अनुवाद।",
		Text:      "Hello world!!!\nGo编程语言",
		CharSet:   "utf-8",
		Encoding:  "base64",
		ReplyTo:   EmailAddress{Address: "replyto@gmail.com", Name: "Jack दस्तावेज़"},
	}
	msg3 := &Email{
		MessageID: "3",
		From:      EmailAddress{Address: "sender@gmail.com", Name: "Jack Sender"},
		To:        []EmailAddress{EmailAddress{Address: "recipient@gmail.com", Name: "John Receiver"}, EmailAddress{Address: "recipient@yahoo.com", Name: "Luke Yahoo"}},
		Subject:   "Hello world 编程",
		Text:      "Hello world!!!\nGo编程语言\n",
		HTML:      "<html><head></head><body><h1>Hello world!!</h1><table></table></body></html>",
		CharSet:   "utf-8",
		Encoding:  "quoted-printable",
		ReplyTo:   EmailAddress{Address: "replyto@gmail.com", Name: "编程语言 Lee"},
	}
	msg3.Headers = msg3.Headers.Add("X-my-personal-header-no-enc", "<òàè+ù;6789@pippo.com>", false)
	msg3.Headers = msg3.Headers.Add("X-my-personal-header-enc", "<òàè+ù;6789@pippo.com>", true)
	msg4 := &Email{
		MessageID: "4",
		From:      EmailAddress{Address: "sender@gmail.com", Name: "Jack Sender"},
		To:        []EmailAddress{EmailAddress{Address: "recipient@gmail.com", Name: "John Receiver"}, EmailAddress{Address: "recipient@yahoo.com", Name: "Luke Yahoo"}},
		Subject:   "Hello world 编程",
		Text:      "Hello world!!!\nGo编程语言\n",
		HTML:      "<html><head></head><body><h1>Hello world!!</h1><h1>编程 编程 编程</h1><table></table></body></html>",
		CharSet:   "utf-8",
		Encoding:  "quoted-printable",
		ReplyTo:   EmailAddress{Address: "replyto@gmail.com", Name: "Jack ReplyTo"},
	}
	msg4.Attachments = make([]*Part, 1)
	msg4.Attachments[0] = &Part{Filename: "text.txt", ContentDisposition: "attachment", ContentType: "text/plain", Encoding: "base64", Body: []byte("Text text text")}
	msg5 := &Email{
		MessageID: "5",
		From:      EmailAddress{Address: "sender@gmail.com", Name: "这是电子邮件Jack Sender"},
		To:        []EmailAddress{EmailAddress{Address: "recipient@gmail.com", Name: "这是电子邮件需要交 John Receiver"}, EmailAddress{Address: "recipient@yahoo.com", Name: "Luke Yahoo"}},
		Subject:   "Hello world 编程",
		HTML:      "<html><head></head><body><h1>Hello world!!</h1><table></table></body></html>",
		CharSet:   "utf-8",
		Encoding:  "quoted-printable",
		ReplyTo:   EmailAddress{Address: "replyto@gmail.com", Name: "这是电子邮件需要交付的地址 ReplyTo"},
	}
	msg6 := &Email{
		MessageID: "5",
		From:      EmailAddress{Address: "sender@gmail.com", Name: "这是电子邮件Jack Sender"},
		To:        []EmailAddress{EmailAddress{Address: "recipient@gmail.com", Name: "这是电子邮件需要交 John Receiver"}, EmailAddress{Address: "recipient@yahoo.com", Name: "Luke Yahoo"}},
		Subject:   "Hello world 编程",
		HTML:      "<html><head></head><body><h1>Hello world!!</h1><table></table></body></html>",
		AMP:       amp_message,
		CharSet:   "utf-8",
		Encoding:  "quoted-printable",
		ReplyTo:   EmailAddress{Address: "replyto@gmail.com", Name: "这是电子邮件需要交付的地址 ReplyTo"},
	}
	tests := []struct {
		name     string
		e        *Email
		wantErr  bool
		snippets []string
	}{
		{name: "Text Message in quoted-printable", e: msg1, wantErr: false,
			snippets: []string{"From: \"Jack Sender\" <sender@gmail.com>", "To: \"John Receiver\" <recipient@gmail.com>, \"Luke Yahoo\" <recipient@yahoo.com>", "Content-Type: text/plain; charset=\"utf-8\""}},
		{name: "Text Message in base64", e: msg2, wantErr: false,
			snippets: []string{"SGVsbG8gd29ybGQhISEKR2/nvJbnqIvor63oqIA=", "Content-Transfer-Encoding: base64", "Content-Type: text/plain; charset=\"utf-8\""}},
		{name: "Text and HTML Message in quoted-printable", e: msg3, wantErr: false,
			snippets: []string{"Content-Type: multipart/mixed; boundary=", "Content-Type: text/html; charset=\"utf-8\"", "Content-Type: text/plain; charset=\"utf-8\"",
				"X-my-personal-header-enc: =?utf-8?q?<=C3=B2=C3=A0=C3=A8+=C3=B9;6789@pippo.com>?=", "X-my-personal-header-no-enc: <òàè+ù;6789@pippo.com>"}},
		{name: "Text and HTML Message in quoted-printable with attachment", e: msg4, wantErr: false,
			snippets: []string{"Content-Transfer-Encoding: base64", "Content-Type: text/plain; name=text.txt"}},
		{name: "HTML Message in quoted-printable", e: msg5, wantErr: false,
			snippets: []string{"=?utf-8?q?=E8=BF=99=E6=98=AF=E7=94=B5=E5=AD=90=E9=82=AE=E4=BB=B6Jack_Send?= =?utf-8?q?er?= <sender@gmail.com>", "Content-Transfer-Encoding: quoted-printable", "<html><head></head><body><h1>Hello world!!</h1><table></table></body></html="}},
		{name: "AMP and HTML Message in quoted-printable", e: msg6, wantErr: false,
			snippets: []string{"Content-Type: multipart/alternative; boundary=", "Content-Type: text/x-amp-html; charset=\"utf-8\"", "Content-Type: text/html; charset=\"utf-8\""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrintMessage(t, tt.e, tt.wantErr, tt.snippets)
		})
	}
}

func TestEmailMessage_AddEmbeddedImage(t *testing.T) {
	msg := NewEmail(EmailAddress{Address: "test@test.com"}, []EmailAddress{EmailAddress{Address: "test2@test.com"}}, "Embedded Images!", "<html><body><h1>Hello</h1></body></html>", "Hello world!")
	type args struct {
		filename    string
		contentType string
		contentID   string
		body        []byte
	}
	tests := []struct {
		name     string
		e        *Email
		args     args
		snippets []string
	}{
		{name: "Add text file", e: msg, args: args{filename: "immagine.jpg", contentType: "image/jpg", contentID: "IMG001", body: []byte("09091234567890987654334567890")},
			snippets: []string{"Content-Id: <IMG001>", "MDkwOTEyMzQ1Njc4OTA5ODc2NTQzMzQ1Njc4OTA=", "Content-Transfer-Encoding: base64", "Content-Type: image/jpg; name=immagine.jpg"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.e.AddEmbeddedImage(tt.args.filename, tt.args.contentType, tt.args.contentID, tt.args.body)
			PrintMessage(t, tt.e, false, tt.snippets)
		})
	}
}

func TestEmailMessage_AddAttachment(t *testing.T) {
	msg := NewEmail(EmailAddress{Address: "test@test.com"}, []EmailAddress{EmailAddress{Address: "test2@test.com"}}, "Attachment!", "<html><body><h1>Hello</h1></body></html>", "Hello world!")
	type args struct {
		filename    string
		contentType string
		body        []byte
	}
	tests := []struct {
		name     string
		e        *Email
		args     args
		snippets []string
	}{
		{name: "Add text file", e: msg, args: args{filename: "testo.txt", contentType: "text/plain", body: []byte("Hello world files!")},
			snippets: []string{"Content-Disposition: attachment; filename=testo.txt; size=18", "SGVsbG8gd29ybGQgZmlsZXMh", "Content-Type: text/plain; name=testo.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.e.AddAttachment(tt.args.filename, tt.args.contentType, tt.args.body)
			PrintMessage(t, tt.e, false, tt.snippets)
		})
	}
}

func TestNewEmailMessage(t *testing.T) {
	type args struct {
		from    EmailAddress
		to      []EmailAddress
		subject string
		html    string
		text    string
	}
	tests := []struct {
		name     string
		args     args
		snippets []string
	}{
		{name: "text and htm", args: args{from: EmailAddress{Address: "test@test.com"}, to: []EmailAddress{EmailAddress{Address: "test2@test.com", Name: "Max"}}, html: "<html><body><h1>Hello</h1></body></html>", text: "Hello world!", subject: "Hello guy!"},
			snippets: []string{"Subject: Hello guy!", "Content-Type: text/plain; charset=\"utf-8\"", "Content-Transfer-Encoding: quoted-printable", "To: \"Max\" <test2@test.com>"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmail(tt.args.from, tt.args.to, tt.args.subject, tt.args.html, tt.args.text); got == nil {
				t.Errorf("NewEmailMessage() nil")
			} else {
				PrintMessage(t, got, false, tt.snippets)
			}
		})
	}
}

func TestEmailMessage_LoadAttachment(t *testing.T) {
	msg := NewEmail(EmailAddress{Address: "test@test.com"}, []EmailAddress{EmailAddress{Address: "test2@test.com"}}, "Load Attachment!", "<html><body><h1>Hello</h1></body></html>", "Hello world!")
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		e       *Email
		args    args
		wantErr bool
	}{
		{name: "load txt", e: msg, args: args{path: "./test/test3.txt"}, wantErr: false},
		{name: "load pdf", e: msg, args: args{path: "./test/test2.pdf"}, wantErr: false},
		{name: "load docx", e: msg, args: args{path: "./test/test1.docx"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.e.LoadAttachment(tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("EmailMessage.LoadAttachment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	for _, att := range msg.Attachments {
		fmt.Printf("Filename: %s - Content-Type: %s - Disposition: %s - Encoding: %s\n", att.Filename, att.ContentType, att.ContentDisposition, att.Encoding)
	}
}

func PrintMessage(t *testing.T, e *Email, wantErr bool, snippets []string) {
	w := bytes.NewBuffer([]byte{})
	if err := e.Write(w); (err != nil) != wantErr {
		t.Errorf("EmailMessage.Write() error = %v, wantErr %v", err, wantErr)
		return
	}
	fmt.Printf("## Start Message\n%s\n", w.String())
	fmt.Printf("## End Message\n\n")
	if snippets != nil {
		FindSnippets(t, w.String(), snippets)
	}
}

func FindSnippets(t *testing.T, msg string, snippets []string) {
	for _, snippet := range snippets {
		if !strings.Contains(msg, snippet) {
			t.Errorf("Snippet '%s' not found", snippet)
		}
	}
}
