package mandala

import "testing"

func TestEmailAddress_String(t *testing.T) {
	tests := []struct {
		name string
		ad   *EmailAddress
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ad.String(); got != tt.want {
				t.Errorf("EmailAddress.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinAddresses(t *testing.T) {
	type args struct {
		addrs []EmailAddress
	}
	tests := []struct {
		name string
		args args
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinAddresses(tt.args.addrs); got != tt.want {
				t.Errorf("JoinAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmailAddress_FormatAddress(t *testing.T) {
	ad1 := &EmailAddress{Name: "快乐的时光 Chao Lee", Address: "chao.lee@baidu.com"}
	type args struct {
		charset string
	}
	tests := []struct {
		name string
		ad   *EmailAddress
		args args
		want string
	}{
		{name: "", ad: ad1, args: args{charset: "utf-8"},
			want: "=?utf-8?q?=E5=BF=AB=E4=B9=90=E7=9A=84=E6=97=B6=E5=85=89_Chao_Lee?= <chao.lee@baidu.com>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ad.FormatAddress(tt.args.charset); got != tt.want {
				t.Errorf("EmailAddress.FormatAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}
