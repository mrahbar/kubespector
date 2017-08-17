package integration

import (
	"fmt"
	"github.com/mrahbar/kubernetes-inspector/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"strings"
	"io/ioutil"
	"encoding/pem"
	"path/filepath"
	"crypto/x509"
	"runtime"
	"golang.org/x/crypto/ssh"
	"os/exec"
)

func UnmarshalConfig() types.Config {
	var config types.Config
	err := viper.Unmarshal(&config)

	if err != nil {
		PrettyPrintErr("Unable to decode config: %v", err)
		os.Exit(1)
	}

	return config
}

func GetNodeAddress(node types.Node) string {
	nodeAddress := node.IP
	if nodeAddress == "" {
		nodeAddress = node.Host
	}

	return nodeAddress
}

func IsNodeAddressValid(node types.Node) bool {
	if node.Host == "" && node.IP == "" {
		return false
	} else {
		return true
	}
}

func NodeEquals(n1, n2 types.Node) bool {
	if IsNodeAddressValid(n1) && IsNodeAddressValid(n2) {
		if n1.IP != "" && n2.IP != "" {
			return n1.IP == n2.IP
		} else if n1.Host != "" && n2.Host != "" {
			return n1.Host == n2.Host
		} else {
			return false
		}
	} else {
		return false
	}
}

func ToNodeLabel(node types.Node) string {
	if !IsNodeAddressValid(node) {
		return ""
	}

	label := fmt.Sprintf("%s", node.Host)

	if node.IP != "" {
		label = fmt.Sprintf("%s (%s)", label, node.IP)
	}

	return label
}

func FindGroupByName(clustergroups []types.ClusterGroup, name string) types.ClusterGroup {
	for _, group := range clustergroups {
		if strings.EqualFold(group.Name, name) {
			return group
		}
	}

	return types.ClusterGroup{}
}

func ElementInArray(array []string, element string) bool {
	contains := false
	for _, v := range array {
		if v == element {
			contains = true
			break
		}
	}

	return contains
}

func CheckRequiredFlags(cmd *cobra.Command, _ []string) error {
	f := cmd.Flags()
	requiredError := false
	flagName := ""

	f.VisitAll(func(flag *pflag.Flag) {
		requiredAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(requiredAnnotation) == 0 {
			return
		}

		flagRequired := requiredAnnotation[0] == "true"

		if flagRequired && !flag.Changed {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		return fmt.Errorf("Required flag `%s` has not been set", flagName)
	}

	return nil
}

// validUnencryptedPrivateKey parses SSH private key
func validUnencryptedPrivateKey(file string, debug bool) (string, error) {
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