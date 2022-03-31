package config

import (
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/joho/godotenv"
)

type Config struct {
	ReadTimeout       int
	WriteTimeout      int
	ReadHeaderTimeout int
	log               *log.Logger
}

func (c *Config) loadConfigFile(file string) error {
	var err error
	var myEnv map[string]string

	myEnv, err = godotenv.Unmarshal(file)
	if err != nil {
		return err
	}
	c.ReadTimeout, err = strconv.Atoi(myEnv["READTIMEOUT"])
	if err != nil {
		c.log.WithField("config", "readtimeout").Error(err)
	}
	c.WriteTimeout, err = strconv.Atoi(myEnv["WRITETIMEOUT"])
	if err != nil {
		c.log.WithField("config", "writetimeout").Error(err)
	}
	c.ReadHeaderTimeout, err = strconv.Atoi(myEnv["READHEADERTIMEOUT"])
	if err != nil {
		c.log.WithField("config", "readheadertimeout").Error(err)
	}

	return nil
}

func LoadConfig(file string, log *log.Logger) (Config, error) {
	var c Config
	c.log = log
	err := c.loadConfigFile(file)
	if err != nil {
		return c, err
	}
	return c, nil
}
