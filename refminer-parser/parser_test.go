package parser

import (
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestMoveMethod_Parse(t *testing.T) {
	type fields struct {
		From string
		To   string
	}
	type args struct {
		detail string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mm := &MoveMethod{
				From: tt.fields.From,
				To:   tt.fields.To,
			}
			mm.Parse(tt.args.detail)
		})
	}
}

func TestParse(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name             string
		args             args
		wantRefactorings []interface{}
		wantErr          bool
	}{
		{
			"move method",
			args{strings.NewReader(`CommitId;RefactoringType;RefactoringDetail
0b34f587ee5e19c24b29a9c3e26374607ff05e8f;Move Method;Move Method        public mostrarQuadroDeCargos(qc QuadroDeCargosVO) : void from class br.mil.eb.cds.sisdot.qc.util.QuadroDeCargosUtil to public mostrarQuadroDeCargos(qc QuadroDeCargosVO) : void from class br.mil.eb.cds.sisdot.qc.modelo.vo.qc.FracaoQcVO`)},
			[]interface{}{&MoveMethod{
				"br.mil.eb.cds.sisdot.qc.util.QuadroDeCargosUtil.mostrarQuadroDeCargos(QuadroDeCargosVO)",
				"br.mil.eb.cds.sisdot.qc.modelo.vo.qc.FracaoQcVO",
			}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRefactorings, err := Parse(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRefactorings, tt.wantRefactorings) {
				t.Errorf("Parse() = %v, want %v", gotRefactorings, tt.wantRefactorings)
			}
		})
	}
}
