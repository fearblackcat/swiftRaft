package types

import (
	"reflect"
	"sort"
	"testing"
)

func TestIDString(t *testing.T) {
	tests := []struct {
		input ID
		want  string
	}{
		{
			input: 12,
			want:  "c",
		},
		{
			input: 4918257920282737594,
			want:  "444129853c343bba",
		},
	}

	for i, tt := range tests {
		got := tt.input.String()
		if tt.want != got {
			t.Errorf("#%d: ID.String failure: want=%v, got=%v", i, tt.want, got)
		}
	}
}

func TestIDFromString(t *testing.T) {
	tests := []struct {
		input string
		want  ID
	}{
		{
			input: "17",
			want:  23,
		},
		{
			input: "612840dae127353",
			want:  437557308098245459,
		},
	}

	for i, tt := range tests {
		got, err := IDFromString(tt.input)
		if err != nil {
			t.Errorf("#%d: IDFromString failure: err=%v", i, err)
			continue
		}
		if tt.want != got {
			t.Errorf("#%d: IDFromString failure: want=%v, got=%v", i, tt.want, got)
		}
	}
}

func TestIDFromStringFail(t *testing.T) {
	tests := []string{
		"",
		"XXX",
		"612840dae127353612840dae127353",
	}

	for i, tt := range tests {
		_, err := IDFromString(tt)
		if err == nil {
			t.Fatalf("#%d: IDFromString expected error, but err=nil", i)
		}
	}
}

func TestIDSlice(t *testing.T) {
	g := []ID{10, 500, 5, 1, 100, 25}
	w := []ID{1, 5, 10, 25, 100, 500}
	sort.Sort(IDSlice(g))
	if !reflect.DeepEqual(g, w) {
		t.Errorf("slice after sort = %#v, want %#v", g, w)
	}
}
