package raft

import (
	"fmt"

	pb "github.com/fearblackcat/swiftRaft/raft/raftpb"
)

type Status struct {
	ID uint64

	pb.HardState
	SoftState

	Applied  uint64
	Progress map[uint64]Progress

	LeadTransferee uint64
}

// getStatus gets a copy of the current raft status.
func getStatus(r *raft) Status {
	s := Status{
		ID:             r.id,
		LeadTransferee: r.leadTransferee,
	}

	s.HardState = r.hardState()
	s.SoftState = *r.softState()

	s.Applied = r.raftLog.applied

	if s.RaftState == StateLeader {
		s.Progress = make(map[uint64]Progress)
		for id, p := range r.prs {
			s.Progress[id] = *p
		}

		for id, p := range r.learnerPrs {
			s.Progress[id] = *p
		}
	}

	return s
}

// MarshalJSON translates the raft status into JSON.
// TODO: try to simplify this by introducing ID type into raft
func (s Status) MarshalJSON() ([]byte, error) {
	j := fmt.Sprintf(`{"id":"%x","term":%d,"vote":"%x","commit":%d,"lead":"%x","raftState":%q,"applied":%d,"progress":{`,
		s.ID, s.Term, s.Vote, s.Commit, s.Lead, s.RaftState, s.Applied)

	if len(s.Progress) == 0 {
		j += "},"
	} else {
		for k, v := range s.Progress {
			subj := fmt.Sprintf(`"%x":{"match":%d,"next":%d,"state":%q},`, k, v.Match, v.Next, v.State)
			j += subj
		}
		// remove the trailing ","
		j = j[:len(j)-1] + "},"
	}

	j += fmt.Sprintf(`"leadtransferee":"%x"}`, s.LeadTransferee)
	return []byte(j), nil
}

func (s Status) String() string {
	b, err := s.MarshalJSON()
	if err != nil {
		raftLogger.Panicf("unexpected error: %v", err)
	}
	return string(b)
}
