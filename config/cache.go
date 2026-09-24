package config

type Cache struct {
	Type       string `mapstructure:"type" json:"type" yaml:"type"`                                  // 数据库类型:memory(默认)|redis
	Prefix     string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`                            // key 前缀
	Interval   int    `mapstructure:"memory_interval" json:"memory_interval" yaml:"memory_interval"` // key 前缀
	DefTimeOut int    `mapstructure:"default_timeout" json:"default_timeout" yaml:"default_timeout"` // 默认超时时间
}
