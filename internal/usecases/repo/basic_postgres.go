// Package repo package repo
package repo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/chindada/capitan/internal/usecases/entity"
	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockgen -source=basic_postgres.go -destination=./mocks/mocks_basic_postgres_test.go -package=mocks

type BasicRepo interface {
	InsertStockDetail(ctx context.Context, t []*pb.StockDetail) error
	SelectStockDetailByCode(ctx context.Context, code string) (*pb.StockDetail, error)
	SelectAllStockDetail(ctx context.Context) ([]*pb.StockDetail, error)

	InsertFutureDetail(ctx context.Context, t []*pb.FutureDetail) error
	SelectFutureDetailByCode(ctx context.Context, code string) (*pb.FutureDetail, error)
	SelectAllFutureDetail(ctx context.Context) ([]*pb.FutureDetail, error)
	UpdateFutureDetailContract(ctx context.Context, req *pb.UpdateFutureDetailRequest) error

	InsertOptionDetail(ctx context.Context, t []*pb.OptionDetail) error
	SelectOptionDetailByCode(ctx context.Context, code string) (*pb.OptionDetail, error)
	SelectAllOptionDetail(ctx context.Context) ([]*pb.OptionDetail, error)

	SearchFutureDetail(ctx context.Context, code string) ([]*pb.FutureDetail, error)

	InsertFutureContract(ctx context.Context, t *pb.FutureContract) error
	SelectAllFutureContract(ctx context.Context) ([]*pb.FutureContract, error)
	SelectFutureContractByID(ctx context.Context, id int64) (*pb.FutureContract, error)
	UpdateFutureContract(ctx context.Context, t *pb.FutureContract) error
	DeleteFutureContract(ctx context.Context, id []int64) error
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
			item.GetDayTrade(),
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
	if err = row.Scan(
		&item.Code,
		&item.Name,
		&item.Exchange,
		&item.Category,
		&item.DayTrade,
		&item.Reference,
		&updateDate,
	); err != nil {
		return nil, err
	}
	item.UpdateDate = updateDate.Format(entity.ShortSlashTimeLayout)
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
		if err = rows.Scan(
			&item.Code,
			&item.Name,
			&item.Exchange,
			&item.Category,
			&item.DayTrade,
			&item.Reference,
			&updateDate,
		); err != nil {
			return nil, err
		}
		item.UpdateDate = updateDate.Format(entity.ShortSlashTimeLayout)
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
			"basic_future.code", "basic_future.symbol", "basic_future.name", "basic_future.category",
			"basic_future.delivery_month", "basic_future.delivery_date",
			"basic_future.underlying_kind", "basic_future.unit", "basic_future.limit_up", "basic_future.limit_down",
			"basic_future.reference", "basic_future.update_date",
			"COALESCE(basic_future.contract_id,0)",
			"COALESCE(basic_future_contract.name,'')", "COALESCE(basic_future_contract.price_per_tick,0)",
			"COALESCE(basic_future_contract.initial_margin,0)", "COALESCE(basic_future_contract.maintenance_margin,0)",
			"COALESCE(basic_future_contract.fee,0)", "COALESCE(basic_future_contract.tax,0)",
			"basic_future_contract.created_at", "basic_future_contract.updated_at",
		).
		From(tableNameBasicFuture).
		LeftJoin("basic_future_contract ON basic_future.contract_id = basic_future_contract.id").
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
	item := pb.FutureDetail{
		Contract: &pb.FutureContract{},
	}
	var dDate, updateData pgtype.Timestamptz
	var contractCreated, contractUpdated pgtype.Timestamptz
	if err = row.Scan(
		&item.Code, &item.Symbol, &item.Name, &item.Category, &item.DeliveryMonth,
		&dDate, &item.UnderlyingKind, &item.Unit, &item.LimitUp, &item.LimitDown, &item.Reference, &updateData,
		&item.ContractId, &item.Contract.Name, &item.Contract.PricePerTick,
		&item.Contract.InitialMargin, &item.Contract.MaintenanceMargin, &item.Contract.Fee, &item.Contract.Tax,
		&contractCreated, &contractUpdated,
	); err != nil {
		return nil, err
	}
	item.DeliveryDate = dDate.Time.Format(entity.ShortSlashTimeLayout)
	item.UpdateDate = updateData.Time.Format(entity.ShortSlashTimeLayout)
	item.Contract.CreatedAt = timestamppb.New(contractCreated.Time)
	item.Contract.UpdatedAt = timestamppb.New(contractUpdated.Time)
	return &item, tx.Commit(ctx)
}

