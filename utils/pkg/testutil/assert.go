package testutil

import (
	"fmt"
	"reflect"
	"testing"
)

func AssertEqual(t *testing.T, e, a interface{}, msg ...string) {
	if (e == nil || a == nil) && (isNil(e) && isNil(a)) {
		return
	}
	if reflect.DeepEqual(e, a) {
		return
	}
	s := ""
	if len(msg) > 1 {
		s = msg[0] + ": "
	}
	s = fmt.Sprintf("%sexpected %+v, got %+v", s, e, a)
	FatalStack(t, s)
}

func AssertNil(t *testing.T, v interface{}) {
	AssertEqual(t, nil, v)
}

func AssertNotNil(t *testing.T, v interface{}) {
	if v == nil {
		t.Fatalf("expected non-nil, got %+v", v)
	}
}

func AssertTrue(t *testing.T, v bool, msg ...string) {
	AssertEqual(t, true, v, msg...)
}

func AssertFalse(t *testing.T, v bool, msg ...string) {
	AssertEqual(t, false, v, msg...)
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() != reflect.Struct && rv.IsNil()
}
