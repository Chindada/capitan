package repo

import (
	"context"
	"time"

	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TradeRepo interface {
	InsertOrUpdateTrade(ctx context.Context, t *pb.Trade) error
	SelectTradesByRequest(ctx context.Context, req *pb.QueryTradeRequest) ([]*pb.Trade, error)
}

type trade struct {
	client.PGClient
}

func NewTrade(pg client.PGClient) TradeRepo {
	return &trade{pg}
}

func (r *trade) getStockCode(t *pb.Trade) any {
	if t.GetType() == pb.OrderType_TYPE_STOCK_SHARE || t.GetType() == pb.OrderType_TYPE_STOCK_LOT {
		return t.GetCode()
	}
	return nil
}

func (r *trade) getFutureCode(t *pb.Trade) any {
	if t.GetType() == pb.OrderType_TYPE_FUTURE {
		return t.GetCode()
	}
	return nil
}

func (r *trade) getOptionCode(t *pb.Trade) any {
	if t.GetType() == pb.OrderType_TYPE_OPTION {
		return t.GetCode()
	}
	return nil
}

func (r *trade) InsertOrUpdateTrade(ctx context.Context, t *pb.Trade) error {
	builder := r.Builder().
		Insert(tableNameTradeRecord).
		Columns(
			"uid", "type", "order_id", "action", "price", "quantity", "filled_quantity",
			"status", "stock_code", "future_code", "option_code", "order_time",
		)

	uid := t.GetUid()
	if uid == "" {
		uid = uuid.NewString()
	}
	builder = builder.Values(
		uid,
		t.GetType(),
		t.GetOrderId(),
		t.GetAction(),
		t.GetPrice(),
		t.GetQuantity(),
		t.GetFilledQuantity(),
		t.GetStatus(),
		r.getStockCode(t),
		r.getFutureCode(t),
		r.getOptionCode(t),
		t.GetOrderTime().AsTime().Local(),
	)
	builder = builder.Suffix(`ON CONFLICT (order_id) DO UPDATE SET
			type = EXCLUDED.type,
			action = EXCLUDED.action,
			price = EXCLUDED.price,
			quantity = EXCLUDED.quantity,
			filled_quantity = EXCLUDED.filled_quantity,
			status = EXCLUDED.status,
			stock_code = EXCLUDED.stock_code,
			future_code = EXCLUDED.future_code,
			option_code = EXCLUDED.option_code,
			order_time = EXCLUDED.order_time
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

type scanStock struct {
	code       pgtype.Text
	name       pgtype.Text
	exchange   pgtype.Text
	category   pgtype.Text
	reference  pgtype.Numeric
	updateDate pgtype.Timestamptz
}

func (s *scanStock) ToPB() *pb.StockDetail {
	if !s.code.Valid {
		return nil
	}
	ref, _ := s.reference.Float64Value()
	return &pb.StockDetail{
		Code:       s.code.String,
		Name:       s.name.String,
		Exchange:   s.exchange.String,
		Category:   s.category.String,
		Reference:  ref.Float64,
		UpdateDate: s.updateDate.Time.Format(time.RFC3339),
	}
}

type scanFuture struct {
	code           pgtype.Text
	symbol         pgtype.Text
	name           pgtype.Text
	category       pgtype.Text
	deliveryMonth  pgtype.Text
	deliveryDate   pgtype.Timestamptz
	underlyingKind pgtype.Text
	unit           pgtype.Numeric
	limitUp        pgtype.Numeric
	limitDown      pgtype.Numeric
	reference      pgtype.Numeric
	updateDate     pgtype.Timestamptz
}

func (f *scanFuture) ToPB() *pb.FutureDetail {
	if !f.code.Valid {
		return nil
	}
	limitUp, _ := f.limitUp.Float64Value()
	limitDown, _ := f.limitDown.Float64Value()
	unit, _ := f.unit.Int64Value()
	return &pb.FutureDetail{
		Code:           f.code.String,
		Symbol:         f.symbol.String,
		Name:           f.name.String,
		Category:       f.category.String,
		DeliveryMonth:  f.deliveryMonth.String,
		DeliveryDate:   f.deliveryDate.Time.Format(time.RFC3339),
		UnderlyingKind: f.underlyingKind.String,
		Unit:           unit.Int64,
		LimitUp:        limitUp.Float64,
		LimitDown:      limitDown.Float64,
		UpdateDate:     f.updateDate.Time.Format(time.RFC3339),
	}
}

type scanOption struct {
	code           pgtype.Text
	symbol         pgtype.Text
	name           pgtype.Text
	category       pgtype.Text
	deliveryMonth  pgtype.Text
	deliveryDate   pgtype.Timestamptz
	strikePrice    pgtype.Numeric
	optionRight    pgtype.Text
	underlyingKind pgtype.Text
	unit           pgtype.Numeric
	limitUp        pgtype.Numeric
	limitDown      pgtype.Numeric
	reference      pgtype.Numeric
	updateDate     pgtype.Timestamptz
}

func (o *scanOption) ToPB() *pb.OptionDetail {
	if !o.code.Valid {
		return nil
	}
	strikePrice, _ := o.strikePrice.Float64Value()
	limitUp, _ := o.limitUp.Float64Value()
	limitDown, _ := o.limitDown.Float64Value()
	unit, _ := o.unit.Int64Value()
	return &pb.OptionDetail{
		Code:           o.code.String,
		Symbol:         o.symbol.String,
		Name:           o.name.String,
		Category:       o.category.String,
		DeliveryMonth:  o.deliveryMonth.String,
		DeliveryDate:   o.deliveryDate.Time.Format(time.RFC3339),
		StrikePrice:    strikePrice.Float64,
		OptionRight:    o.optionRight.String,
		UnderlyingKind: o.underlyingKind.String,
		Unit:           unit.Int64,
		LimitUp:        limitUp.Float64,
		LimitDown:      limitDown.Float64,
		UpdateDate:     o.updateDate.Time.Format(time.RFC3339),
	}
}

func (r *trade) SelectTradesByRequest(ctx context.Context, req *pb.QueryTradeRequest) ([]*pb.Trade, error) {
	builder := r.Builder().
		Select(
			"trade_record.uid", "trade_record.type", "trade_record.order_id", "trade_record.action",
			"trade_record.price", "trade_record.quantity", "trade_record.filled_quantity",
			"trade_record.status", "trade_record.order_time",
			"trade_record.stock_code", "basic_stock.name", "basic_stock.exchange", "basic_stock.category", "basic_stock.last_close", "basic_stock.update_date",
			"trade_record.future_code", "basic_future.symbol", "basic_future.name", "basic_future.category", "basic_future.delivery_month",
			"basic_future.delivery_date", "basic_future.underlying_kind", "basic_future.unit", "basic_future.limit_up", "basic_future.limit_down", "basic_future.reference", "basic_future.update_date",
			"trade_record.option_code", "basic_option.symbol", "basic_option.name", "basic_option.category", "basic_option.delivery_month",
			"basic_option.delivery_date", "basic_option.strike_price", "basic_option.option_right", "basic_option.underlying_kind",
			"basic_option.unit", "basic_option.limit_up", "basic_option.limit_down", "basic_option.reference", "basic_option.update_date",
		).
		From(tableNameTradeRecord).
		LeftJoin("basic_stock ON trade_record.stock_code = basic_stock.code").
		LeftJoin("basic_future ON trade_record.future_code = basic_future.code").
		LeftJoin("basic_option ON trade_record.option_code = basic_option.code").
		OrderBy("trade_record.order_time DESC")

	if req.GetOrderId() != "" {
		builder = builder.Where("trade_record.order_id = ?", req.GetOrderId())
	}
	if req.GetStartTime().IsValid() && req.GetEndTime().IsValid() {
		builder = builder.Where("trade_record.order_time >= ? AND trade_record.order_time <= ?", req.GetStartTime().AsTime().Local(), req.GetEndTime().AsTime().Local())
	}
	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.Pool().Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trades []*pb.Trade
	stock := scanStock{}
	future := scanFuture{}
	option := scanOption{}
	var orderTime pgtype.Timestamptz
	for rows.Next() {
		t := &pb.Trade{}
		if err = rows.Scan(
			&t.Uid, &t.Type, &t.OrderId, &t.Action,
			&t.Price, &t.Quantity, &t.FilledQuantity,
			&t.Status, &orderTime,
			&stock.code, &stock.name, &stock.exchange, &stock.category, &stock.reference, &stock.updateDate,
			&future.code, &future.symbol, &future.name, &future.category, &future.deliveryMonth,
			&future.deliveryDate, &future.underlyingKind, &future.unit, &future.limitUp, &future.limitDown, &future.reference, &future.updateDate,
			&option.code, &option.symbol, &option.name, &option.category, &option.deliveryMonth,
			&option.deliveryDate, &option.strikePrice, &option.optionRight, &option.underlyingKind,
			&option.unit, &option.limitUp, &option.limitDown, &option.reference, &option.updateDate,
		); err != nil {
			return nil, err
		}
		trades = append(trades, t)
		t.OrderTime = timestamppb.New(orderTime.Time.Local())
		t.Stock = stock.ToPB()
		t.Future = future.ToPB()
		t.Option = option.ToPB()
		switch {
		case stock.code.Valid:
			t.Code = stock.code.String
		case future.code.Valid:
			t.Code = future.code.String
		case option.code.Valid:
			t.Code = option.code.String
		}
	}
	return trades, nil
}
