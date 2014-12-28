package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Lftp struct {
	Path string
	Args string
	Site string
}

func (l *Lftp) String() string {
	return fmt.Sprintf("%s -e '%s && exit' %s", l.Path, l.Args, l.Site)
}

func (l *Lftp) Cmd() *exec.Cmd {
	args := []string{"-e", l.Args + " && exit", l.Site}
	return exec.Command(l.Path, args...)
}

func Join(cmds []Lftp) (Lftp, error) {
	if len(cmds) == 0 {
		return Lftp{}, fmt.Errorf("cmds is empty")
	}
	res := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		res = append(res, cmd.Args)
	}
	res = append(res, "queue start", "wait")
	args := strings.Join(res, " && ")
	return Lftp{
		Path: cmds[0].Path,
		Args: args,
		Site: cmds[0].Site,
	}, nil
}

func (l *Lftp) Run() error {
	cmd := l.Cmd()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	go func() {
		if _, err := io.Copy(os.Stdout, stdout); err != nil {
			panic(err)
		}
	}()
	go func() {
		if _, err := io.Copy(os.Stderr, stderr); err != nil {
			panic(err)
		}
	}()
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
