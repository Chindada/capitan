// Package repo package repo
package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/chindada/capitan/internal/usecases/entity"
	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
)

//go:generate mockgen -source=basic_postgres.go -destination=./mocks/mocks_basic_postgres_test.go -package=mocks

type BasicRepo interface {
	InsertStockDetail(ctx context.Context, t []*pb.StockDetail) error
	SelectStockDetailByCode(ctx context.Context, code string) (*pb.StockDetail, error)
	SelectAllStockDetail(ctx context.Context) ([]*pb.StockDetail, error)

	InsertFutureDetail(ctx context.Context, t []*pb.FutureDetail) error
	SelectFutureDetailByCode(ctx context.Context, code string) (*pb.FutureDetail, error)
	SelectAllFutureDetail(ctx context.Context) ([]*pb.FutureDetail, error)

	InsertOptionDetail(ctx context.Context, t []*pb.OptionDetail) error
	SelectOptionDetailByCode(ctx context.Context, code string) (*pb.OptionDetail, error)
	SelectAllOptionDetail(ctx context.Context) ([]*pb.OptionDetail, error)

	SearchFutureDetail(ctx context.Context, code string) ([]*pb.FutureDetail, error)
}

type basic struct {
	client.PGClient
}

func NewBasic(pg client.PGClient) BasicRepo {
	return &basic{pg}
}

// CREATE TABLE basic_stock(
//     "code" varchar PRIMARY KEY,
//     "name" varchar NOT NULL,
//     "exchange" varchar NOT NULL,
//     "category" varchar NOT NULL,
//     "day_trade" boolean NOT NULL,
//     "last_close" DECIMAL NOT NULL,
//     "update_date" timestamptz NOT NULL
// );

func (r *basic) InsertStockDetail(ctx context.Context, t []*pb.StockDetail) error {
	builder := r.Builder().
		Insert(tableNameBasicStock).
		Columns(
			"code", "name", "exchange", "category", "day_trade", "last_close", "update_date",
		)

	for _, item := range t {
		updateTime, err := time.ParseInLocation(entity.ShortSlashTimeLayout, item.GetUpdateDate(), time.Local)
		if err != nil {
			return err
		}
		builder = builder.Values(
			item.GetCode(),
			item.GetName(),
			item.GetExchange(),
			item.GetCategory(),
			item.GetDayTrade() == entity.DayTradeYes,
			item.GetReference(),
			updateTime,
		)
	}
	builder = builder.Suffix(`ON CONFLICT (code) DO UPDATE SET
            name = EXCLUDED.name,
            exchange = EXCLUDED.exchange,
            category = EXCLUDED.category,
            day_trade = EXCLUDED.day_trade,
            last_close = EXCLUDED.last_close,
            update_date = EXCLUDED.update_date
        `)

	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer r.Rollback(ctx, tx)

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *basic) SelectStockDetailByCode(ctx context.Context, code string) (*pb.StockDetail, error) {
	builder := r.Builder().
		Select(
			"code", "name", "exchange", "category", "day_trade", "last_close", "update_date",
		).
		From(tableNameBasicStock).
		Where(squirrel.Eq{"code": code})

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	row := tx.QueryRow(ctx, sql, args...)
	var item pb.StockDetail
	var updateDate time.Time
	var dayTrade bool
	if err = row.Scan(
		&item.Code,
		&item.Name,
		&item.Exchange,
		&item.Category,
		&dayTrade,
		&item.Reference,
		&updateDate,
	); err != nil {
		return nil, err
	}
	item.UpdateDate = updateDate.Format(entity.ShortSlashTimeLayout)
	if dayTrade {
		item.DayTrade = entity.DayTradeYes
	} else {
		item.DayTrade = entity.DayTradeNo
	}
	return &item, tx.Commit(ctx)
}

func (r *basic) SelectAllStockDetail(ctx context.Context) ([]*pb.StockDetail, error) {
	builder := r.Builder().
		Select(
			"code", "name", "exchange", "category", "day_trade", "last_close", "update_date",
		).
		From(tableNameBasicStock).
		OrderBy("code ASC")

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stocks []*pb.StockDetail
	for rows.Next() {
		var item pb.StockDetail
		var updateDate time.Time
		var dayTrade bool
		if err = rows.Scan(
			&item.Code,
			&item.Name,
			&item.Exchange,
			&item.Category,
			&dayTrade,
			&item.Reference,
			&updateDate,
		); err != nil {
			return nil, err
		}
		item.UpdateDate = updateDate.Format(entity.ShortSlashTimeLayout)
		if dayTrade {
			item.DayTrade = entity.DayTradeYes
		} else {
			item.DayTrade = entity.DayTradeNo
		}
		stocks = append(stocks, &item)
	}
	return stocks, tx.Commit(ctx)
}

