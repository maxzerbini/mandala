package mandala

import (
	"bytes"
	"fmt"
	"mime"
)

// EmailAddress contains an email name and address.
type EmailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (ad *EmailAddress) String() string {
	if ad.Name != "" {
		return fmt.Sprintf("%s <%s>", ad.Name, ad.Address)
	}
	return ad.Address
}

// FormatAddress formats an address and a name as a valid RFC 5322 address.
func (ad *EmailAddress) FormatAddress(charset string) string {
	if ad.Name == "" {
		return ad.Address
	}
	enc := mime.QEncoding.Encode(charset, ad.Name)
	var buf bytes.Buffer
	if enc == ad.Name {
		buf.WriteByte('"')
		for i := 0; i < len(ad.Name); i++ {
			b := ad.Name[i]
			if b == '\\' || b == '"' {
				buf.WriteByte('\\')
			}
			buf.WriteByte(b)
		}
		buf.WriteByte('"')
	} else {
		buf.WriteString(enc)
	}
	buf.WriteString(" <")
	buf.WriteString(ad.Address)
	buf.WriteByte('>')

	addr := buf.String()
	buf.Reset()
	return addr
}

// JoinAddresses produces a concatenation of email addresses
func JoinAddresses(addrs []EmailAddress) string {
	var buffer bytes.Buffer
	limit := len(addrs) - 1
	for i, val := range addrs {
		buffer.WriteString(val.String())
		if i < limit {
			buffer.WriteString(", ")
		}
	}
	return buffer.String()
}

// JoinFormattedAddresses produces a concatenation of formatted email addresses
func JoinFormattedAddresses(addrs []EmailAddress, charset string) string {
	var buffer bytes.Buffer
	limit := len(addrs) - 1
	for i, val := range addrs {
		buffer.WriteString(val.FormatAddress(charset))
		if i < limit {
			buffer.WriteString(", ")
		}
	}
	return buffer.String()
}
