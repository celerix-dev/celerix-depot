package storage

import (
	"io"
	"os"
	"path/filepath"
)

func StoreFile(reader io.Reader, storageDir, fileName string) (string, int64, error) {
	if _, err := os.Stat(storageDir); os.IsNotExist(err) {
		err := os.MkdirAll(storageDir, 0755)
		if err != nil {
			return "", 0, err
		}
	}

	filePath := filepath.Join(storageDir, fileName)
	out, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer out.Close()

	size, err := io.Copy(out, reader)
	if err != nil {
		return "", 0, err
	}

	return filePath, size, nil
}

func GetFile(filePath string) (io.ReadCloser, error) {
	return os.Open(filePath)
}

func DeleteFile(filePath string) error {
	return os.Remove(filePath)
}
