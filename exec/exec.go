package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

// SafeExec executes the sh script and returns the stdout and stderr, errors will result in a nil return only.
func SafeExec(sh string, args ...interface{}) (string, bool) {
	cmd := exec.Command("bash", "-c", fmt.Sprintf(sh, args...))
	data, err := cmd.CombinedOutput()
	if err != nil {
		log.Debugf("Failed to exec %s, %s %s\n", sh, data, err)
		return "", false
	}

	if !cmd.ProcessState.Success() {
		log.Debugf("Command did not succeed %s\n", sh)
		return "", false
	}
	return string(data), true

}

// Exec runs the sh script and forwards stderr/stdout to the console
func Exec(sh string) error {
	return Execf(sh)
}

// ExecfWithEnv runs the sh script and forwards stderr/stdout to the console
func ExecfWithEnv(sh string, env map[string]string, args ...interface{}) error {
	if log.IsLevelEnabled(log.TraceLevel) {
		envString := ""
		for k, v := range env {
			envString += fmt.Sprintf("%s=%s ", k, v)
		}
		log.Tracef("exec: %s %s\n", envString, fmt.Sprintf(sh, args...))
	} else {
		log.Debugf("exec: %s\n", fmt.Sprintf(sh, args...))
	}

	cmd := exec.Command("bash", "-c", fmt.Sprintf(sh, args...))

	var buf bytes.Buffer

	cmd.Stderr = io.MultiWriter(&buf, os.Stderr)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s failed with: %s, stderr: %s", sh, err, buf.String())
	}

	if !cmd.ProcessState.Success() {
		return fmt.Errorf("%s failed to run", sh)
	}
	return nil
}

// Execf runs the sh script and forwards stderr/stdout to the console
func Execf(sh string, args ...interface{}) error {
	return ExecfWithEnv(sh, make(map[string]string), args...)
}
