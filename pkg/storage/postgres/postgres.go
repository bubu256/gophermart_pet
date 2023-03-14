package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

// type Storage interface {
// 	SetUser(user, passwordHash string) error
// 	GetUserID(login string, hash string) (userID uint16, err error)
// 	SetOrder(userID uint16, number string) error
// 	SetOrderStatus(number string, status string) error
// 	GetOrders(userID uint16) ([]schema.Order, error)
// 	GetBalance(userID uint16) (schema.Balance, error)
// 	SetBonusFlow(userID uint16, amount float64) error
// }

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
	// err = p.SetOrderStatus(number, "NEW")
	// if err != nil {
	// 	return err
	// }
	return nil
}

func (p *PosgresDB) SetOrderStatus(number string, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
		INSERT INTO order_status(order_id, status_id)
		select o.order_id, s.status_id
		from orders o, status s where s.name = $1 and o.number = $2
		`
	_, err := p.DB.ExecContext(ctx, query, status, number)
	if err != nil {
		if strings.Contains(err.Error(), pgerrcode.UniqueViolation) {
			return errorapp.ErrDuplicate
		}
		return err
	}
	return nil
}

func (p *PosgresDB) GetOrders(userID uint16) ([]schema.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()

	query := `
		SELECT o.number, status.name, bf.amount, os.datetime
		FROM orders o LEFT JOIN bonus_flow as bf ON bf.order_id = o.order_id and o.user_id = $1
			JOIN (SELECT order_id, max(status_id) from order_status group by order_id) 
				as os ON o.order_id = os.order_id
			JOIN status s ON s.status_id = os.status_id 
		WHERE os.
		ORDER BY o.order_id ASC
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

func (p *PosgresDB) SetBonusFlow(userID uint16, orderNumber string, amount float64) error {
	err := p.SetOrder(userID, orderNumber)
	if err != nil {
		return fmt.Errorf("ошибка при попытке списания бонусов; %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
		INSERT INTO bonus_flow(user_id, order_id, amount)
		select $1, o.order_id, $3
		from orders o
		where o.user_id = $1 and o.number = $2;
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
