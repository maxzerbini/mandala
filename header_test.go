package mandala

import (
	"bytes"
	"testing"
)

func TestHeaders_Write(t *testing.T) {
	type args struct {
		charset string
	}
	h1 := Headers{}
	h1 = append(h1, &Header{Name: "Content-Type", Value: "text/html", Encoded: true})
	h2 := Headers{&Header{Name: "X-Very-Long", Value: "ùaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaddddddddddddddddddddddddddddddddddddffffff ffffffffffffffffffffffffffffffffffff", Encoded: true}}
	tests := []struct {
		name    string
		h       Headers
		args    args
		wantW   string
		wantErr bool
	}{
		{name: "test one header", h: h1, args: args{charset: "UTF-8"}, wantW: "Content-Type: text/html\r\n\r\n", wantErr: false},
		{name: "test very long header", h: h2, args: args{charset: "UTF-8"}, wantW: "X-Very-Long: =?UTF-8?q?=C3=B9aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaddddddddddddddddddddddddddd?= =?UTF-8?q?dddddddddffffff_ffffffffffffffffffffffffffffffffffff?=\r\n\r\n", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := tt.h.Write(w, tt.args.charset); (err != nil) != tt.wantErr {
				t.Errorf("Headers.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Headers.Write() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
