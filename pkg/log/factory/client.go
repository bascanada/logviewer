package factory

import (
	"errors"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/log/impl/cloudwatch"
	"github.com/bascanada/logviewer/pkg/log/impl/docker"
	"github.com/bascanada/logviewer/pkg/log/impl/elk/kibana"
	"github.com/bascanada/logviewer/pkg/log/impl/elk/opensearch"
	"github.com/bascanada/logviewer/pkg/log/impl/k8s"
	"github.com/bascanada/logviewer/pkg/log/impl/local"
	splunk "github.com/bascanada/logviewer/pkg/log/impl/splunk/logclient"
	"github.com/bascanada/logviewer/pkg/log/impl/ssh"
	"github.com/bascanada/logviewer/pkg/ty"
)

type LogClientFactory interface {
	Get(name string) (*client.LogClient, error)
}

type logClientFactory struct {
	clients ty.LazyMap[string, client.LogClient]
}

func (lcf *logClientFactory) Get(name string) (*client.LogClient, error) {
	return lcf.clients.Get(name)
}

func GetLogClientFactory(clients config.Clients) (LogClientFactory, error) {

	logClientFactory := new(logClientFactory)
	logClientFactory.clients = make(ty.LazyMap[string, client.LogClient])

	for k, v := range clients {
		// IMPORTANT: shadow loop variable so each closure below captures its own copy.
		v := v
		// Resolve environment variables inside client option values (string only)
		v.Options = v.Options.ResolveVariables()
		switch v.Type {
		case "opensearch":
			options := v.Options
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := opensearch.GetClient(opensearch.OpenSearchTarget{
					Endpoint: options.GetString("endpoint"),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "kibana":
			options := v.Options
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := kibana.GetClient(kibana.KibanaTarget{Endpoint: options.GetString("endpoint")})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "local":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := local.GetLogClient()
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "k8s":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := k8s.GetLogClient(k8s.K8sLogClientOptions{
					KubeConfig: v.Options.GetString("kubeConfig"),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "ssh":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				user := v.Options.GetString("user")
				addr := v.Options.GetString("addr")
				pk := v.Options.GetString("privateKey")
				vv, err := ssh.GetLogClient(ssh.SSHLogClientOptions{
					User:       user,
					Addr:       addr,
					PrivateKey: pk,
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "splunk":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := splunk.GetClient(splunk.SplunkLogSearchClientOptions{
					Url:        v.Options.GetString("url"),
					Headers:    v.Options.GetMS("headers").ResolveVariables(),
					SearchBody: v.Options.GetMS("searchBody").ResolveVariables(),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "docker":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				host := v.Options.GetString("host")
				if host == "" {
					// Fallback to default local docker socket
					host = "unix:///var/run/docker.sock"
				}
				vv, err := docker.GetLogClient(host)
				return &vv, err
			})
		case "cloudwatch":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				// Pass the client-specific options to our new factory function
				vv, err := cloudwatch.GetLogClient(v.Options)
				if err != nil {
					return nil, err
				}
				return &vv, nil
			})
		default:
			return nil, errors.New("invalid type for client : " + v.Type)
		}
	}

	return logClientFactory, nil
}
