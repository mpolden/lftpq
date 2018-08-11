package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
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
	failDirs     []string
	dirList      []os.FileInfo
}

func (c *testClient) Consume(path string) error {
	if !c.consumeQueue {
		return fmt.Errorf("unexpected call with path=%s", path)
	}
	return nil
}

func (c *testClient) List(name, path string) ([]os.FileInfo, error) {
	for _, d := range c.failDirs {
		if d == path {
			return nil, fmt.Errorf("read error")
		}
	}
	return c.dirList, nil
}

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
	client := testClient{consumeQueue: false}
	return &CLI{
		Config:   name,
		stderr:   &buf,
		stdout:   &buf,
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
	cli, buf := newTestCLI(`
{
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "movie",
      "Template": "/tmp/"
    }
  ],
  "Sites": []
}
`)
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
    "LocalDir": "",
    "Priorities": null,
    "PostCommand": "",
    "Merge": false,
    "Skip": false
  },
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "movie",
      "Template": "/tmp/",
      "Replacements": null
    }
  ],
  "Sites": []
}
`
	if got := buf.String(); got != want {
		t.Fatalf("want %q, got %q", want, got)
	}
}

func TestRunImport(t *testing.T) {
	cli, buf := newTestCLI(`
{
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "movie",
      "Template": "/tmp/"
    }
  ],
  "Default": {
    "LocalDir": "d1",
    "GetCmd": "mirror"
  },
  "Sites": [
    {
      "MaxAge": "0",
      "Name": "t1"
    },
    {
      "MaxAge": "0",
      "Name": "t2"
    }
  ]
}
`)
	defer os.Remove(cli.Config)
	toImport := `
t1 /foo/bar.2017
t2 /baz/foo.2018
`
	stdin := strings.NewReader(toImport)

	// Queue is consumed by client
	cli.Import = true
	cli.stdin = stdin
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
open t2
queue mirror /baz/foo.2018 /tmp/foo.2018
queue start
wait
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
[
  {
    "RemotePath": "/baz/foo.2018",
    "LocalPath": "/tmp/foo.2018",
    "ModTime": "0001-01-01T00:00:00Z",
    "Transfer": true,
    "Reason": "Import=true",
    "Media": {
      "Release": "foo.2018",
      "Name": "foo",
      "Year": 2018,
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
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "movie",
      "Template": "/tmp/"
    }
  ],
  "Default": {
    "LocalDir": "d1",
    "GetCmd": "mirror",
    "Patterns": [".*"]
   },
  "Sites": [
    {
      "MaxAge": "0",
      "Name": "t1",
      "Dirs": [
        "/baz"
      ]
    }
   ]
}`)
	defer os.Remove(cli.Config)

	// Empty queue
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want := "lftpq: t1 queue is empty\n"
	if got := buf.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}

	// Queue is consumed by client
	client := testClient{consumeQueue: true, dirList: []os.FileInfo{file{name: "/baz/foo.2017"}}}
	cli.consumer = &client
	cli.lister = &client
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}

	// Dry run with lftp output
	buf.Reset()
	client.consumeQueue = false
	cli.Dryrun = true
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want = `open t1
queue mirror /baz/foo.2017 /tmp/foo.2017
queue start
wait
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

func TestRunSkipSite(t *testing.T) {
	cli, buf := newTestCLI(`
{
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "movie",
      "Template": "/tmp/"
    }
  ],
  "Sites": [
    {
      "MaxAge": "0",
      "Name": "t1",
      "LocalDir": "d1",
      "Skip": true
    }
   ]
}`)
	defer os.Remove(cli.Config)
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want := "lftpq: skipping site t1\n"
	if got := buf.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestRunListError(t *testing.T) {
	cli, buf := newTestCLI(`
{
  "LocalDirs": [
    {
      "Name": "d1",
      "Parser": "movie",
      "Template": "/tmp/"
    }
  ],
  "Sites": [
    {
      "MaxAge": "0",
      "Name": "t1",
      "Dirs": [
        "/foo"
      ],
      "LocalDir": "d1"
    }
   ]
}`)
	defer os.Remove(cli.Config)

	client := testClient{failDirs: []string{"/foo"}}
	cli.consumer = &client
	cli.lister = &client
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want := "lftpq: error while listing /foo on t1: read error\nlftpq: t1 queue is empty\n"
	if got := buf.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}

	// Prints only error when quiet
	buf.Reset()
	cli.Quiet = true
	if err := cli.Run(); err != nil {
		t.Fatal(err)
	}
	want = "lftpq: error while listing /foo on t1: read error\n"
	if got := buf.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}
