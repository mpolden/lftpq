package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type Lftp struct {
	Path       string
	Script     string
	Args       []string
	Site       string
	ScriptName string
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

func Write(cmds []Lftp) (Lftp, error) {
	if len(cmds) == 0 {
		return Lftp{}, fmt.Errorf("need atleast one cmd")
	}
	f, err := ioutil.TempFile("", "lftpfetch")
	if err != nil {
		return Lftp{}, err
	}
	defer f.Close()
	f.WriteString("open " + cmds[0].Site + "\n")
	for _, cmd := range cmds {
		f.WriteString(cmd.Script + "\n")
	}
	f.WriteString("queue start\nwait\nexit\n")
	args := []string{"-f", f.Name()}
	return Lftp{
		Path:       cmds[0].Path,
		Args:       args,
		ScriptName: f.Name(),
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
