package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"
)

type file struct {
	name    string
	modTime time.Time
	mode    os.FileMode
}

func (f file) Name() string       { return f.name }
func (f file) Size() int64        { return 0 }
func (f file) Mode() os.FileMode  { return f.mode }
func (f file) ModTime() time.Time { return f.modTime }
func (f file) IsDir() bool        { return f.Mode().IsDir() }
func (f file) Sys() interface{}   { return nil }

type testClient struct {
	consumeQueue bool
	dirList      []os.FileInfo
}

func (c *testClient) Consume(path string) error {
	if !c.consumeQueue {
		return fmt.Errorf("unexpected call with path=%s", path)
	}
	return nil
}

func (c *testClient) List(name, path string) ([]os.FileInfo, error) { return c.dirList, nil }

func writeTestConfig(config string) (string, error) {
	f, err := ioutil.TempFile("", "lftpq")
	if err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(f.Name(), []byte(config), 0644); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func newTestCLI(config string) (*CLI, *bytes.Buffer) {
	name, err := writeTestConfig(config)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	logger := log.New(&buf, "", log.LstdFlags)
	client := testClient{consumeQueue: false}
	return &CLI{
		Config:   name,
		wr:       &buf,
		log:      logger,
		consumer: &client,
		lister:   &client,
	}, &buf
}

func TestInvalidConfig(t *testing.T) {
	cli, _ := newTestCLI("42")
	defer os.Remove(cli.Config)
	if err := cli.Run(); !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Fatalf("want marshal error, got '%q'", err.Error())
	}
}

func TestConfigTest(t *testing.T) {
	cli, buf := newTestCLI(`{"Default": {"Parser": "show"}, "Sites": []}`)
	defer os.Remove(cli.Config)
	cli.Test = true
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want := `{
  "Default": {
    "GetCmd": "",
    "Name": "",
    "Dirs": null,
    "MaxAge": "",
    "Patterns": null,
    "Filters": null,
    "SkipSymlinks": false,
    "SkipExisting": false,
    "SkipFiles": false,
    "Parser": "show",
    "LocalDir": "",
    "Priorities": null,
    "Deduplicate": false,
    "PostCommand": "",
    "Replacements": null,
    "Merge": false,
    "Skip": false
  },
  "Sites": []
}
`
	if got := buf.String(); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestRunWithImport(t *testing.T) {
	cli, buf := newTestCLI(`
{
  "Default": {
    "Parser": "movie",
    "GetCmd": "mirror"
   },
   "Sites": [
    {
      "MaxAge": "0",
      "Name": "t1",
      "LocalDir": "/tmp/"
    }
   ]
}`)
	toImport := "/foo/bar.2017\n"
	stdin := strings.NewReader(toImport)

	// Queue is consumed by client
	cli.Import = "t1"
	cli.rd = stdin
	client := testClient{consumeQueue: true}
	cli.consumer = &client
	cli.lister = &client
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}

	// Dry run with lftp output
	stdin.Reset(toImport)
	client.consumeQueue = false
	cli.Dryrun = true
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want := `open t1
queue mirror /foo/bar.2017 /tmp/bar.2017
queue start
wait
exit
`
	if got := buf.String(); got != want {
		t.Errorf("want %s, got %s", want, got)
	}

	// Dry run with JSON output
	buf.Reset()
	stdin.Reset(toImport)
	cli.Format = "json"
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want = `[
  {
    "RemotePath": "/foo/bar.2017",
    "LocalPath": "/tmp/bar.2017",
    "ModTime": "0001-01-01T00:00:00Z",
    "Transfer": true,
    "Reason": "Import=true",
    "Media": {
      "Release": "bar.2017",
      "Name": "bar",
      "Year": 2017,
      "Season": 0,
      "Episode": 0
    },
    "Duplicate": false,
    "Merged": false
  }
]
`
	if got := buf.String(); got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}

func TestRun(t *testing.T) {
	cli, buf := newTestCLI(`
{
  "Default": {
    "Parser": "movie",
    "GetCmd": "mirror",
    "Patterns": [".*"]
   },
   "Sites": [
    {
      "MaxAge": "0",
      "Name": "t1",
      "Dirs": [
        "/baz"
      ],
      "LocalDir": "/tmp/"
    }
   ]
}`)

	// Queue is consumed by client
	client := testClient{consumeQueue: true, dirList: []os.FileInfo{file{name: "/baz/foo.2017"}}}
	cli.consumer = &client
	cli.lister = &client
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}

	// Dry run with lftp output
	client.consumeQueue = false
	cli.Dryrun = true
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want := `open t1
queue mirror /baz/foo.2017 /tmp/foo.2017
queue start
wait
exit
`
	if got := buf.String(); got != want {
		t.Errorf("want %s, got %s", want, got)
	}

	// Dry run with JSON output
	buf.Reset()
	cli.Format = "json"
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want = `[
  {
    "RemotePath": "/baz/foo.2017",
    "LocalPath": "/tmp/foo.2017",
    "ModTime": "0001-01-01T00:00:00Z",
    "Transfer": true,
    "Reason": "Match=.*",
    "Media": {
      "Release": "foo.2017",
      "Name": "foo",
      "Year": 2017,
      "Season": 0,
      "Episode": 0
    },
    "Duplicate": false,
    "Merged": false
  }
]
`
	if got := buf.String(); got != want {
		t.Errorf("want %s, got %s", want, got)
	}
}
