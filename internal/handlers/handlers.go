package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/internal/errorapp"
	"github.com/bubu256/gophermart_pet/internal/mediator"
	"github.com/bubu256/gophermart_pet/internal/schema"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// хендлеры и роутинг

type Handler struct {
	Mediator *mediator.Mediator
	logger   zerolog.Logger
	Router   *chi.Mux
}

func New(mediator *mediator.Mediator, cfg config.CfgServer, logger zerolog.Logger) *Handler {
	handler := Handler{Mediator: mediator, logger: logger, Router: chi.NewRouter()}
	handler.MountBaseRouter()
	return &handler
}

func (h *Handler) MountBaseRouter() {
	// хендлеры с проверкой токена в мидлваре
	privateRouter := chi.NewRouter()
	privateRouter.Use(h.MiddlewareTokenChecker)
	privateRouter.Post("/api/user/orders", h.PostUserOrders)
	privateRouter.Get("/api/user/orders", h.GetUserOrders)
	privateRouter.Get("/api/user/balance", h.GetUserBalance)
	h.Router.Mount("/", privateRouter)

	// хендлеры без мидлвара на проверку токена
	h.Router.Post("/api/user/register", h.UserRegister)
	h.Router.Post("/api/user/login", h.UserLogin)
}

// ============Middlewares===============//

// Проверяет токен и возвращая 401 если пользователь не авторизован
func (h *Handler) MiddlewareTokenChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieToken, err := r.Cookie("token")
		// h.logger.Debug().Msgf("Token %s", cookieToken.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !h.Mediator.CheckToken(cookieToken.Value) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

//============Middlewares===============//
//......................................//
//============Handlers==================//

// регистрации пользователя
func (h *Handler) UserRegister(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("error is here 446846541")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	loginPassword := schema.LoginPassword{}
	err = json.Unmarshal(body, &loginPassword)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// отдаем медиатору для хеширования и записи в бд
	err = h.Mediator.SetNewUser(loginPassword)
	if err != nil {
		if errors.Is(err, errorapp.ErrDuplicate) {
			w.WriteHeader(http.StatusConflict)
			return
		}
		h.logger.Error().Err(err).Msg("error is here 65151321")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// берем токен авторизации и пишем в куки
	token, err := h.Mediator.GetTokenAuthorization(loginPassword)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.logger.Error().Err(err).Msg("ошибка аутентификации после регистрации пользователя; error is here 3468453;")
		return
	}
	cookieToken := http.Cookie{Name: "token", Value: token, Path: "/"}
	http.SetCookie(w, &cookieToken)
	w.WriteHeader(http.StatusOK)
}

// авторизация пользователя
func (h *Handler) UserLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("error is here 446846541")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	loginPassword := schema.LoginPassword{}
	err = json.Unmarshal(body, &loginPassword)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// берем токен авторизации и пишем в куки
	token, err := h.Mediator.GetTokenAuthorization(loginPassword)
	if err != nil {
		if errors.Is(err, errorapp.ErrWrongLoginPassword) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		h.logger.Error().Err(err).Msg("ошибка при выдаче токена; error is here 168131685")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cookieToken := http.Cookie{Name: "token", Value: token, Path: "/"}
	http.SetCookie(w, &cookieToken)
	w.WriteHeader(http.StatusOK)
}

// Загрузка номера заказа
// Хендлер: POST /api/user/orders.
func (h *Handler) PostUserOrders(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "text/plain" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	cookieToken, err := r.Cookie("token")
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка при чтении токена из кук; error is here 168131685")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка при чтении тела запроса; err is here 32135354;")
		w.WriteHeader(http.StatusInternalServerError)
	}
	numberOrder := string(body)
	// проверка номера
	if !mediator.ValidateOrderNumber(numberOrder) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	// добавляем заказ
	err = h.Mediator.SetNewOrder(cookieToken.Value, numberOrder)
	switch {
	case errors.Is(err, errorapp.ErrDuplicate):
		// номер уже добавлен другим пользователем
		w.WriteHeader(http.StatusConflict)
		return
	case errors.Is(err, errorapp.ErrAlreadyAdded):
		// пользователь уже добавлял этот заказ
		w.WriteHeader(http.StatusOK)
		return
	case err != nil:
		h.logger.Error().Err(err).Msg("ошибка при добавлении нового заказ; err is here 65456121354")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)

}

// Получение списка загруженных номеров заказов
// Хендлер: GET /api/user/orders.
func (h *Handler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	cookieToken, err := r.Cookie("token")
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка при чтении токена из кук; error is here 16813145685")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	orders, err := h.Mediator.GetUserOrders(cookieToken.Value)
	if err != nil {
		if errors.Is(err, errorapp.ErrEmptyResult) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.logger.Error().Err(err).Msg("ошибка при попытке получить список заказов; err is here 643154;")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	ordersByte, err := json.Marshal(orders)
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка кодирования списка заказов в json; err is here 64331154;")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(ordersByte)
}

// GET /api/user/balance — получение текущего баланса счёта баллов лояльности пользователя;
func (h *Handler) GetUserBalance(w http.ResponseWriter, r *http.Request) {
	cookieToken, err := r.Cookie("token")
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка при чтении токена из кук; error is here 16813145685;")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	balance, err := h.Mediator.GetUserBalance(cookieToken.Value)
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка при получении баланса; err is here 64815168';")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	byteBalance, err := json.Marshal(balance)
	if err != nil {
		h.logger.Error().Err(err).Msg("ошибка кодирования в json; err is here 5465135;")
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(byteBalance)
}

//============Handlers==================//
//......................................//
