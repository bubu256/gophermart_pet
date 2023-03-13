package main

import (
	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/pkg/logger"
	"github.com/bubu256/gophermart_pet/pkg/storage/postgres"
)

func main() {
	log := logger.New()
	log.Info().Msg("Start application")

	cfg := config.New(log)
	cfg.LoadFromFlag() // загрузка параметров из флагов запуска или значения по умолчанию
	cfg.LoadFromEnv()  // загрузка параметров из переменных окружения

	db := postgres.New(cfg.DataBase, log)
	// service := shortener.New(dataStorage, cfg.Service)
	// handler := handlers.New(service, cfg.Server)
	// log.Println("Сервер:", cfg.Server.ServerAddress)
	// log.Fatal(http.ListenAndServe(cfg.Server.ServerAddress, handler.Router))
}
