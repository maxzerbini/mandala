package mandala

import (
	"mime/multipart"
	"os"
	"testing"
)

func TestPart_WriteMultipart(t *testing.T) {
	type args struct {
		w *multipart.Writer
	}
	pt := &Part{
		ContentType: "text/plain",
		CharSet:     "utf-8",
		Encoding:    "quoted-printable",
		Body:        []byte("Hello worllllld!!!\nGo编程语言\n"),
	}
	tests := []struct {
		name    string
		a       *Part
		args    args
		wantErr bool
	}{
		{name: "text plain part", a: pt, args: args{w: multipart.NewWriter(os.Stdout)}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.a.WriteMultipart(tt.args.w); (err != nil) != tt.wantErr {
				t.Errorf("Part.Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
