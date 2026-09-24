package config

type Stb struct {
	SN       string `mapstructure:"sn" json:"sn" yaml:"sn"`
	UID      string `mapstructure:"uid" json:"uid" yaml:"uid"`
	MAC      string `mapstructure:"mac" json:"mac" yaml:"mac"`
	IP       string `mapstructure:"ip" json:"ip" yaml:"ip"`
	Type     string `mapstructure:"type" json:"type" yaml:"type"`
	AuthHost string `mapstructure:"auth_host" json:"auth_host" yaml:"auth_host"`
}
