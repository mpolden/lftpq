package cmd

import (
	"reflect"
	"testing"
)

func TestString(t *testing.T) {
	cmd := Lftp{
		Path: "lftp",
		Args: "mirror /foo /tmp",
	}
	expected := "lftp -e mirror /foo /tmp && exit"
	if cmd.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, cmd.String())
	}
}

func TestCmd(t *testing.T) {
	cmd := Lftp{
		Path: "/bin/lftp",
		Args: "mirror /foo /tmp",
	}
	c := cmd.Cmd()
	if c.Path != "/bin/lftp" {
		t.Fatalf("Expected /bin/lftp, got %s", c.Path)
	}
	args := []string{"-e", "mirror /foo /tmp && exit"}
	if reflect.DeepEqual(c.Args, args) {
		t.Fatalf("Expected '%s', got '%s'", args, c.Args)
	}
}
