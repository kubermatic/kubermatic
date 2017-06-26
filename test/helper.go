package test

import "os"

// IsOnCi returns if we're running on a ci system where no config is mounted
func IsOnCi() bool {
	_, err := os.Stat("../../config/kubermatic/static/master/")
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		panic(err)
	}

	return false
}
