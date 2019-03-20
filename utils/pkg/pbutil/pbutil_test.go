package pbutil

import (
	"errors"
	"reflect"
	"testing"
)

func TestMarshaler(t *testing.T) {
	data := []byte("test data")
	m := &fakeMarshaler{data: data}
	if g := MustMarshal(m); !reflect.DeepEqual(g, data) {
		t.Errorf("data = %s, want %s", g, m.data)
	}
}

func TestMarshalerPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("recover = nil, want error")
		}
	}()
	m := &fakeMarshaler{err: errors.New("blah")}
	MustMarshal(m)
}

func TestUnmarshaler(t *testing.T) {
	data := []byte("test data")
	m := &fakeUnmarshaler{}
	MustUnmarshal(m, data)
	if !reflect.DeepEqual(m.data, data) {
		t.Errorf("data = %s, want %s", m.data, data)
	}
}

func TestUnmarshalerPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("recover = nil, want error")
		}
	}()
	m := &fakeUnmarshaler{err: errors.New("blah")}
	MustUnmarshal(m, nil)
}

func TestGetBool(t *testing.T) {
	tests := []struct {
		b    *bool
		wb   bool
		wset bool
	}{
		{nil, false, false},
		{Boolp(true), true, true},
		{Boolp(false), false, true},
	}
	for i, tt := range tests {
		b, set := GetBool(tt.b)
		if b != tt.wb {
			t.Errorf("#%d: value = %v, want %v", i, b, tt.wb)
		}
		if set != tt.wset {
			t.Errorf("#%d: set = %v, want %v", i, set, tt.wset)
		}
	}
}

type fakeMarshaler struct {
	data []byte
	err  error
}

func (m *fakeMarshaler) Marshal() ([]byte, error) {
	return m.data, m.err
}

type fakeUnmarshaler struct {
	data []byte
	err  error
}

func (m *fakeUnmarshaler) Unmarshal(data []byte) error {
	m.data = data
	return m.err
}
