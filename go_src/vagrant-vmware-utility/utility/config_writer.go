package utility

import (
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func WriteConfigFile(file string, content interface{}) (err error) {
	data, err := GenerateConfigFile(content)
	if err != nil {
		return
	}
	d := filepath.Dir(file)
	if err = os.MkdirAll(d, 0755); err != nil {
		return
	}
	o, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer o.Close()
	_, err = o.Write(data)
	return
}

func GenerateConfigFile(content interface{}) ([]byte, error) {
	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(content, f.Body())
	return f.Bytes(), nil
}
