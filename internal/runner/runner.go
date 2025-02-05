package runner

import (
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/voidcontests/backend/internal/lib/logger/sl"
)

const TIMEOUT = "5s"

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func Execute(filename string) (*Result, error) {
	cmd := fmt.Sprintf(
		`gcc /sandbox/%s && timeout %s /sandbox/a.out && find /sandbox -type f -name "a.out" -delete`,
		filename, TIMEOUT,
	)

	return isolate(cmd)
}

func isolate(command string) (*Result, error) {
	cmd := exec.Command("docker", "run", "--rm",
		"--cpus=0.5",
		"--memory=128m",
		"--memory-swap=256m",
		"--pids-limit=50",
		"--read-only",
		"--network=none",
		"-v", "./:/sandbox",
		"runner",
		"bash", "-c", command,
	)

	var r Result
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			r.ExitCode = ee.ExitCode()
			r.Stderr = string(ee.Stderr)
		} else {
			slog.Error("can't execute command", sl.Err(err))
			return nil, err
		}
	}
	r.Stdout = string(out)

	return &r, nil
}
