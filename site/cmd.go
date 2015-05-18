package site

import (
	"os"
	"os/exec"
)

type Lftp struct {
	Path string
	Args []string
}

func (l *Lftp) Cmd() *exec.Cmd {
	return exec.Command(l.Path, l.Args...)
}

func (l *Lftp) Run() error {
	cmd := l.Cmd()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
