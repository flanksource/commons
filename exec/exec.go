package exec

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
)

//SafeExec executes the sh script and returns the stdout and stderr, errors will result in a nil return only.
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

//Exec runs the sh script and forwards stderr/stdout to the console
func Exec(sh string) error {
	return Execf(sh)
}

//Execf runs the sh script and forwards stderr/stdout to the console
func Execf(sh string, args ...interface{}) error {
	log.Debugf("exec: "+sh, args...)
	cmd := exec.Command("bash", "-c", fmt.Sprintf(sh, args...))
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s failed with %s", sh, err)
	}

	if !cmd.ProcessState.Success() {
		return fmt.Errorf("%s failed to run", sh)
	}
	return nil
}
