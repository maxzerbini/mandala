package mandala

import (
	"net/smtp"
	"testing"
)

func TestSendMail(t *testing.T) {
	type args struct {
		addr string
		a    smtp.Auth
		msg  *Email
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendMail(tt.args.addr, tt.args.a, tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("SendMail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
