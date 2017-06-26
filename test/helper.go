package test

import "os"

// IsOnCi returns if we're running on a ci system where no config is mounted
func IsOnCi(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		panic(err)
	}

	return false
}
