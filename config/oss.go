package config

type OSS struct {
	Enable     bool   `mapstructure:"enable" json:"enable" yaml:"enable"`
	UploadCron string `mapstructure:"upload_cron" json:"upload_cron" yaml:"upload_cron"`
	EndPoint   string `mapstructure:"endpoint" json:"endpoint" yaml:"endpoint"`
	UseSSL     bool   `mapstructure:"use-ssl" json:"use-ssl" yaml:"use-ssl"`
	Bucket     string `mapstructure:"bucket" json:"bucket" yaml:"bucket"`
	AccessKey  string `mapstructure:"access-key" json:"access-key" yaml:"access-key"`
	SecretKey  string `mapstructure:"secret-key" json:"secret-key" yaml:"secret-key"`
}
