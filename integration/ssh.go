package integration



import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/crypto/ssh"
	"strings"
	"github.com/mrahbar/kubernetes-inspector/util"
	"io"
)

var baseSSHArgs = []string{
	"-F", "/dev/null",
	"-o", "PasswordAuthentication=no",
	"-o", "StrictHostKeyChecking=no",
	"-o", "UserKnownHostsFile=/dev/null",
	"-o", "LogLevel=quiet", // suppress "Warning: Permanently added '[localhost]:2022' (ECDSA) to the list of known hosts."
	"-o", "ConnectionAttempts=3", // retry 3 times if SSH connection fails
	"-o", "ConnectTimeout=10", // timeout after 10 seconds
	"-o", "ControlMaster=no", // disable ssh multiplexing
	"-o", "ControlPath=none",
}

func PerformSSHCmd(out io.Writer, sshOpts *SSHConfig, node *Node, cmd string) (string, error) {
	client, err := NewClient(node.IP, sshOpts.Port, sshOpts.User, sshOpts.Key,
		strings.FieldsFunc(sshOpts.Options, func(r rune) bool {
			return r == ' ' || r == ','
		}))

	if err != nil {
		msg := fmt.Sprintf("Error creating SSH client for host %s (%s): %v", node.Host, node.IP, err)
		util.PrettyPrintErr(out, msg)
		return "", err
	}

	return client.Output(sshOpts.Pty, cmd)
}

type Client interface {
	Output(pty bool, args ...string) (string, error)
	Shell(pty bool, args ...string) error
}

type ExternalClient struct {
	BaseArgs   []string
	BinaryPath string
	cmd        *exec.Cmd
}

// NewClient verifies ssh is available in the PATH and returns an SSH client
func NewClient(host string, port int, user string, key string, options []string) (Client, error) {
	if err := ValidUnencryptedPrivateKey(key); err != nil {
		return nil, err
	}

	sshBinaryPath, err := exec.LookPath("ssh")
	if err != nil {
		return nil, fmt.Errorf("command not found: ssh")
	}

	return newExternalClient(sshBinaryPath, user, host, port, key, options)
}

func newExternalClient(sshBinaryPath string, user string, host string, port int, key string, options []string) (*ExternalClient, error) {
	// Get default args with user and host
	args := append(baseSSHArgs, options...)
	// set port
	args = append(args, fmt.Sprintf("%s@%s", user, host))
	// set port
	args = append(args, "-p", fmt.Sprintf("%d", port))
	// set key
	args = append(args, "-i", key)

	client := &ExternalClient{
		BinaryPath: sshBinaryPath,
		BaseArgs:   args,
	}

	return client, nil
}

// Output runs the ssh command and returns the output
func (client *ExternalClient) Output(pty bool, args ...string) (string, error) {
	args = append(client.BaseArgs, args...)
	cmd := getSSHCmd(client.BinaryPath, pty, args...)
	// for pseudo-tty and sudo to work correctly Stdin must be set to os.Stdin
	if pty {
		cmd.Stdin = os.Stdin
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Shell runs the ssh command, binding Stdin, Stdout and Stderr
func (client *ExternalClient) Shell(pty bool, args ...string) error {
	args = append(client.BaseArgs, args...)
	cmd := getSSHCmd(client.BinaryPath, pty, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func getSSHCmd(binaryPath string, pty bool, args ...string) *exec.Cmd {
	if pty {
		args = append([]string{"-t"}, args...)
	}
	return exec.Command(binaryPath, args...)
}

// ValidUnencryptedPrivateKey parses SSH private key
func ValidUnencryptedPrivateKey(file string) error {
	// Check private key before use it
	fi, err := os.Stat(file)
	if err != nil {
		// Abort if key not accessible
		return err
	}

	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	isEncrypted, err := isEncrypted(buffer)
	if err != nil {
		return fmt.Errorf("Parse SSH key error")
	}

	if isEncrypted {
		return fmt.Errorf("Encrypted SSH key is not permitted")
	}

	_, err = ssh.ParsePrivateKey(buffer)
	if err != nil {
		return fmt.Errorf("Parse SSH key error: %v", err)
	}

	if runtime.GOOS != "windows" {
		mode := fi.Mode()

		// Private key file should have strict permissions
		perm := mode.Perm()
		if perm&0400 == 0 {
			return fmt.Errorf("'%s' is not readable", file)
		}
		if perm&0077 != 0 {
			return fmt.Errorf("permissions %#o for '%s' are too open", perm, file)
		}
	}

	return nil
}

func isEncrypted(buffer []byte) (bool, error) {
	// There is no error, just a nil block
	block, _ := pem.Decode(buffer)
	// File cannot be decoded, maybe it's some unexpected format
	if block == nil {
		return false, fmt.Errorf("Parse SSH key error")
	}

	return x509.IsEncryptedPEMBlock(block), nil
}