package conf

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/HughNian/nmid/pkg/model"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var SConfig model.ServerConfig
var mutex sync.RWMutex

func Init() (err error) {
	configURL := "config/server.yaml" //这个路径相对于main函数文件的路径

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile(configURL)
	confContent, err := ioutil.ReadFile(configURL)
	if err != nil {
		log.Fatalf(fmt.Sprintf("Read config file fail: %s", err.Error()))
	}
	err = v.ReadConfig(strings.NewReader(os.ExpandEnv(string(confContent))))
	if err != nil {
		log.Fatalf("Fatal error config file: %s", err.Error())
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Println("config file changed:", e.Name)
		if err := v.Unmarshal(&SConfig); err != nil {
			log.Println(err)
		}
	})
	if err := v.Unmarshal(&SConfig); err != nil {
		log.Fatalf("Unmarshal error config file: %s", err.Error())
		return err
	}

	return
}

func GetGlobalConfig() model.ServerConfig {
	mutex.RLock()
	defer mutex.RUnlock()
	return SConfig
}

func GetConfig() model.ServerConfig {
	sConfig, err := ParseYaml4File("config/server.yaml") //这个路径相对于main函数文件的路径
	if err != nil {
		log.Println(err.Error())
	}

	return sConfig
}
