package config

type Server struct {
	Zap    Zap    `mapstructure:"zap" json:"zap" yaml:"zap"`
	Redis  Redis  `mapstructure:"redis" json:"redis" yaml:"redis"`
	System System `mapstructure:"system" json:"system" yaml:"system"`
	Cache  Cache  `mapstructure:"cache" json:"cache" yaml:"cache"`
	Mysql  Mysql  `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	Stb    Stb    `mapstructure:"stb" json:"stb" yaml:"stb"`
	Epg    Epg    `mapstructure:"epg" json:"epg" yaml:"epg"`
	OSS    OSS    `mapstructure:"oss" json:"oss" yaml:"oss"`
}
