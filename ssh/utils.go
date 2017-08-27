package ssh

import (
	"errors"
	"fmt"
    "github.com/mrahbar/kubernetes-inspector/integration"
	"github.com/mrahbar/kubernetes-inspector/ssh/communicator"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/mrahbar/kubernetes-inspector/util"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"net"
	"os"
	"strings"
	"time"
)

func GetFirstAccessibleNode(sshOpts types.SSHConfig, nodes []types.Node, printer *integration.Printer) types.Node {
	if util.IsNodeAddressValid(sshOpts.LocalOn) {
		for _, n := range nodes {
			if util.NodeEquals(sshOpts.LocalOn, n) {
				return n
			}
		}
	}

    client := Executor{
        SshOpts: sshOpts,
        Printer: printer,
    }

	for _, n := range nodes {
        client.Node = n
        _, err := client.PerformCmd("hostname")
		if err == nil {
			return n
		}
	}

	return types.Node{}
}

func CombineOutput(sshout *types.SSHOutput) string {
	return fmt.Sprintf("Stdout: %s\nStderr: %s", sshout.Stdout, sshout.Stderr)
}

func prepareSSHConfig(sshConfig *types.SSHConfig) []error {
	c := sshConfig.Connection
	if c.Port == 0 {
		c.Port = 22
	}

	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}

	if c.HandshakeAttempts == 0 {
		c.HandshakeAttempts = 10
	}

	if util.IsNodeAddressValid(sshConfig.Bastion.Node) {
		bc := sshConfig.Bastion.SSHConnection
		if bc.Port == 0 {
			bc.Port = 22
		}

		if bc.PrivateKey == "" && c.PrivateKey != "" {
			bc.PrivateKey = c.PrivateKey
		}
	}

	if c.FileTransferMethod == "" {
		c.FileTransferMethod = "scp"
	}

	// Validation
	var errs []error
	if c.Username == "" {
		errs = append(errs, errors.New("An ssh_username must be specified\n  Note: some builders used to default ssh_username to \"root\"."))
	}

	if c.PrivateKey != "" {
		if _, err := os.Stat(c.PrivateKey); err != nil {
			errs = append(errs, fmt.Errorf(
				"ssh_private_key_file is invalid: %s", err))
		} else if _, err := communicator.FileSigner(c.PrivateKey); err != nil {
			errs = append(errs, fmt.Errorf(
				"ssh_private_key_file is invalid: %s", err))
		}
	}

	if util.IsNodeAddressValid(sshConfig.Bastion.Node) && !sshConfig.Bastion.SSHConnection.AgentAuth {
		if sshConfig.Bastion.SSHConnection.Password == "" && sshConfig.Bastion.SSHConnection.PrivateKey == "" {
			errs = append(errs, errors.New(
				"ssh_bastion_password or ssh_bastion_private_key_file must be specified"))
		}
	}

	if c.FileTransferMethod != "scp" && c.FileTransferMethod != "sftp" {
		errs = append(errs, fmt.Errorf(
			"ssh_file_transfer_method ('%s') is invalid, valid methods: sftp, scp",
			c.FileTransferMethod))
	}

	return errs
}

func establishSSHCommunication(sshOpts types.SSHConfig, address string, printer *integration.Printer) (*communicator.Comm, error) {
    commConfig, err := createCommunicationConfig(sshOpts, address, printer)
	if err != nil {
		return &communicator.Comm{}, err
	}

	var comm *communicator.Comm
	handshakeAttempts := 0

	for {
        comm, err = communicator.New(address, commConfig,
			func(msg string, a ...interface{}) {
                printer.PrintTrace(msg, a...)
			})

		if err != nil {
            printer.PrintDebug("SSH handshake err: %s", err)

			// Only count this as an attempt if we were able to attempt
			// to authenticate. Note this is very brittle since it depends
			// on the string of the error... but I don't see any other way.
			if strings.Contains(err.Error(), "authenticate") {
                printer.PrintDebug("Detected authentication error. Increasing handshake attempts.")
				handshakeAttempts += 1
			}

			if handshakeAttempts < sshOpts.Connection.HandshakeAttempts {
				// Try to connect via SSH a handful of times. We sleep here
				// so we don't get a ton of authentication errors back to back.
				time.Sleep(2 * time.Second)
				continue
			} else {
				return &communicator.Comm{}, err
			}
		} else {
			break
		}
	}

	return comm, nil
}

