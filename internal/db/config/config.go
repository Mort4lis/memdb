package config

type Config struct {
	Logging Logging `yaml:"logging"`
}

type Logging struct {
	Level  string `env-default:"info" yaml:"level"`
	Format string `env-default:"text" yaml:"format"`
}
