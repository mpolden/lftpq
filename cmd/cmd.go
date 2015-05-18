package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Lftp struct {
	Path   string
	Script string
	Args   []string
	Site   string
}

func (l *Lftp) String() string {
	if l.Script != "" {
		return fmt.Sprintf("%s -e '%s && exit' %s", l.Path, l.Script,
			l.Site)
	}
	return l.Path + " " + strings.Join(l.Args, " ")
}

func (l *Lftp) Cmd() *exec.Cmd {
	var args []string
	if l.Script != "" {
		args = []string{"-e", l.Script + " && exit", l.Site}
	} else {
		args = l.Args
	}
	return exec.Command(l.Path, args...)
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
