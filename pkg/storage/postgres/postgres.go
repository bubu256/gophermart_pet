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
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	logger.Info().Msg("Подключение с БД готово;")

	pdb.migrateUp()
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
// статус PROCESSED начиляет бонусы
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
	// если статус PROCESSED
	// зачисляем бонусы на счет
	if status == schema.StatusOrderProcessed {
		query2 := `
			INSERT INTO bonus_flow(user_id, order_number, amount)
			SELECT user_id, $1, $2
			FROM orders WHERE number = $1
			`
		insertResult, err := p.DB.ExecContext(ctx, query2, number, accrual)
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
	}
	return nil
}

// возвращает все заказы в структуре []schema.Order.
// номер, статус, начисление, датавремя добавления
func (p *PosgresDB) GetOrders(userID uint16) ([]schema.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
	SELECT distinct on (os.order_id) 
		o.number num, 
		s.name stat, 
		os.accrual acc, 
		o.datetime upload
	FROM orders o JOIN order_status os ON o.order_id = os.order_id and o.user_id = $1
		JOIN status s ON s.status_id = os.status_id
	ORDER BY os.order_id, os.datetime desc
	`
	rows, err := p.DB.QueryContext(ctx, query, userID)
	// p.logger.Debug().Err(err).Msg("")
	if err != nil {
		// это почему то не работает(
		// if errors.Is(err, sql.ErrNoRows) {
		// 	return nil, errorapp.ErrEmptyResult
		// }
		return nil, err
	}

	result := make([]schema.Order, 0)
	for rows.Next() {
		order := schema.Order{}
		err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt.Time)
		if err != nil {
			p.logger.Error().Err(err).Msg("err is here 16541321;")
			continue
		}
		// p.logger.Debug().Msgf("%v", order)
		result = append(result, order)
	}
	if err := rows.Err(); err != nil {
		p.logger.Error().Err(err).Msg("error is here 346842419846")
	}
	if len(result) == 0 {
		return result, errorapp.ErrEmptyResult
	}
	return result, nil
}

// возвращает баланс и общую сумму потраченных баллов
func (p *PosgresDB) GetBalance(userID uint16) (schema.Balance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
	SELECT coalesce(sum(bf.amount), 0), 
		coalesce(sum(
			CASE
				WHEN bf.amount < 0 THEN -bf.amount
			END
			), 0)
	FROM bonus_flow bf
	WHERE bf.user_id = $1
	`
	balance := schema.Balance{}
	err := p.DB.QueryRowContext(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err != nil {
		p.logger.Error().Err(err).Msg("ошибка при получении из базу баланса; err is here 6843545;")
		return balance, err
	}
	return balance, nil
}

// движение бонусов
func (p *PosgresDB) SetBonusFlow(userID uint16, orderNumber string, amount float32) error {
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

// возвращает список выводов пользователя
func (p *PosgresDB) GetBonusFlow(userID uint16) ([]schema.OrderSum, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
	select order_number, amount * (-1), datetime
	from bonus_flow bf 
	where user_id = $1 and amount < 0
	order by datetime 
	`
	rows, err := p.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	result := make([]schema.OrderSum, 0)
	for rows.Next() {
		orderSum := schema.OrderSum{}
		err := rows.Scan(&orderSum.Order, &orderSum.Sum, &orderSum.ProcessedAt.Time)
		if err != nil {
			p.logger.Error().Err(err).Msg("err is here 165541321;")
			continue
		}
		result = append(result, orderSum)
	}
	if err := rows.Err(); err != nil {
		p.logger.Error().Err(err).Msg("error is here 3468423419846")
	}
	if len(result) == 0 {
		return result, errorapp.ErrEmptyResult
	}

	return result, nil
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

// up миграции БД
func (p *PosgresDB) migrateUp() error {
	m, err := migrate.New(
		"file://migrations",
		p.URI,
	)
	if err != nil {
		p.logger.Fatal().Err(err).Msg("удалось подключиться к БД для выполнения миграции; err is here 9879516;")
		return err
	}
	defer m.Close()
	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		p.logger.Info().Msg("Миграция БД не требуется;")
		return nil
	}
	if err != nil {
		p.logger.Error().Err(err).Msg("ошибка при выполнении миграции; err is here 0321518")
	}
	p.logger.Info().Msgf("Миграция применена к БД; %v", m)
	return nil
}

// возвращает номера и статусы заказов ожидающих расчета начисления (только заказы в статусе NEW и PROCESSING)
func (p *PosgresDB) GetWaitingOrders() ([]schema.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	query := `
	SELECT num, stat FROM (
		SELECT distinct on (os.order_id) 
			o.number num, 
			s.name stat, 
			os.accrual acc, 
			o.datetime upload
		FROM orders o JOIN order_status os ON o.order_id = os.order_id
			JOIN status s ON s.status_id = os.status_id
		ORDER BY os.order_id, os.datetime desc
		) q
	WHERE stat IN ('PROCESSING', 'NEW')
	`
	rows, err := p.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	result := make([]schema.Order, 0)
	for rows.Next() {
		order := schema.Order{}
		err := rows.Scan(&order.Number, &order.Status)
		if err != nil {
			p.logger.Error().Err(err).Msg("err is here 6516541321;")
			continue
		}
		result = append(result, order)
	}
	if err := rows.Err(); err != nil {
		p.logger.Error().Err(err).Msg("error is here 346546842419846")
	}
	if len(result) == 0 {
		return result, errorapp.ErrEmptyResult
	}

	return result, nil
}
