package cmd

import (
	"os/exec"
	"strings"
)

type Lftp struct {
	Path string
	Args string
}

func (l *Lftp) args() []string {
	return []string{"-e", l.Args + " && exit"}
}

func (l *Lftp) String() string {
	return l.Path + " " + strings.Join(l.args(), " ")
}

func (l *Lftp) Cmd() *exec.Cmd {
	args := l.args()
	return exec.Command(l.Path, args...)
}

func Join(cmds []Lftp) string {
	res := make([]string, 0, len(cmds))
	for _, cmd := range cmds {
		res = append(res, cmd.Args)
	}
	res = append(res, "queue start", "wait")
	return strings.Join(res, " && ")
}