func (r *basic) SelectAllFutureDetail(ctx context.Context) ([]*pb.FutureDetail, error) {
	builder := r.Builder().
		Select(
			"basic_future.code", "basic_future.symbol", "basic_future.name", "basic_future.category",
			"basic_future.delivery_month", "basic_future.delivery_date",
			"basic_future.underlying_kind", "basic_future.unit", "basic_future.limit_up", "basic_future.limit_down",
			"basic_future.reference", "basic_future.update_date",
			"COALESCE(basic_future.contract_id,0)",
			"COALESCE(basic_future_contract.name,'')", "COALESCE(basic_future_contract.price_per_tick,0)",
			"COALESCE(basic_future_contract.initial_margin,0)", "COALESCE(basic_future_contract.maintenance_margin,0)",
			"COALESCE(basic_future_contract.fee,0)", "COALESCE(basic_future_contract.tax,0)",
			"basic_future_contract.created_at", "basic_future_contract.updated_at",
		).
		From(tableNameBasicFuture).
		LeftJoin("basic_future_contract ON basic_future.contract_id = basic_future_contract.id").
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
		item := pb.FutureDetail{
			Contract: &pb.FutureContract{},
		}
		var dDate, updateData pgtype.Timestamptz
		var contractCreated, contractUpdated pgtype.Timestamptz
		if err = rows.Scan(
			&item.Code, &item.Symbol, &item.Name, &item.Category, &item.DeliveryMonth,
			&dDate, &item.UnderlyingKind, &item.Unit, &item.LimitUp, &item.LimitDown, &item.Reference, &updateData,
			&item.ContractId, &item.Contract.Name, &item.Contract.PricePerTick,
			&item.Contract.InitialMargin, &item.Contract.MaintenanceMargin, &item.Contract.Fee, &item.Contract.Tax,
			&contractCreated, &contractUpdated,
		); err != nil {
			return nil, err
		}
		item.DeliveryDate = dDate.Time.Format(entity.ShortSlashTimeLayout)
		item.UpdateDate = updateData.Time.Format(entity.ShortSlashTimeLayout)
		item.Contract.CreatedAt = timestamppb.New(contractCreated.Time)
		item.Contract.UpdatedAt = timestamppb.New(contractUpdated.Time)
		futures = append(futures, &item)
	}
	return futures, tx.Commit(ctx)
}

