package utility

import "errors"

func GetDarwinMajor() (m int, err error) {
	err = errors.New("Platform is not darwin")
	return
}
