package systemd

import "net"

// DialJournal returns no error if the process can dial journal socket.
// Returns an error if dial failed, whichi indicates journald is not available
// (e.g. run embedded etcd as docker daemon).
// Reference: https://github.com/coreos/go-systemd/blob/master/journal/journal.go.
func DialJournal() error {
	conn, err := net.Dial("unixgram", "/run/systemd/journal/socket")
	if conn != nil {
		defer conn.Close()
	}
	return err
}
