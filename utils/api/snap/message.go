package snap

import (
	"io"

	"github.com/fearblackcat/swiftRaft/raft/raftpb"
	"github.com/fearblackcat/swiftRaft/utils/pkg/ioutil"
)

// Message is a struct that contains a raft Message and a ReadCloser. The type
// of raft message MUST be MsgSnap, which contains the raft meta-data and an
// additional data []byte field that contains the snapshot of the actual state
// machine.
// Message contains the ReadCloser field for handling large snapshot. This avoid
// copying the entire snapshot into a byte array, which consumes a lot of memory.
//
// User of Message should close the Message after sending it.
type Message struct {
	raftpb.Message
	ReadCloser io.ReadCloser
	TotalSize  int64
	closeC     chan bool
}

func NewMessage(rs raftpb.Message, rc io.ReadCloser, rcSize int64) *Message {
	return &Message{
		Message:    rs,
		ReadCloser: ioutil.NewExactReadCloser(rc, rcSize),
		TotalSize:  int64(rs.Size()) + rcSize,
		closeC:     make(chan bool, 1),
	}
}

// CloseNotify returns a channel that receives a single value
// when the message sent is finished. true indicates the sent
// is successful.
func (m Message) CloseNotify() <-chan bool {
	return m.closeC
}

func (m Message) CloseWithError(err error) {
	if cerr := m.ReadCloser.Close(); cerr != nil {
		err = cerr
	}
	if err == nil {
		m.closeC <- true
	} else {
		m.closeC <- false
	}
}
