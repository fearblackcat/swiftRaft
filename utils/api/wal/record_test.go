package wal

import (
	"bytes"
	"hash/crc32"
	"io"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/fearblackcat/swiftRaft/utils/api/wal/walpb"
)

var (
	infoData   = []byte("\b\xef\xfd\x02")
	infoRecord = append([]byte("\x0e\x00\x00\x00\x00\x00\x00\x00\b\x01\x10\x99\xb5\xe4\xd0\x03\x1a\x04"), infoData...)
)

func TestReadRecord(t *testing.T) {
	badInfoRecord := make([]byte, len(infoRecord))
	copy(badInfoRecord, infoRecord)
	badInfoRecord[len(badInfoRecord)-1] = 'a'

	tests := []struct {
		data []byte
		wr   *walpb.Record
		we   error
	}{
		{infoRecord, &walpb.Record{Type: 1, Crc: crc32.Checksum(infoData, crcTable), Data: infoData}, nil},
		{[]byte(""), &walpb.Record{}, io.EOF},
		{infoRecord[:8], &walpb.Record{}, io.ErrUnexpectedEOF},
		{infoRecord[:len(infoRecord)-len(infoData)-8], &walpb.Record{}, io.ErrUnexpectedEOF},
		{infoRecord[:len(infoRecord)-len(infoData)], &walpb.Record{}, io.ErrUnexpectedEOF},
		{infoRecord[:len(infoRecord)-8], &walpb.Record{}, io.ErrUnexpectedEOF},
		{badInfoRecord, &walpb.Record{}, walpb.ErrCRCMismatch},
	}

	rec := &walpb.Record{}
	for i, tt := range tests {
		buf := bytes.NewBuffer(tt.data)
		decoder := newDecoder(ioutil.NopCloser(buf))
		e := decoder.decode(rec)
		if !reflect.DeepEqual(rec, tt.wr) {
			t.Errorf("#%d: block = %v, want %v", i, rec, tt.wr)
		}
		if !reflect.DeepEqual(e, tt.we) {
			t.Errorf("#%d: err = %v, want %v", i, e, tt.we)
		}
		rec = &walpb.Record{}
	}
}

func TestWriteRecord(t *testing.T) {
	b := &walpb.Record{}
	typ := int64(0xABCD)
	d := []byte("Hello world!")
	buf := new(bytes.Buffer)
	e := newEncoder(buf, 0, 0)
	e.encode(&walpb.Record{Type: typ, Data: d})
	e.flush()
	decoder := newDecoder(ioutil.NopCloser(buf))
	err := decoder.decode(b)
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if b.Type != typ {
		t.Errorf("type = %d, want %d", b.Type, typ)
	}
	if !reflect.DeepEqual(b.Data, d) {
		t.Errorf("data = %v, want %v", b.Data, d)
	}
}