func createCommunicationConfig(sshOpts types.SSHConfig, nodeAddress string, printer *integration.Printer) (*communicator.Config, error) {
    errs := prepareSSHConfig(&sshOpts)
	if len(errs) > 0 {
		return &communicator.Config{}, flattenMultiError(errs)
	}

	var connFunc func() (net.Conn, error)
	address := fmt.Sprintf("%s:%d", nodeAddress, sshOpts.Connection.Port)

	if util.IsNodeAddressValid(sshOpts.Bastion.Node) {
		// We're using a bastion host, so use the bastion connfunc
		bAddr := fmt.Sprintf("%s:%d", util.GetNodeAddress(sshOpts.Bastion.Node), sshOpts.Bastion.Port)
		bConf, err := sshBastionConfig(&sshOpts.Bastion)
		if err != nil {
            printer.PrintDebug("BastionConfig failed: %s", err)
			return &communicator.Config{}, err
		}
		connFunc = communicator.BastionConnectFunc(
			"tcp", bAddr, bConf, "tcp", address)
	} else {
		// No bastion host, connect directly
		connFunc = communicator.ConnectFunc("tcp", address)
	}

	nc, err := connFunc()
	if err != nil {
        printer.PrintDebug("TCP connection to SSH ip/port failed: %s", err)
		return &communicator.Config{}, err
	}
	nc.Close()

	sshConfig, err := sshConfigFunc(&sshOpts.Connection)
	if err != nil {
        printer.PrintDebug("SSHConfig failed: %s", err)
		return &communicator.Config{}, err
	}

	// Then we attempt to connect via SSH
	commConfig := &communicator.Config{
		Connection:       connFunc,
		SSHConfig:        sshConfig,
		Pty:              false,
		DisableAgent:     true,
		UseSftp:          sshOpts.Connection.FileTransferMethod == "sftp",
		HandshakeTimeout: connectionTimeout(sshOpts.Connection),
	}

	return commConfig, nil
}

func sshConfigFunc(config *types.SSHConnection) (*ssh.ClientConfig, error) {
	auth := []ssh.AuthMethod{
		ssh.Password(config.Password),
		ssh.KeyboardInteractive(
			communicator.PasswordKeyboardInteractive(config.Password)),
	}

	if config.PrivateKey != "" {
		signer, err := communicator.FileSigner(config.PrivateKey)
		if err != nil {
			return nil, err
		}

		auth = append(auth, ssh.PublicKeys(signer))
	}

	if config.AgentAuth {
		agentAuthMethod, err := getAgentAuth()
		if err != nil {
			return nil, err
		}

		auth = append(auth, agentAuthMethod)
	}

	return &ssh.ClientConfig{
		User:            config.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func sshBastionConfig(config *types.BastionSSHConnection) (*ssh.ClientConfig, error) {
	auth := make([]ssh.AuthMethod, 0, 2)
	if config.Password != "" {
		auth = append(auth,
			ssh.Password(config.Password),
			ssh.KeyboardInteractive(
				communicator.PasswordKeyboardInteractive(config.Password)))
	}

	if config.PrivateKey != "" {
		signer, err := communicator.FileSigner(config.PrivateKey)
		if err != nil {
			return nil, err
		}

		auth = append(auth, ssh.PublicKeys(signer))
	}

	if config.AgentAuth {
		agentAuthMethod, err := getAgentAuth()
		if err != nil {
			return nil, err
		}

		auth = append(auth, agentAuthMethod)
	}

	return &ssh.ClientConfig{
		User:            config.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}, nil
}

func getAgentAuth() (ssh.AuthMethod, error) {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK is not set")
	}

	sshAgent, err := net.Dial("unix", authSock)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to SSH Agent socket %q: %s", authSock, err)
	}

	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers), nil
}

func flattenMultiError(errs []error) error {
	points := make([]string, len(errs))
	for i, err := range errs {
		points[i] = fmt.Sprintf("- %s", err)
	}

	return fmt.Errorf("%d error(s) occurred:\n%s", len(errs), strings.Join(points, "\n"))
}

func connectionTimeout(conn types.SSHConnection) time.Duration {
	t := 10 * time.Second
	if conn.Timeout > 0 {
		t = conn.Timeout
	}

	return t
}
