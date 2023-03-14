package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog"
)

//
/* конфигурации приложения */

func New(logger zerolog.Logger) Configuration {

	return Configuration{logger: logger}
}

type Configuration struct {
	DataBase CfgDataBase
	Server   CfgServer
	Mediator CfgMediator
	logger   zerolog.Logger
}

type CfgMediator struct {
	SecretKey string `env:"KEY"`
}

type CfgDataBase struct {
	DataBaseURI string `env:"DATABASE_URI"`
}

type CfgServer struct {
	RunAddress string `env:"RUN_ADDRESS"`
}

// Заполняет конфиг из переменных окружения
// используемые переменные окружения:
// RUN_ADDRESS  - адрес поднимаемого сервера, например "localhost:8080"
// DATABASE_URI - строка подключения к базе данных
func (c *Configuration) LoadFromEnv() {
	err := env.Parse(&(c.Server))
	if err != nil {
		c.logger.Warn().Msgf("не удалось загрузить конфигурацию сервера из переменных окружения; %v", err)
	}

	err = env.Parse(&(c.DataBase))
	if err != nil {
		c.logger.Warn().Msgf("не удалось загрузить конфигурацию сервера из переменных окружения; %v", err)
	}
}

// функция парсит флаги запуска
func (c *Configuration) LoadFromFlag() {
	flag.StringVar(&(c.Server.RunAddress), "a", "localhost:8080", "Address to start the server (RUN_ADDRESS environment)")
	flag.StringVar(&(c.DataBase.DataBaseURI), "d", "", "connecting string to DB (DATABASE_URI environment)")
	flag.StringVar(&(c.Mediator.SecretKey), "k", "", "Secret key for token generating (KEY environment)")
	flag.Parse()
}
