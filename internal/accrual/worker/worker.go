package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/internal/errorapp"
	"github.com/bubu256/gophermart_pet/internal/schema"
	"github.com/bubu256/gophermart_pet/pkg/storage"
	"github.com/rs/zerolog"
)

// пакет воркера который стучится в аккрол сервис и обновляет статусы заказов

type Queue struct {
}

type AccrualWorker struct {
	db            storage.Storage
	logger        zerolog.Logger
	serverAddress string
	// queue *Queue
}

// запускает воркер который в горутине регулярно обновляет статусы заказов
func Run(db storage.Storage, logger zerolog.Logger, cfg config.CfgServer) {

	worker := AccrualWorker{db: db, logger: logger, serverAddress: cfg.RunAddress}
	ticker := time.NewTicker(1 * time.Second)
	// done := make(chan struct{})
	go func() {
		for range ticker.C {
			worker.UpdateStatuses()
		}
	}()
	logger.Info().Msg("Воркер запущен")
}

// получает из базы список заказов ожидающих расчет начислений
//
//	func (a *AccrualWorker) getWaitingOrders(order string) ([]schema.Order, error) {
//		waitingOrders :=
//	}
func (a *AccrualWorker) UpdateStatuses() {
	a.logger.Debug().Msg("запуск UpdateStatuses")
	//получаем все заказы нуждающиеся в обновлении статуса
	waitingOrders, err := a.db.GetWaitingOrders()
	a.logger.Debug().Msgf("заказы ожидающие обновления статуса: %v", waitingOrders)
	if err != nil {
		if errors.Is(err, errorapp.ErrEmptyResult) {
			a.logger.Debug().Msg("нет заказов для обновления статусов;")
			return
		}
		a.logger.Error().Err(err).Msg("ошибка получения заказов в UpdateStatuses; err is here 226456481")
		return
	}
	// проверяем аккрол статус заказов и если требуется обновляем данные в БД
	for _, order := range waitingOrders {
		answerAccrual, err := a.getAccrual(order.Number)
		if err != nil {
			a.logger.Debug().Err(err).Msg("ошибка при получении статуса из аккрол сервиса;")
			continue
		}
		switch answerAccrual.Status {
		case schema.AccrualStatusInvalid:
			err := a.db.SetOrderStatus(answerAccrual.Order, schema.StatusOrderInvalid, 0)
			if err != nil {
				a.logger.Error().Err(err).Msgf("ошибка при попытке обновить статус заказа %s; err is here 2265451221", answerAccrual.Order)
				continue
			}
			a.logger.Info().Msgf("обновлен статус заказа %s на %s;", answerAccrual.Order, schema.StatusOrderInvalid)

		case schema.AccrualStatusProcessed:
			err := a.db.SetOrderStatus(answerAccrual.Order, schema.StatusOrderProcessed, 0)
			if err != nil {
				a.logger.Error().Err(err).Msgf("ошибка при попытке обновить статус заказа %s; err is here 2265213151", answerAccrual.Order)
				continue
			}
			a.logger.Info().Msgf("обновлен статус заказа %s на %s;", answerAccrual.Order, schema.StatusOrderProcessed)

		case schema.AccrualStatusProcessing, schema.AccrualStatusRegistered:
			if order.Status == string(schema.StatusOrderNew) {
				err := a.db.SetOrderStatus(answerAccrual.Order, schema.StatusOrderProcessing, 0)
				if err != nil {
					a.logger.Error().Err(err).Msgf("ошибка при попытке обновить статус заказа %s; err is here 2265213151", answerAccrual.Order)
					continue
				}
			}
			a.logger.Info().Msgf("обновлен статус заказа %s на %s;", answerAccrual.Order, schema.StatusOrderProcessing)
		}
	}
}

// получает статус и сумму начисления из сервиса аккрол
func (a *AccrualWorker) getAccrual(order string) (schema.AnswerAccrualService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	answerAccrual := schema.AnswerAccrualService{}
	request, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s/api/orders/%s", a.serverAddress, order), nil)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	request.Header.Add("Accept", "application/json")
	if err != nil {
		// a.logger.Error().Err(err).Msg("ошибка при создании запроса для сервиса аккрол; err is here 2235498;")
		return answerAccrual, err
	}
	// resp, err := http.DefaultClient.Do(request)
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return answerAccrual, err
	}
	// проверка статус кода
	switch resp.StatusCode {
	case http.StatusInternalServerError:
		return answerAccrual, errors.New("accraul status code is 500;")
	case http.StatusNoContent:
		return answerAccrual, errors.New("заказ не зарегистрирован в системе расчёта;")
	case http.StatusTooManyRequests:
		return answerAccrual, errors.New("превышено количество запросов к сервису превышено количество запросов к сервису;")
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		a.logger.Warn().Msg("неожиданный тип ответа от сервиса аккрол; i am here 22345354;")
	}
	a.logger.Debug().Str("Content-Type", resp.Header.Get("Content-Type")).Msg("")
	// читаем ответ и возвращаем результат
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	a.logger.Debug().Str("body", string(body)).Msg("")
	if err != nil {
		return answerAccrual, err
	}
	err = json.Unmarshal(body, &answerAccrual)
	if err != nil {
		return answerAccrual, err
	}
	return answerAccrual, nil
}
