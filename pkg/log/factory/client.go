package factory

import (
	"errors"

	"github.com/berlingoqc/logviewer/pkg/log/client"
	"github.com/berlingoqc/logviewer/pkg/log/client/config"
	"github.com/berlingoqc/logviewer/pkg/log/impl/docker"
	"github.com/berlingoqc/logviewer/pkg/log/impl/elk/kibana"
	"github.com/berlingoqc/logviewer/pkg/log/impl/elk/opensearch"
	"github.com/berlingoqc/logviewer/pkg/log/impl/k8s"
	"github.com/berlingoqc/logviewer/pkg/log/impl/local"
	splunk "github.com/berlingoqc/logviewer/pkg/log/impl/splunk/logclient"
	"github.com/berlingoqc/logviewer/pkg/log/impl/ssh"
	"github.com/berlingoqc/logviewer/pkg/ty"
)

type logClientFactory struct {
	clients ty.LazyMap[string, client.LogClient]
}

func GetLogClientFactory(clients config.Clients) (*logClientFactory, error) {

	logClientFactory := new(logClientFactory)
	logClientFactory.clients = make(ty.LazyMap[string, client.LogClient])

	for k, v := range clients {
		switch v.Type {
		case "opensearch":
			options := v.Options
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := opensearch.GetClient(opensearch.OpenSearchTarget{
					Endpoint: options.GetString("Endpoint"),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "kibana":
			options := v.Options
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := kibana.GetClient(kibana.KibanaTarget{Endpoint: options.GetString("Endpoint")})
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
					KubeConfig: v.Options.GetString("KubeConfig"),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "ssh":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := ssh.GetLogClient(ssh.SSHLogClientOptions{
					User:       v.Options.GetString("User"),
					Addr:       v.Options.GetString("Addr"),
					PrivateKey: v.Options.GetString("PrivateKey"),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "splunk":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := splunk.GetClient(splunk.SplunkLogSearchClientOptions{
					Url:        v.Options.GetString("Url"),
					Headers:    v.Options.GetMS("Headers").ResolveVariables(),
					SearchBody: v.Options.GetMS("SearchBody").ResolveVariables(),
				})
				if err != nil {
					return nil, err
				}

				return &vv, nil
			})
		case "docker":
			logClientFactory.clients[k] = ty.GetLazy(func() (*client.LogClient, error) {
				vv, err := docker.GetLogClient(v.Options.GetString("Host"))

				return &vv, err
			})
		default:
			return nil, errors.New("invalid type for client : " + v.Type)
		}
	}

	return logClientFactory, nil
}
