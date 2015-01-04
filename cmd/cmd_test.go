package cmd

import (
	"io/ioutil"
	"os"
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

func TestWrite(t *testing.T) {
	cmds := []Lftp{
		Lftp{
			Path:   "/bin/lftp",
			Script: "queue mirror /foo /tmp",
			Site:   "siteA",
		},
		Lftp{
			Path:   "/bin/lftp",
			Script: "queue mirror /bar /tmp",
			Site:   "siteA",
		},
	}
	c, err := Write(cmds)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Remove(c.ScriptName)
	}()
	expected := `open siteA
queue mirror /foo /tmp
queue mirror /bar /tmp
queue start
wait
exit
`
	f, err := ioutil.ReadFile(c.ScriptName)
	if err != nil {
		t.Fatal(err)
	}
	content := string(f)
	if content != expected {
		t.Fatalf("Expected '%s', got '%s'", expected, content)
	}
}
