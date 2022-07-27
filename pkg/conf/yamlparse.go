package conf

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"nmid-v2/pkg/model"
	"os"
)

func ReadFileData(FileUrl string) (fileData []byte, err error) {
	if _, err = os.Stat(FileUrl); os.IsNotExist(err) {
		err = errors.New("The yaml configuration file does not exist")
		return
	}

	fileData, err = ioutil.ReadFile(FileUrl)
	return
}

func ParseYaml4Bytes(orignData []byte) (sconfig model.ServerConfig, err error) {
	if len(orignData) == 0 {
		err = errors.New("yaml source data is empty")
		return
	}

	err = yaml.Unmarshal(orignData, &sconfig)

	return
}

//ParseYaml4File data source is file path.
func ParseYaml4File(yamlFileUrl string) (sconfig model.ServerConfig, err error) {
	var fileData []byte

	fileData, err = ReadFileData(yamlFileUrl)

	if err != nil {
		return
	}

	sconfig, err = ParseYaml4Bytes(fileData)

	return
}
