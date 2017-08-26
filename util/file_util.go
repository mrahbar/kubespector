package util

import (
	"os"
	"path/filepath"
)

func InitializeOutputFile(file string) error {
	fd, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return err
	} else {
		fd.Close()
		return nil
	}
}

func WriteOutputFile(filename, data string) error {
	fd, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer fd.Close()

	if _, err = fd.WriteString(data); err != nil {
		return err
	}

	return nil
}

func GetExecutablePath() (string, error) {
	ex, err := os.Executable()
	if err != nil {
		return "", nil
	}

	return filepath.Dir(ex), nil
}
