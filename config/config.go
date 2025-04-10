package config

import (
	"log"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	ServConf ServerCfg   `yaml:"server"`
	DBConf   DatabaseCfg `yaml:"database"`
}

type ServerCfg struct {
	HostgRPC string        `yaml:"hostgRPC" env:"HOSTgPRC" env-default:":8080"`
	HostREST string        `yaml:"hostREST" env:"HOSTREST" env-default:":50051"`
	Timeout  time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"10s"`
}

type DatabaseCfg struct {
	Port     string `yaml:"port" env:"DB_PORT" env-default:"5432"`
	User     string `yaml:"user" env:"DB_USER" env-default:"postgres"`
	Password string `yaml:"password" env:"DB_PASSWORD" env-default:"1234"`
	DBName   string `yaml:"dbname" env:"DB_NAME" env-default:"postgres"`
	Host     string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
}

func MustLoad() *Config {
	cfg := Config{}
	err := cleanenv.ReadConfig("config.yaml", &cfg)
	if err != nil {
		log.Fatalf("cannot read config: %s", err)
	}
	return &cfg
}
