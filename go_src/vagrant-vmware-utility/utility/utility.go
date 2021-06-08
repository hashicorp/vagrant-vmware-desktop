package utility

func IsBigSurMin() bool {
	m, err := GetDarwinMajor()
	if err != nil {
		return false
	}
	return m >= 20
}
