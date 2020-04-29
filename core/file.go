package core

import (
	"fmt"
	"os"
)

func SaveToFile(path string, data string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file %v failed: %v", path, err)
	}

	_, err = f.Write([]byte(data))
	if err != nil {
		return fmt.Errorf("write to file %v failed: %v", path, err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("close file %v failed: %v", path, err)
	}

	return nil
}
