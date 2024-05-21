package kube

import (
	"bufio"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strings"

	"fyne.io/fyne/v2/data/binding"
)

type PreferencesStruct struct {
	Active    bool
	Context   string
	Namespace string
	Config    string
}
type saveOutput struct {
	savedOutput []byte
}

func (so *saveOutput) Write(p []byte) (n int, err error) {
	so.savedOutput = append(so.savedOutput, p...)
	return len(p), nil
}

var context, namespace string
var memoryMonitorQuit = make(chan bool)
var memoryMonitorRunning = false

func Setup(context1, namespace1 string) {
	context = context1
	namespace = namespace1
}

func GetMemoryForPod(podname string, results binding.ExternalStringList) {
	if memoryMonitorRunning {
		memoryMonitorQuit <- true
	}
	fmt.Printf("One at a time\n")
	memoryMonitorRunning = true
	go func() {
		cmdArray := []string{`kubectl`,
			"--context=" + context,
			"--namespace=" + namespace,
			"exec",
			"--stdin",
			"--tty",
			podname,
			"--",
			"bash",
			"-c",
			"while true ; do free && sleep 300 ; done"}
		cmd := exec.Command(cmdArray[0], cmdArray[1:]...)

		stdout, _ := cmd.StdoutPipe()
		err := cmd.Start()
		fmt.Printf("Start %v\n", cmdArray)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Printf(".")
			select {
			case <-memoryMonitorQuit:
				return
			default:
				m := scanner.Text()
				fmt.Printf("\n%s\n", m)
				if strings.Contains(m, "Mem: ") {
					re := regexp.MustCompile(`\s+`)
					split := re.Split(m, -1)
					results.Append(split[2])
				}
			}
		}
		fmt.Printf("Wait")
		cmd.Wait()
	}()
	fmt.Printf("Done!")
}

func GetDeployments() ([]string, error) {
	cmdVariables := []string{
		`kubectl`,
		"--context=" + context,
		"--namespace=" + namespace,
		"--output=json",
		"get",
		"deployments",
		"-o",
		"jsonpath='{.items[*].metadata.name}'",
	}
	bob, err := runAndReturn(cmdVariables)
	if err == nil {
		return strings.Split(strings.Trim(bob[0], "'"), " "), nil
	}
	return bob, err
}

func GetPods() ([]string, error) {
	cmdVariables := []string{
		`kubectl`,
		"--context=" + context,
		"--namespace=" + namespace,
		"--output=json",
		"get",
		"pods",
		"-o",
		"jsonpath='{.items[*].metadata.name}'",
	}
	bob, err := runAndReturn(cmdVariables)
	if err == nil {
		return strings.Split(strings.Trim(bob[0], "'"), " "), nil
	}
	return bob, err
}

func runAndReturn(cmdVariables []string) ([]string, error) {
	fmt.Printf("Running %v\n", cmdVariables)
	cmd := exec.Command(cmdVariables[0], cmdVariables[1:]...)

	var so saveOutput
	cmd.Stdout = &so
	cmd.Stderr = &so
	bob := cmd.Run()
	if bob != nil && bob.Error() != "exit status 1" {
		log.Fatalf("Damn %v\n", bob)
		return []string{}, bob
	}
	xy := string(so.savedOutput)
	fmt.Printf("Returning %s\n", xy)
	return strings.Split(strings.Trim(xy, "\r\n"), "\r\n"), bob
}
