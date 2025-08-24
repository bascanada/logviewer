package k8s

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/reader"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	// Import all auth plugins (incl. exec, OIDC, GCP, Azure, etc.) so kubeconfigs
	// referencing them (e.g. auth-provider: oidc) are supported without extra code.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	FieldNamespace = "namespace"
	FieldContainer = "container"
	FieldPrevious  = "previous"
	FieldPod       = "pod"

	OptionsTimestamp = "timestamp"
)

type K8sLogClientOptions struct {
	KubeConfig string `json:"kubeConfig"`
}

/*

* Need to support regex for pod name , to be able to get all pods from a deployment or someting
* similar and get them in the same log flow to maybe parse them afterwards
 */

type k8sLogClient struct {
	clientset *kubernetes.Clientset
}

func (lc k8sLogClient) Get(search *client.LogSearch) (client.LogSearchResult, error) {

	namespace := search.Options.GetString(FieldNamespace)
	pod := search.Options.GetString(FieldPod)
	container := search.Options.GetString(FieldContainer)
	previous := search.Options.GetBool(FieldPrevious)
	timestamp := search.Options.GetBool(OptionsTimestamp)

	follow := search.Refresh.Duration.Value != ""

	tailLines := int64(search.Size.Value)

	ipod := lc.clientset.CoreV1().Pods(namespace)

	logOptions := v1.PodLogOptions{
		TailLines:  &tailLines,
		Follow:     follow,
		Timestamps: timestamp,
		Container:  container,
		Previous:   previous,
	}

	if search.Range.Last.Value != "" {
		lastDuration, err := time.ParseDuration(search.Range.Last.Value)
		if err != nil {
			return nil, err
		}
		seconds := int64(lastDuration.Seconds())
		logOptions.SinceSeconds = &seconds
	} else if search.Range.Gte.Value != "" {
		time, err := time.Parse(time.RFC3339, search.Range.Gte.Value)
		if err != nil {
			return nil, err
		}
		metaTime := metav1.NewTime(time)
		logOptions.SinceTime = &metaTime
	}

	req := ipod.GetLogs(pod, &logOptions)

	ctx := context.Background()

	podLogs, err2 := req.Stream(ctx)
	if err2 != nil {
		return nil, err2
	}

	scanner := bufio.NewScanner(podLogs)

	return reader.GetLogResult(search, scanner, podLogs), nil
}

func ensureKubeconfig(kubeconfig string) error {
	if _, err := os.Stat(kubeconfig); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// Attempt auto-generation only for integration environment pattern
	// Expect running container named k3s-server
	if os.Getenv("LOGVIEWER_AUTO_K3S") == "false" {
		return errors.New("kubeconfig not found and auto-generation disabled: " + kubeconfig)
	}

	// docker cp k3s-server:/etc/rancher/k3s/k3s.yaml <kubeconfig>
	if err := os.MkdirAll(filepath.Dir(kubeconfig), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("docker", "cp", "k3s-server:/etc/rancher/k3s/k3s.yaml", kubeconfig)
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.New("failed to copy kubeconfig from k3s-server: " + string(out))
	}
	// Replace 127.0.0.1 with localhost to match compose port mapping semantics
	b, err := os.ReadFile(kubeconfig)
	if err == nil {
		updated := []byte{}
		updated = append(updated, b...)
		// simple replacement (avoid bringing in strings dep already imported indirectly)
		content := string(updated)
		content = strings.ReplaceAll(content, "127.0.0.1", "localhost")
		if err = os.WriteFile(kubeconfig, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func GetLogClient(options K8sLogClientOptions) (client.LogClient, error) {
	var kubeconfig string
	if options.KubeConfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	} else {
		kubeconfig = options.KubeConfig
	}

	if err := ensureKubeconfig(kubeconfig); err != nil {
		return nil, err
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return k8sLogClient{clientset: clientset}, nil
}
