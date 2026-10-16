// Package inittest checks that a binary's package initialization is free of protobuf registration conflicts.
package inittest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

var conflicts = [][]byte{
	[]byte("duplicate proto type registered"),
	[]byte("duplicate enum registered"),
	[]byte("is already registered"),
	[]byte("registration conflict"),
}

// Check re-runs the current test binary without tests and fails t if initialization panics or reports a protobuf registration conflict.
func Check(t *testing.T) {
	out, err := exec.Command(os.Args[0], "-test.run=^$").CombinedOutput()
	if err := verdict(out, err); err != nil {
		t.Fatal(err)
	}
}

func verdict(out []byte, err error) error {
	if err != nil {
		return fmt.Errorf("init failed: %v\n%s", err, out)
	}
	for _, c := range conflicts {
		if bytes.Contains(out, c) {
			return fmt.Errorf("protobuf registration conflict:\n%s", out)
		}
	}
	return nil
}
