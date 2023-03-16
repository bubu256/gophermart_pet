package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/bubu256/gophermart_pet/config"
	"github.com/bubu256/gophermart_pet/internal/errorapp"
	"github.com/bubu256/gophermart_pet/internal/schema"
	"github.com/bubu256/gophermart_pet/pkg/storage"
	"github.com/jackc/pgerrcode"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
)

// все взаимодействия с БД

type PosgresDB struct {
	storage.Storage
	URI    string
	DB     *sql.DB
	logger zerolog.Logger
}

func New(cfg config.CfgDataBase, logger zerolog.Logger) storage.Storage {
	db, err := sql.Open("pgx", cfg.DataBaseURI)
	if err != nil {
		logger.Error().Err(err)
	}

	pdb := &PosgresDB{
		DB:     db,
		URI:    cfg.DataBaseURI,
		logger: logger,
	}

	err = pdb.Ping()
	if err != nil {
		logger.Fatal().Err(err).Msg("DB not available; error is here 58545346")
	}
	return pdb
}

func (p *PosgresDB) SetUser(user, passwordHash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "INSERT INTO users(login, password_hash) VALUES ($1, $2)"
	_, err := p.DB.ExecContext(ctx, query, user, passwordHash)
	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			return errorapp.ErrDuplicate
		}
		return err
	}
	return nil
}

func (p *PosgresDB) GetUserID(login string, hashPassword string) (userID uint16, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "select user_id from users where login = $1 and password_hash = $2"
	var id uint16
	err = p.DB.QueryRowContext(ctx, query, login, hashPassword).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errorapp.ErrWrongLoginPassword
		}
		return 0, err
	}
	return id, nil
}

// добавляет новый заказ для пользователя
func (p *PosgresDB) SetOrder(userID uint16, number string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := "INSERT INTO orders(user_id, number) VALUES ($1, $2)"
	_, err := p.DB.ExecContext(ctx, query, userID, number)
	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			return errorapp.ErrDuplicate
		}
		return err
	}
	return nil
}

// устанавливает статус расчета заказа
func (p *PosgresDB) SetOrderStatus(number string, status schema.StatusOrder, accrual float32) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
		INSERT INTO order_status(order_id, status_id, accrual)
		select o.order_id, s.status_id, $3
		from orders o, status s where s.name = $1 and o.number = $2
		`
	_, err := p.DB.ExecContext(ctx, query, status, number, accrual)
	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			return errorapp.ErrDuplicate
		}
		return err
	}
	return nil
}

// возвращает все заказы в структуре []schema.Order.
// номер, статус, начисление, датавремя добавления
func (p *PosgresDB) GetOrders(userID uint16) ([]schema.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	query := `
		select num, stat, acc, upload
		FROM (
			SELECT distinct on (os.order_id) os.order_id, 
				o.number num, 
				s.name stat, 
				os.accrual acc, 
				o.datetime upload
			FROM orders o JOIN order_status os ON o.order_id = os.order_id and o.user_id == $1
				JOIN status s ON s.status_id = os.status_id
			ORDER BY os.datetime DESC
		) q1
		ORDER BY upload ASC
		`
	rows, err := p.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	result := make([]schema.Order, 0)
	for rows.Next() {
		order := schema.Order{}
		err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			p.logger.Error().Err(err)
			continue
		}
		result = append(result, order)
	}
	if err := rows.Err(); err != nil {
		p.logger.Error().Err(err).Msg("error is here 346842419846")
	}

	return result, nil
}

// возвращает баланс и общую сумму потраченных баллов
func (p *PosgresDB) GetBalance(userID uint16) (schema.Balance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
	SELECT SUM(bf.amount), SUM(
			CASE
				WHEN bf.amount < 0 THEN -bf.amount
			END
		)
	FROM bonus_flow bf
	WHERE bf.user_id = $1
	`
	balance := schema.Balance{}
	err := p.DB.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		return balance, err
	}
	return balance, nil
}

// движение бонусов
func (p *PosgresDB) SetBonusFlow(userID uint16, orderNumber string, amount float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
		INSERT INTO bonus_flow(user_id, order_number, amount)
		VALUES ($1, $2, $3)
		`
	insertResult, err := p.DB.ExecContext(ctx, query, userID, orderNumber, amount)
	if err != nil {
		return err
	}
	insertCount, err := insertResult.RowsAffected()
	if err != nil {
		return err
	}
	if insertCount == 0 {
		return errorapp.ErrEmptyInsert
	}
	return nil
}

// возвращает айди юзера добавившего заказ
func (p *PosgresDB) GetUserIDfromOrders(numberOrder string) (uint16, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
	SELECT user_id FROM orders WHERE number = $1 LIMIT 1
	`
	var userID uint16
	err := p.DB.QueryRowContext(ctx, query, numberOrder).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errorapp.ErrEmptyResult
		}
		return 0, err
	}
	return userID, nil
}

// проверка доступности БД
func (p *PosgresDB) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	db, err := sql.Open("pgx", p.URI)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.PingContext(ctx)
}
