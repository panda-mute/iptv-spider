package config

type Epg struct {
	Generator string `mapstructure:"generator" json:"generator" yaml:"generator"`
	Source    string `mapstructure:"source" json:"source" yaml:"source"`
	XmlUrl    string `mapstructure:"xml_url" json:"xml_url" yaml:"xml_url"`
	FetchCron string `mapstructure:"fetch_cron" json:"fetch_cron" yaml:"fetch_cron"`
}
