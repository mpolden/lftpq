package cmd

import (
	"reflect"
	"testing"
)

func TestString(t *testing.T) {
	cmd := Lftp{
		Path:   "lftp",
		Script: "mirror /foo /tmp",
		Site:   "siteA",
	}
	expected := "lftp -e 'mirror /foo /tmp && exit' siteA"
	if cmd.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, cmd.String())
	}
}

func TestStringArgs(t *testing.T) {
	cmd := Lftp{
		Path: "lftp",
		Args: []string{"-f", "/tmp/foo"},
	}
	expected := "lftp -f /tmp/foo"
	if cmd.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, cmd.String())
	}
}

func TestCmd(t *testing.T) {
	cmd := Lftp{
		Path:   "/bin/lftp",
		Script: "mirror /foo /tmp",
		Site:   "siteA",
	}
	c := cmd.Cmd()
	if c.Path != "/bin/lftp" {
		t.Fatalf("Expected /bin/lftp, got %s", c.Path)
	}
	args := []string{"-e", "mirror /foo /tmp && exit", "siteA"}
	if reflect.DeepEqual(c.Args, args) {
		t.Fatalf("Expected '%s', got '%s'", args, c.Args)
	}
}
