package confer

import (
	"goframe/pkg/util"
)

func ConfigNmidGetString(key string, defaultConfig string) string {
	config := ConfigNmidGet(key)
	if config == nil {
		return defaultConfig
	} else {
		configStr := config.(string)
		if util.UtilIsEmpty(configStr) {
			configStr = defaultConfig
		}
		return configStr
	}
}

func ConfigNmidGetInt(key string, defaultConfig int) int {
	config := ConfigNmidGet(key)
	if config == nil {
		return defaultConfig
	} else {
		configInt := config.(int)
		if configInt == 0 {
			configInt = defaultConfig
		}
		return configInt
	}
}

func ConfigNmidGet(key string) interface{} {
	globalConfig.RLock()
	defer globalConfig.RUnlock()
	//将配置文件中的app字段转为map
	config, exists := globalConfig.Nmid[key]
	if !exists {
		return nil
	}

	return config
}
