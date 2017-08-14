package test

// IsOnCi returns if we're running on a ci system where no config is mounted
func IsOnCi(path string) bool {
	return true
	/*_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true
		}
		panic(err)
	} */
}
