package site

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Site struct {
	LftpPath string
	Name     string
	Dir      string
	MaxAge_  string `json:"MaxAge"`
	Shows    []string
}

func (s *Site) MaxAge() (time.Duration, error) {
	return time.ParseDuration(s.MaxAge_)
}

func (s *Site) ListCommand() *exec.Cmd {
	options := fmt.Sprintf(`-e "cd %s &&
cls --date --time-style='%%F %%T %%z %%Z' &&
exit"`, s.Dir)
	args := strings.Split(options, " ")
	return exec.Command(s.LftpPath, args...)
}

func (s *Site) GetDirs() ([]Dir, error) {
	cmd := s.ListCommand()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	dirs := []Dir{}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " \t\r\n")
		if len(line) == 0 {
			continue
		}
		dir, err := ParseDir(line)
		if err != nil {
			return nil, err
		}

		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func (s *Site) FilterDirs() ([]Dir, error) {
	dirs, err := s.GetDirs()
	if err != nil {
		return nil, err
	}
	res := []Dir{}
	for _, dir := range dirs {
		if dir.IsSymlink {
			continue
		}
		age, err := s.MaxAge()
		if err != nil {
			return nil, err
		}
		if !dir.CreatedAfter(age) {
			continue
		}
		if !dir.MatchAny(s.Shows) {
			continue
		}
		res = append(res, dir)
	}
	return res, nil
}
