package flagopts

import "testing"

func TestStringArray_Set(t *testing.T) {
	tests := []struct {
		name    string
		s       *StringArray
		args    string
		wantErr bool
	}{
		{
			name: "simple",
			args: "hello",
			s:    &StringArray{"hello"},
		},
		{
			name: "empty",
			args: "",
			s:    &StringArray{},
		},
		{
			name: "few of them",
			args: "hello,world",
			s:    &StringArray{"hello", "world"},
		},
		{
			name: "with gaps",
			args: "hello,,,world",
			s:    &StringArray{"hello", "world"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.s.Set(tt.args); (err != nil) != tt.wantErr {
				t.Errorf("StringArray.Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
