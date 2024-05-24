package kube

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2/data/binding"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/homedir"
)

type PreferencesStruct struct {
	Active    bool
	Context   string
	Namespace string
	Config    string
}

var thisContext, thisNamespace string
var memoryMonitorQuit = make(chan bool)
var memoryMonitorRunning = false
var Kubeconfig *string
var FilteredByDeployment string = ""

func Setup(context1, namespace1 string) {
	if home := homedir.HomeDir(); home != "" {
		Kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		Kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
	thisContext = context1
	thisNamespace = namespace1

	os.Setenv("PATH", fmt.Sprintf("%s:/Users/s457972/.krew/bin/:/Users/s457972/.docker/bin/", os.Getenv("PATH")))
	cmd := exec.Command(
		"kubectl",
		"oidc-login",
		"get-token",
		"--oidc-issuer-url=https://auth.griffith.edu.au",
		"--oidc-client-id=oidc-kubernetes",
		"--oidc-extra-scope=groups",
		"--oidc-extra-scope=department",
		"--grant-type=authcode",
	)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logOut(fmt.Sprintf("Error: %v\nStdOut: %v\nStdErr: %v\n", err, out.String(), stderr.String()))
	} else {
		logOut(fmt.Sprintf("Output: %v\n", out.String()))
	}
}
func BuildContextConfigFromFlags(masterUrl, kubeconfigPath string) (*restclient.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterUrl}, CurrentContext: thisContext}).ClientConfig()
}

func SwitchContext(context1 string) (*kubernetes.Clientset, error) {
	thisContext = context1
	return GetClientset()
}

func GetClientset() (*kubernetes.Clientset, error) {
	// use the current context in kubeconfig
	config, err := BuildContextConfigFromFlags("", *Kubeconfig)
	if err != nil {
		logOut(fmt.Sprintf("Clientset failed %v\n", err))
		return nil, err
	}
	logOut(fmt.Sprintf("Config: %v\n", config))

	// create the clientset
	return kubernetes.NewForConfig(config)
}

func GetMemoryForPod(podname string, results binding.ExternalIntList, maxMemory *int) {
	if memoryMonitorRunning {
		memoryMonitorQuit <- true
	}
	memoryMonitorRunning = true
	go func() {
		cmdArray := []string{`kubectl`,
			"--context=" + thisContext,
			"--namespace=" + thisNamespace,
			"exec",
			"--stdin",
			"--tty",
			podname,
			"--",
			"bash",
			"-c",
			"while true ; do free && sleep 60 ; done"}
		cmd := exec.Command(cmdArray[0], cmdArray[1:]...)

		stdout, _ := cmd.StdoutPipe()
		err := cmd.Start()
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-memoryMonitorQuit:
				return
			default:
				m := scanner.Text()
				if strings.Contains(m, "Mem: ") {
					re := regexp.MustCompile(`\s+`)
					split := re.Split(m, -1)
					*maxMemory, _ = strconv.Atoi(split[1])
					newMemory, _ := strconv.Atoi(split[2])
					results.Append(newMemory)
				}
			}
		}
		cmd.Wait()
	}()
}

func GetDeployments() (returnme []string, err error) {
	deps, err := getDeployments()
	if err == nil {
		returnme = []string{}
		for _, x := range deps.Items {
			returnme = append(returnme, x.Name)
		}
	}
	return
}

func getDeployments() (*apps.DeploymentList, error) {
	clientset, err := GetClientset()

	if err != nil {
		x := apps.DeploymentList{}
		return &x, err
	}
	p, e := clientset.AppsV1().Deployments(thisNamespace).List(context.TODO(), metav1.ListOptions{})
	logOut(fmt.Sprintf("Depl error: %v\n", e))
	logOut(fmt.Sprintf("Found %d deps\n - %s,%s\n", len(p.Items), thisContext, thisNamespace))
	return p, e
}

func GetPods() (returnme []string, err error) {
	deps, err := getPods()
	if err == nil {
		returnme = []string{}
		for _, x := range deps.Items {
			returnme = append(returnme, x.Name)
		}
	}
	return
}

func getPods() (*v1.PodList, error) {
	clientset, err := GetClientset()

	if err != nil {
		x := v1.PodList{}
		return &x, err
	}
	p, e := clientset.CoreV1().Pods(thisNamespace).List(context.TODO(), metav1.ListOptions{})
	logOut(fmt.Sprintf("Pod error: %v\n", e))
	if FilteredByDeployment != "" {
		lastIndexFilter := len(FilteredByDeployment)
		n := 0
		for _, p1 := range p.Items {
			if len(p1.Name) > lastIndexFilter && p1.Name[0:lastIndexFilter] == FilteredByDeployment {
				p.Items[n] = p1
				n++
			}
		}
		p.Items = p.Items[:n]
	}
	logOut(fmt.Sprintf("Found %d pods\n - %s,%s\n", len(p.Items), thisContext, thisNamespace))
	return p, e
}

func logOut(this string) {
	f, _ := os.OpenFile("/tmp/test.txt",
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f.WriteString(this)
	f.Close()
}
