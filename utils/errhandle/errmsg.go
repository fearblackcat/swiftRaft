package errhandle

import "fmt"

var Msg = map[int]string{
	// raft
	E_LEADER_LOST:           "leader is lost",
	E_REMOVE_RAFT_NODE_FAIL: "remove raft node failed",
	E_ELECTION_ERROR:        "exit the election",
	E_LEADER_DOWN:           "leader down error",
}

func FormatCode(code int) string {
	return fmt.Sprintf("errno=%v||errmsg=%s", code, Msg[code])
}
