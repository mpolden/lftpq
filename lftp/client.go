package lftp

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Client struct {
	Path   string
	GetCmd string
}

func (c *Client) Run(args []string, inheritIO bool) error {
	cmd := exec.Command(c.Path, args...)
	if inheritIO {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func parseDirList(r io.Reader) ([]os.FileInfo, error) {
	var files []os.FileInfo
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 {
			continue
		}
		f, err := ParseFile(line)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func listArgs(name, path string) []string {
	script := "cls -1 --classify --date --time-style='%F %T %z %Z' " + path + " && exit"
	return []string{"-e", script, name}
}

func (c *Client) List(name, path string) ([]os.FileInfo, error) {
	cmd := exec.Command(c.Path, listArgs(name, path)...)

	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	dirs, err := parseDirList(stdout)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return dirs, nil
}
