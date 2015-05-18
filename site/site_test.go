package site

import "testing"

func TestListCmd(t *testing.T) {
	s := Site{Dir: "/foo"}
	expected := "cls --date --time-style='%F %T %z %Z' /foo"
	listCmd := s.ListCmd()
	if listCmd.Script != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, listCmd.Script)
	}
}
