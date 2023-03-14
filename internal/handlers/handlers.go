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
	r := chi.NewRouter()
	// подключение ручек
	r.Post("/api/user/register", h.UserRegister)
	r.Post("/api/user/login", h.UserLogin)

	h.Router.Mount("/", r)
}

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
	cookie := http.Cookie{Name: "token", Value: token, Path: "/"}
	http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
}

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
	cookie := http.Cookie{Name: "token", Value: token, Path: "/"}
	http.SetCookie(w, &cookie)
	w.WriteHeader(http.StatusOK)
}
