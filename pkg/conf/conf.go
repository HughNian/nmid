package conf

import "log"

func GetConfig() ServerConfig {
	sConfig, err := ParseYaml4File("config/server.yaml") //这个路径相对于main函数文件的路径
	if err != nil {
		log.Println(err.Error())
	}

	return sConfig
}
