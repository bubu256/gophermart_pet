package main

import (
	"fmt"
	"net/http"

	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/internal/handlers"
	"github.com/bubu256/gophermart_pet/internal/mediator"
	"github.com/bubu256/gophermart_pet/pkg/logger"
	"github.com/bubu256/gophermart_pet/pkg/storage/postgres"
	"github.com/rs/zerolog"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log := logger.New()
	log.Info().Msg("Start application")

	cfg := config.New(log)
	cfg.LoadFromFlag() // загрузка параметров из флагов запуска или значения по умолчанию
	cfg.LoadFromEnv()  // загрузка параметров из переменных окружения

	db := postgres.New(cfg.DataBase, log)
	mediator := mediator.New(db, cfg.Mediator, log)
	handler := handlers.New(mediator, cfg.Server, log)
	log.Info().Msgf("Сервер запущен: %s", cfg.Server.RunAddress)
	err := http.ListenAndServe(cfg.Server.RunAddress, handler.Router)
	if err != nil {
		log.Fatal().Err(err)
	}
	fmt.Print("Exit")
}
