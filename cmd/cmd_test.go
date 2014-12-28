package cmd

import (
	"reflect"
	"testing"
)

func TestString(t *testing.T) {
	cmd := Lftp{
		Path: "lftp",
		Args: "mirror /foo /tmp",
		Site: "siteA",
	}
	expected := "lftp -e 'mirror /foo /tmp && exit' siteA"
	if cmd.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, cmd.String())
	}
}

func TestCmd(t *testing.T) {
	cmd := Lftp{
		Path: "/bin/lftp",
		Args: "mirror /foo /tmp",
		Site: "siteA",
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

func TestJoin(t *testing.T) {
	cmds := []Lftp{
		Lftp{
			Path: "/bin/lftp",
			Args: "queue mirror /foo /tmp",
			Site: "siteA",
		},
		Lftp{
			Path: "/bin/lftp",
			Args: "queue mirror /bar /tmp",
			Site: "siteA",
		},
	}
	c, err := Join(cmds)
	if err != nil {
		t.Fatal(err)
	}
	expected := "queue mirror /foo /tmp &&" +
		" queue mirror /bar /tmp &&" +
		" queue start && wait"
	if c.Args != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, c.Args)
	}
	expected = "/bin/lftp -e '" + expected + " && exit' siteA"
	if c.String() != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, c.String())
	}
}
