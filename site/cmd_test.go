package site

import (
	"reflect"
	"testing"
)

func TestCmd(t *testing.T) {
	cmd := Lftp{
		Path: "/bin/lftp",
		Args: []string{"-e", "mirror /foo /tmp", "siteA"},
	}
	c := cmd.Cmd()
	if c.Path != "/bin/lftp" {
		t.Fatalf("Expected /bin/lftp, got %s", c.Path)
	}
	if reflect.DeepEqual(cmd.Args, c.Args) {
		t.Fatalf("Expected %q, got %q", cmd.Args, c.Args)
	}
}
