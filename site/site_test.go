package site

import (
	"reflect"
	"testing"
)

func TestListCmd(t *testing.T) {
	s := Site{
		Client: Client{LftpPath: "lftp"},
		Name:   "bar",
		Dir:    "/foo",
	}
	expected := []string{"-e", "cls --date --time-style='%F %T %z %Z' /foo && exit", "bar"}
	listCmd := s.listCmd()
	if !reflect.DeepEqual(expected, listCmd.Args) {
		t.Fatalf("Expected %q, got %q", expected, listCmd.Args)
	}
}
