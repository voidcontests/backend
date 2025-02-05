package runner

import (
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/voidcontests/backend/internal/lib/logger/sl"
)

type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

func Execute(filename string) (*Result, error) {
	cmd := exec.Command("docker", "run", "--rm",
		"--cpus=0.5",
		"--memory=128m",
		"--memory-swap=256m",
		"--pids-limit=50",
		"--read-only",
		"--network=none",
		"-v", "./:/sandbox",
		"runner",
		"bash", "-c",
		fmt.Sprintf(`gcc /sandbox/%s && timeout 5s /sandbox/a.out && find /sandbox -type f -name "a.out" -delete`, filename),
	)

	var r Result
	out, err := cmd.Output()
	if err != nil {
		if eerr, ok := err.(*exec.ExitError); ok {
			r.ExitCode = eerr.ExitCode()
			r.Stderr = string(eerr.Stderr)
		} else {
			slog.Error("Error executing command", sl.Err(err))
			return nil, err
		}
	}
	r.Stdout = string(out)

	return &r, nil
}