func (r *basic) UpdateFutureDetailContract(ctx context.Context, req *pb.UpdateFutureDetailRequest) error {
	var contractID *int64
	if req.GetContractId() <= 0 {
		contractID = nil
	} else {
		tmp := req.GetContractId()
		contractID = &tmp
	}
	builder := r.Builder().
		Update(tableNameBasicFuture).
		Set("contract_id", contractID).
		Where(squirrel.Eq{"code": req.GetCodes()})

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

func (r *basic) SearchFutureDetail(ctx context.Context, code string) ([]*pb.FutureDetail, error) {
	if strings.HasSuffix(code, "R1") || strings.HasSuffix(code, "R2") {
		return nil, fmt.Errorf("code %s is not allowed to search", code)
	}
	builder := r.Builder().
		Select(
			"basic_future.code", "basic_future.symbol", "basic_future.name", "basic_future.category",
			"basic_future.delivery_month", "basic_future.delivery_date",
			"basic_future.underlying_kind", "basic_future.unit", "basic_future.limit_up", "basic_future.limit_down",
			"basic_future.reference", "basic_future.update_date",
			"COALESCE(basic_future.contract_id,0)",
			"COALESCE(basic_future_contract.name,'')", "COALESCE(basic_future_contract.price_per_tick,0)",
			"COALESCE(basic_future_contract.initial_margin,0)", "COALESCE(basic_future_contract.maintenance_margin,0)",
			"COALESCE(basic_future_contract.fee,0)", "COALESCE(basic_future_contract.tax,0)",
			"basic_future_contract.created_at", "basic_future_contract.updated_at",
		).
		From(tableNameBasicFuture).
		LeftJoin("basic_future_contract ON basic_future.contract_id = basic_future_contract.id").
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
		item := pb.FutureDetail{
			Contract: &pb.FutureContract{},
		}
		var dDate, updateData pgtype.Timestamptz
		var contractCreated, contractUpdated pgtype.Timestamptz
		if err = rows.Scan(
			&item.Code, &item.Symbol, &item.Name, &item.Category, &item.DeliveryMonth,
			&dDate, &item.UnderlyingKind, &item.Unit, &item.LimitUp, &item.LimitDown, &item.Reference, &updateData,
			&item.ContractId, &item.Contract.Name, &item.Contract.PricePerTick,
			&item.Contract.InitialMargin, &item.Contract.MaintenanceMargin, &item.Contract.Fee, &item.Contract.Tax,
			&contractCreated, &contractUpdated,
		); err != nil {
			return nil, err
		}
		item.DeliveryDate = dDate.Time.Format(entity.ShortSlashTimeLayout)
		item.UpdateDate = updateData.Time.Format(entity.ShortSlashTimeLayout)
		item.Contract.CreatedAt = timestamppb.New(contractCreated.Time)
		item.Contract.UpdatedAt = timestamppb.New(contractUpdated.Time)
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

// CREATE TABLE basic_future_contract(
//     "id" serial PRIMARY KEY,
//     "name" varchar NOT NULL UNIQUE,
//     "price_per_tick" DECIMAL NOT NULL,
//     "initial_margin" int NOT NULL,
//     "maintenance_margin" int NOT NULL,
//     "fee" int NOT NULL,
//     "tax" DECIMAL NOT NULL,
//     "created_at" timestamptz NOT NULL,
//     "updated_at" timestamptz NOT NULL
// );

func (r *basic) InsertFutureContract(ctx context.Context, t *pb.FutureContract) error {
	if t == nil {
		return errors.New("future contract cannot be nil")
	}

	builder := r.Builder().
		Insert(tableNameBasicFutureContract).
		Columns(
			"name", "price_per_tick", "initial_margin", "maintenance_margin", "fee", "tax",
			"created_at", "updated_at",
		).Values(
		t.GetName(),
		t.GetPricePerTick(),
		t.GetInitialMargin(),
		t.GetMaintenanceMargin(),
		t.GetFee(),
		t.GetTax(),
		time.Now(),
		time.Now(),
	)

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

func (r *basic) SelectAllFutureContract(ctx context.Context) ([]*pb.FutureContract, error) {
	builder := r.Builder().
		Select(
			"id",
			"name", "price_per_tick", "initial_margin", "maintenance_margin", "fee", "tax",
			"created_at", "updated_at",
		).
		From(tableNameBasicFutureContract).
		OrderBy("id ASC")

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

	var contracts []*pb.FutureContract
	for rows.Next() {
		var item pb.FutureContract
		var createdAt, updatedAt pgtype.Timestamptz
		if err = rows.Scan(
			&item.Id,
			&item.Name,
			&item.PricePerTick,
			&item.InitialMargin,
			&item.MaintenanceMargin,
			&item.Fee,
			&item.Tax,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		item.CreatedAt = timestamppb.New(createdAt.Time)
		item.UpdatedAt = timestamppb.New(updatedAt.Time)
		contracts = append(contracts, &item)
	}
	return contracts, tx.Commit(ctx)
}

func (r *basic) SelectFutureContractByID(ctx context.Context, id int64) (*pb.FutureContract, error) {
	builder := r.Builder().
		Select(
			"id",
			"name", "price_per_tick", "initial_margin", "maintenance_margin", "fee", "tax",
			"created_at", "updated_at",
		).
		From(tableNameBasicFutureContract).
		Where(squirrel.Eq{"id": id})

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
	var item pb.FutureContract
	var createdAt, updatedAt pgtype.Timestamptz
	if err = row.Scan(
		&item.Id,
		&item.Name,
		&item.PricePerTick,
		&item.InitialMargin,
		&item.MaintenanceMargin,
		&item.Fee,
		&item.Tax,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	item.CreatedAt = timestamppb.New(createdAt.Time)
	item.UpdatedAt = timestamppb.New(updatedAt.Time)
	return &item, tx.Commit(ctx)
}

func (r *basic) UpdateFutureContract(ctx context.Context, t *pb.FutureContract) error {
	if t.GetId() == 0 {
		return errors.New("future contract ID is required for update")
	}

	builder := r.Builder().
		Update(tableNameBasicFutureContract).
		Set("name", t.GetName()).
		Set("price_per_tick", t.GetPricePerTick()).
		Set("initial_margin", t.GetInitialMargin()).
		Set("maintenance_margin", t.GetMaintenanceMargin()).
		Set("fee", t.GetFee()).
		Set("tax", t.GetTax()).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": t.GetId()})

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

func (r *basic) DeleteFutureContract(ctx context.Context, id []int64) error {
	if len(id) == 0 {
		return errors.New("no future contract IDs provided for deletion")
	}

	builder := r.Builder().
		Delete(tableNameBasicFutureContract).
		Where(squirrel.Eq{"id": id})

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
