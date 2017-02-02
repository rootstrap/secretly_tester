package remote

import (
	"errors"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/toptier/secretly_tester/interleaver"
	"golang.org/x/crypto/ssh"
)

type Node struct {
	Host     string
	Username string
	Auth     []ssh.AuthMethod
	client   *ssh.Client
	session  *ssh.Session
}

func (node *Node) Connect() error {
	client, err := ssh.Dial("tcp", node.Host, &ssh.ClientConfig{
		Timeout: 10 * time.Second,
		Auth:    node.Auth,
		User:    node.Username,
	})
	if err != nil {
		return err
	}
	node.client = client
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return err
	}
	node.session = session
	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return err
	}
	go interleaver.Stdout.Copy(stdout)
	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return err
	}
	go io.Copy(os.Stderr, stderr)
	return nil
}

func (node *Node) Start(cmd string) error {
	return node.session.Start(cmd)
}

type Remote struct {
	Nodes []*Node
}

func NewRemote(hostsString string, privateKeyPath string) (*Remote, error) {
	var privateKeyAuthMethod ssh.AuthMethod
	var err error
	if privateKeyPath != "" {
		privateKeyAuthMethod, err = privateKeyAutMethod(privateKeyPath)
		if err != nil {
			return nil, err
		}
	}

	nodes := []*Node{}
	for _, host := range hostsSeparatorRegexp.Split(hostsString, -1) {
		parsed, err := url.Parse("proto://" + host)
		if err != nil {
			return nil, err
		}
		if parsed.User == nil {
			return nil, errors.New("host " + host + " must have user like user@host")
		}

		authMethods := []ssh.AuthMethod{}
		if password, hasPassword := parsed.User.Password(); hasPassword {
			authMethods = append(authMethods, ssh.Password(password))
		}
		if privateKeyPath != "" {
			authMethods = append(authMethods, privateKeyAuthMethod)
		}

		host := parsed.Host
		if !hostWithPortRegexp.MatchString(host) {
			host = host + ":22"
		}

		nodes = append(nodes, &Node{
			Auth:     authMethods,
			Username: parsed.User.Username(),
			Host:     host,
		})
	}
	return &Remote{Nodes: nodes}, nil
}

func (remote *Remote) Connect() error {
	for _, node := range remote.Nodes {
		if err := node.Connect(); err != nil {
			return err
		}
	}
	return nil
}

func (remote *Remote) Start(cmd string) error {
	for _, node := range remote.Nodes {
		if err := node.Start(cmd); err != nil {
			return err
		}
	}
	return nil
}

func (remote *Remote) StartEach(cmd func() (string, error)) error {
	for _, node := range remote.Nodes {
		cmd, err := cmd()
		if err != nil {
			return err
		}
		if err = node.Start(cmd); err != nil {
			return err
		}
	}
	return nil
}

func privateKeyAutMethod(file string) (method ssh.AuthMethod, err error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return
	}
	method = ssh.PublicKeys(key)
	return
}

var hostsSeparatorRegexp = regexp.MustCompile("[ ]+")
var hostWithPortRegexp = regexp.MustCompile(".*:\\d{1,5}")
