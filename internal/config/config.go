package config

import (
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	BotToken     string `yaml:"bot_token"`
	VoterWallet  string `yaml:"voter_wallet"`
	KeyChainPass string `yaml:"keychain_password"`
	DaemonPath   string `yaml:"deamon_path"`
}

func ParseConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Errorf("failed to read file %s due to %v", path, err)
		return nil, errors.Wrapf(err, "failed to read file %s", path)
	}
	conf := &Config{}
	if err := yaml.Unmarshal(content, conf); err != nil {
		log.Errorf("failed to parse config file %s due to %v", path, err)
		return nil, errors.Wrapf(err, "failed to parse config file %s", path)
	}
	return conf, nil
}
