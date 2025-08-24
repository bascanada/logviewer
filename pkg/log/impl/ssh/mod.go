package ssh

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/reader"
	sshc "golang.org/x/crypto/ssh"
	"k8s.io/client-go/util/homedir"
)

const (
	OptionsCmd = "Cmd"
)
type SSHLogClientOptions struct {
	User string `json:"user"`
	Addr string `json:"addr"`

	PrivateKey string `json:"privateKey"`
}

type sshLogClient struct {
	conn *sshc.Client
}

func (lc sshLogClient) Get(ctx context.Context, search *client.LogSearch) (client.LogSearchResult, error) {
	cmd := search.Options.GetString(OptionsCmd)

	if cmd == "" {
		panic(errors.New("cmd is missing for sshLogClient"))
	}

	session, err := lc.conn.NewSession()
	if err != nil {
		return nil, err
	}

	modes := sshc.TerminalModes{
		sshc.ECHO:          0,     // disable echoing
		sshc.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		sshc.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return nil, err
	}

	_, err = session.StdinPipe()
	if err != nil {
		return nil, err
	}

	errOut, err := session.StderrPipe()
	if err != nil {
		return nil, err
	}

	out, err := session.StdoutPipe()
	if err != nil {
		return nil, err
	}

	go func() {
		_, err = session.Output(cmd)
		if err != nil {

			by, _ := ioutil.ReadAll(errOut)
			fmt.Println("Error : " + string(by))

			panic(err)
		}
	}()

	scanner := bufio.NewScanner(out)

	return reader.GetLogResult(search, scanner, session), nil
}

func GetLogClient(options SSHLogClientOptions) (client.LogClient, error) {

	if options.Addr == "" {
		return nil, errors.New("ssh address (Addr) is empty")
	}
	if options.User == "" {
		return nil, errors.New("ssh user (User) is empty")
	}

	var privateKeyFile string
	if options.PrivateKey != "" {
		privateKeyFile = options.PrivateKey
	} else {
		privateKeyFile = filepath.Join(homedir.HomeDir(), ".ssh", "id_rsa")
	}

	key, err := ioutil.ReadFile(privateKeyFile)
	if err != nil {
		return nil, err
	}
	signer, err := sshc.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	sshConfig := &sshc.ClientConfig{
		User: options.User,
		Auth: []sshc.AuthMethod{
			sshc.PublicKeys(signer),
		},
		HostKeyCallback: sshc.HostKeyCallback(
			func(hostname string, remote net.Addr, key sshc.PublicKey) error {
				return nil
			}),
	}

	conn, err := sshc.Dial("tcp", options.Addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return sshLogClient{conn}, nil
}