// CREATE TABLE basic_future(
//     "code" varchar PRIMARY KEY,
//     "symbol" varchar NOT NULL,
//     "name" varchar NOT NULL,
//     "category" varchar NOT NULL,
//     "delivery_month" varchar NOT NULL,
//     "delivery_date" timestamptz NOT NULL,
//     "underlying_kind" varchar NOT NULL,
//     "unit" int NOT NULL,
//     "limit_up" DECIMAL NOT NULL,
//     "limit_down" DECIMAL NOT NULL,
//     "reference" DECIMAL NOT NULL,
//     "update_date" timestamptz NOT NULL
// );

func (r *basic) InsertFutureDetail(ctx context.Context, t []*pb.FutureDetail) error {
	builder := r.Builder().
		Insert(tableNameBasicFuture).
		Columns(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		)

	for _, item := range t {
		updateTime, err := time.ParseInLocation(entity.ShortSlashTimeLayout, item.GetUpdateDate(), time.Local)
		if err != nil {
			return err
		}
		dDate, e := time.ParseInLocation(entity.ShortSlashTimeLayout, item.GetDeliveryDate(), time.Local)
		if e != nil {
			continue
		}
		builder = builder.Values(
			item.GetCode(),
			item.GetSymbol(),
			item.GetName(),
			item.GetCategory(),
			item.GetDeliveryMonth(),
			dDate.Add(810*time.Minute),
			item.GetUnderlyingKind(),
			item.GetUnit(),
			item.GetLimitUp(),
			item.GetLimitDown(),
			item.GetReference(),
			updateTime,
		)
	}
	builder = builder.Suffix(`ON CONFLICT (code) DO UPDATE SET
			symbol = EXCLUDED.symbol,
            name = EXCLUDED.name,
			category = EXCLUDED.category,
			delivery_month = EXCLUDED.delivery_month,
			delivery_date = EXCLUDED.delivery_date,
			underlying_kind = EXCLUDED.underlying_kind,
			unit = EXCLUDED.unit,
			limit_up = EXCLUDED.limit_up,
			limit_down = EXCLUDED.limit_down,
			reference = EXCLUDED.reference,
			update_date = EXCLUDED.update_date
        `)

	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer r.Rollback(ctx, tx)

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *basic) SelectFutureDetailByCode(ctx context.Context, code string) (*pb.FutureDetail, error) {
	builder := r.Builder().
		Select(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		).
		From(tableNameBasicFuture).
		Where(squirrel.Eq{"code": code})

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	row := tx.QueryRow(ctx, sql, args...)
	var item pb.FutureDetail
	var dDate, updateData time.Time
	if err = row.Scan(
		&item.Code,
		&item.Symbol,
		&item.Name,
		&item.Category,
		&item.DeliveryMonth,
		&dDate,
		&item.UnderlyingKind,
		&item.Unit,
		&item.LimitUp,
		&item.LimitDown,
		&item.Reference,
		&updateData,
	); err != nil {
		return nil, err
	}
	item.DeliveryDate = dDate.Format(entity.ShortSlashTimeLayout)
	item.UpdateDate = updateData.Format(entity.ShortSlashTimeLayout)
	return &item, tx.Commit(ctx)
}

func (r *basic) SelectAllFutureDetail(ctx context.Context) ([]*pb.FutureDetail, error) {
	builder := r.Builder().
		Select(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		).
		From(tableNameBasicFuture).
		OrderBy("code ASC")

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var futures []*pb.FutureDetail
	for rows.Next() {
		var item pb.FutureDetail
		var dDate, updateData time.Time
		if err = rows.Scan(
			&item.Code,
			&item.Symbol,
			&item.Name,
			&item.Category,
			&item.DeliveryMonth,
			&dDate,
			&item.UnderlyingKind,
			&item.Unit,
			&item.LimitUp,
			&item.LimitDown,
			&item.Reference,
			&updateData,
		); err != nil {
			return nil, err
		}
		item.DeliveryDate = dDate.Format(entity.ShortSlashTimeLayout)
		item.UpdateDate = updateData.Format(entity.ShortSlashTimeLayout)
		futures = append(futures, &item)
	}
	return futures, tx.Commit(ctx)
}

func (r *basic) SearchFutureDetail(ctx context.Context, code string) ([]*pb.FutureDetail, error) {
	if strings.HasSuffix(code, "R1") || strings.HasSuffix(code, "R2") {
		return nil, fmt.Errorf("code %s is not allowed to search", code)
	}
	builder := r.Builder().
		Select(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		).
		From(tableNameBasicFuture).
		Where(squirrel.Like{"code": fmt.Sprintf("%s%%", code)}).
		Where(squirrel.NotEq{"code": fmt.Sprintf("%sR1", code)}).
		Where(squirrel.NotEq{"code": fmt.Sprintf("%sR2", code)}).
		OrderBy("delivery_date ASC")

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var futures []*pb.FutureDetail
	for rows.Next() {
		var item pb.FutureDetail
		var dDate, updateData time.Time
		if err = rows.Scan(
			&item.Code,
			&item.Symbol,
			&item.Name,
			&item.Category,
			&item.DeliveryMonth,
			&dDate,
			&item.UnderlyingKind,
			&item.Unit,
			&item.LimitUp,
			&item.LimitDown,
			&item.Reference,
			&updateData,
		); err != nil {
			return nil, err
		}
		item.DeliveryDate = dDate.Format(entity.ShortSlashTimeLayout)
		item.UpdateDate = updateData.Format(entity.ShortSlashTimeLayout)
		futures = append(futures, &item)
	}
	return futures, tx.Commit(ctx)
}

