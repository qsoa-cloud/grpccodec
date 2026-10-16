package inittest

import (
	"errors"
	"testing"
)

func TestVerdict(t *testing.T) {
	for _, c := range []struct {
		out  string
		err  error
		fail bool
	}{
		{"PASS\n", nil, false},
		{"2026/10/16 proto: duplicate proto type registered: FileReq\nPASS\n", nil, true},
		{"panic: proto: duplicate enum registered: FileResp_Error_Type\n", errors.New("exit status 2"), true},
		{"proto: file \"dfs.proto\" is already registered\n", nil, true},
		{"", errors.New("exit status 1"), true},
	} {
		if got := verdict([]byte(c.out), c.err) != nil; got != c.fail {
			t.Errorf("verdict(%q, %v) failed=%v, want %v", c.out, c.err, got, c.fail)
		}
	}
}

func TestCheck(t *testing.T) { Check(t) }
