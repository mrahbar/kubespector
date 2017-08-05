package integration

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/mrahbar/kubernetes-inspector/types"
	"io/ioutil"
)

type SecureShellBinary struct {
	binaryName string
	binaryPath string
	portArg    string
}

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

type Client interface {
	Output(pty bool, debug bool, args ...string) (string, error)
	Shell(pty bool, args ...string) error
}

type ExternalClient struct {
	BaseArgs   []string
	BinaryPath string
	cmd        *exec.Cmd
}

func PerformSSHCmd(sshOpts types.SSHConfig, node types.Node, cmd string, debug bool) (string, error) {
	nodeAddress := GetNodeAddress(node)

	client, err := newSSHClient(fmt.Sprintf("%s@%s", sshOpts.User, nodeAddress), sshOpts.Port, sshOpts.Key,
		strings.FieldsFunc(sshOpts.Options, func(r rune) bool {
			return r == ' ' || r == ','
		}), debug)

	if err != nil {
		msg := fmt.Sprintf("Error creating SSH client for host %s: %v", nodeAddress, err)
		PrettyPrintErr(msg)
		return "", err
	}

	if sshOpts.Sudo && !strings.HasPrefix(cmd, "sudo") {
		cmd = "sudo " + cmd
	}

	return client.Output(sshOpts.Pty, debug, cmd)
}

// newSSHClient verifies ssh is available in the PATH and returns an SSH client
func newSSHClient(remoteHost string, port int, key string, options []string, debug bool) (Client, error) {
	return newClient(SecureShellBinary{binaryName: "ssh", portArg: "-p"}, remoteHost, port, key, options, debug)
}

// newClient verifies ssh is available in the PATH and returns an SSH client
func newClient(binary SecureShellBinary, remoteHost string, port int, key string, options []string, debug bool) (Client, error) {
	key, err := ValidUnencryptedPrivateKey(key, debug)
	if err != nil {
		return nil, err
	}

	binaryPath, err := exec.LookPath(binary.binaryName)
	if err != nil {
		return nil, fmt.Errorf("command not found: %s", binary)
	}

	binary.binaryPath = binaryPath
	return newExternalClient(binary, remoteHost, port, key, options)
}

func newExternalClient(binary SecureShellBinary, host string, port int, key string, options []string) (*ExternalClient, error) {
	// Get default args with user and host
	args := append(baseSSHArgs, options...)
	// set port
	args = append(args, binary.portArg, fmt.Sprintf("%d", port))
	// set key
	args = append(args, "-i", key)
	// set host
	args = append(args, host)

	client := &ExternalClient{
		BinaryPath: binary.binaryPath,
		BaseArgs:   args,
	}

	return client, nil
}

// Output runs the ssh command and returns the output
func (client *ExternalClient) Output(pty bool, debug bool, args ...string) (string, error) {
	args = append(client.BaseArgs, args...)
	cmd := executeCmd(client.BinaryPath, pty, args...)
	if debug {
		cmdDebug := append([]string{}, cmd.Args...)
		fmt.Printf("Executing command: %s\n", cmdDebug)
	}
	// for pseudo-tty and sudo to work correctly Stdin must be set to os.Stdin
	if pty {
		cmd.Stdin = os.Stdin
	}

	output, err := cmd.CombinedOutput()
	if debug {
		fmt.Printf("Result of command:\n\tResult: %s\tErr: %s\n", strings.TrimSpace(string(output)), err)
	}

	return strings.TrimSpace(string(output)), err
}

// Shell runs the ssh command, binding Stdin, Stdout and Stderr
func (client *ExternalClient) Shell(pty bool, args ...string) error {
	args = append(client.BaseArgs, args...)
	cmd := executeCmd(client.BinaryPath, pty, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func executeCmd(binaryPath string, pty bool, args ...string) *exec.Cmd {
	if pty {
		args = append([]string{"-tt"}, args...)
	}
	return exec.Command(binaryPath, args...)
}

// ValidUnencryptedPrivateKey parses SSH private key
func ValidUnencryptedPrivateKey(file string, debug bool) (string, error) {
	// Check private key before use it
	fi, err := os.Stat(file)
	if err != nil {
		// Abort if key not accessible
		return "", err
	}

	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	isEncrypted, err := isEncrypted(buffer)
	if err != nil {
		return "", fmt.Errorf("Parse SSH key error")
	}

	if isEncrypted {
		return "", fmt.Errorf("Encrypted SSH key is not permitted")
	}

	// Check if x/crypto/ssh can parse the key
	_, err = ssh.ParsePrivateKey(buffer)
	if err != nil {
		//return fmt.Errorf("Parse SSH key error: %v", err)
		file, _ = convertBerToDerFormat(buffer, debug)
		if err != nil {
			fi, err = os.Stat(file)
		}
	}

	if runtime.GOOS != "windows" {
		mode := fi.Mode()

		// Private key file should have strict permissions
		perm := mode.Perm()
		if perm&0400 == 0 {
			return "", fmt.Errorf("'%s' is not readable", file)
		}
		if perm&0077 != 0 {
			return "", fmt.Errorf("permissions %#o for '%s' are too open", perm, file)
		}
	}

	return file, nil
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

// Work around for https://github.com/mitchellh/packer/issues/2526
func convertBerToDerFormat(ber []byte, debug bool) (string, error) {
	if debug {
		// Can't parse the key, maybe it's BER encoded. Try to convert it with OpenSSL.
		fmt.Println("Couldn't parse SSH key, trying work around for [GH-2526].")
	}

	derFilePath := filepath.Join(".", ".kubernetes-inspector-privatekey-formated.der")
	if _, err := os.Stat(derFilePath); err == nil {
		if debug {
			fmt.Println("DER formated private key file already exists.")
		}
		return derFilePath, nil
	}

	openSslPath, err := exec.LookPath("openssl")
	if err != nil {
		return "", fmt.Errorf("Couldn't find OpenSSL, aborting work around: %s\n", err)
	}

	berFormattedKey, err := ioutil.TempFile("", "kubernetes-inspector-ber-privatekey-")
	defer os.Remove(berFormattedKey.Name())
	if err != nil {
		return "", err
	}

	ioutil.WriteFile(berFormattedKey.Name(), ber, os.ModeAppend)
	derFormattedKey, err := os.OpenFile(derFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}

	args := []string{"rsa", "-in", berFormattedKey.Name(), "-out", derFormattedKey.Name()}
	if debug {
		fmt.Printf("Executing: %s %v\n", openSslPath, args)

	}
	if err := exec.Command(openSslPath, args...).Run(); err != nil {
		return "", fmt.Errorf("OpenSSL failed with error: %s\n", err)
	}

	return derFormattedKey.Name(), nil
}