// CREATE TABLE basic_option(
//     "code" varchar PRIMARY KEY,
//     "symbol" varchar NOT NULL,
//     "name" varchar NOT NULL,
//     "category" varchar NOT NULL,
//     "delivery_month" varchar NOT NULL,
//     "delivery_date" timestamptz NOT NULL,
//     "strike_price" DECIMAL NOT NULL,
//     "option_right" varchar NOT NULL,
//     "underlying_kind" varchar NOT NULL,
//     "unit" int NOT NULL,
//     "limit_up" DECIMAL NOT NULL,
//     "limit_down" DECIMAL NOT NULL,
//     "reference" DECIMAL NOT NULL,
//     "update_date" timestamptz NOT NULL
// );

func (r *basic) InsertOptionDetail(ctx context.Context, t []*pb.OptionDetail) error {
	builder := r.Builder().
		Insert(tableNameBasicOption).
		Columns(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"strike_price", "option_right",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		)

	for _, item := range t {
		updateTime, err := time.ParseInLocation(entity.ShortSlashTimeLayout, item.GetUpdateDate(), time.Local)
		if err != nil {
			return err
		}
		dDate, e := time.ParseInLocation(entity.ShortSlashTimeLayout, item.GetDeliveryDate(), time.Local)
		if e != nil {
			continue
		}
		builder = builder.Values(
			item.GetCode(),
			item.GetSymbol(),
			item.GetName(),
			item.GetCategory(),
			item.GetDeliveryMonth(),
			dDate.Add(810*time.Minute),
			item.GetStrikePrice(),
			item.GetOptionRight(),
			item.GetUnderlyingKind(),
			item.GetUnit(),
			item.GetLimitUp(),
			item.GetLimitDown(),
			item.GetReference(),
			updateTime,
		)
	}
	builder = builder.Suffix(`ON CONFLICT (code) DO UPDATE SET
			symbol = EXCLUDED.symbol,
            name = EXCLUDED.name,
			category = EXCLUDED.category,
			delivery_month = EXCLUDED.delivery_month,
			delivery_date = EXCLUDED.delivery_date,
			strike_price = EXCLUDED.strike_price,
			option_right = EXCLUDED.option_right,
			underlying_kind = EXCLUDED.underlying_kind,
			unit = EXCLUDED.unit,
			limit_up = EXCLUDED.limit_up,
			limit_down = EXCLUDED.limit_down,
			reference = EXCLUDED.reference,
			update_date = EXCLUDED.update_date
        `)

	sql, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer r.Rollback(ctx, tx)

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *basic) SelectOptionDetailByCode(ctx context.Context, code string) (*pb.OptionDetail, error) {
	builder := r.Builder().
		Select(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"strike_price", "option_right",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		).
		From(tableNameBasicOption).
		Where(squirrel.Eq{"code": code})

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	row := tx.QueryRow(ctx, sql, args...)
	var item pb.OptionDetail
	var dDate, updateData time.Time
	if err = row.Scan(
		&item.Code,
		&item.Symbol,
		&item.Name,
		&item.Category,
		&item.DeliveryMonth,
		&dDate,
		&item.StrikePrice,
		&item.OptionRight,
		&item.UnderlyingKind,
		&item.Unit,
		&item.LimitUp,
		&item.LimitDown,
		&item.Reference,
		&updateData,
	); err != nil {
		return nil, err
	}
	item.DeliveryDate = dDate.Format(entity.ShortSlashTimeLayout)
	item.UpdateDate = updateData.Format(entity.ShortSlashTimeLayout)
	return &item, tx.Commit(ctx)
}

func (r *basic) SelectAllOptionDetail(ctx context.Context) ([]*pb.OptionDetail, error) {
	builder := r.Builder().
		Select(
			"code", "symbol", "name", "category", "delivery_month", "delivery_date",
			"strike_price", "option_right",
			"underlying_kind", "unit", "limit_up", "limit_down", "reference", "update_date",
		).
		From(tableNameBasicOption).
		OrderBy("code ASC")

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Rollback(ctx, tx)

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []*pb.OptionDetail
	for rows.Next() {
		var item pb.OptionDetail
		var dDate, updateData time.Time
		if err = rows.Scan(
			&item.Code,
			&item.Symbol,
			&item.Name,
			&item.Category,
			&item.DeliveryMonth,
			&dDate,
			&item.StrikePrice,
			&item.OptionRight,
			&item.UnderlyingKind,
			&item.Unit,
			&item.LimitUp,
			&item.LimitDown,
			&item.Reference,
			&updateData,
		); err != nil {
			return nil, err
		}
		item.DeliveryDate = dDate.Format(entity.ShortSlashTimeLayout)
		item.UpdateDate = updateData.Format(entity.ShortSlashTimeLayout)
		options = append(options, &item)
	}
	return options, tx.Commit(ctx)
}
