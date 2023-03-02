package fs

import "testing"

func TestGetFreeSpace(t *testing.T) {
	if GetFreeSpace("/tmp") <= 0 {
		t.Fatalf("Expected free space of /tmp to be greater than 0")
	}
}
