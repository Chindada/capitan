package repo

import (
	"context"
	"time"

	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockgen -source=event_postgres.go -destination=./mocks/mocks_event_postgres_test.go -package=mocks

type EventRepo interface {
	InsertLoginEvent(ctx context.Context, events []*pb.LoginEvent) error
	SelectLoginEvent(ctx context.Context, limit int64) ([]*pb.LoginEvent, error)

	InsertShioajiEvent(ctx context.Context, event *pb.ShioajiEvent) error
}

type event struct {
	client.PGClient
}

func NewEventRepo(pg client.PGClient) EventRepo {
	return &event{pg}
}

func (r *event) InsertLoginEvent(ctx context.Context, events []*pb.LoginEvent) error {
	builder := r.Builder().
		Insert(tableNameSystemEventLogin).
		Columns("account_id, ip, resp_code, created_at")

	for _, event := range events {
		builder = builder.Values(
			event.GetUser().GetId(),
			event.GetIp(),
			event.GetRespCode(),
			event.GetCreatedAt().AsTime().Local(),
		)
	}

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

func (r *event) SelectLoginEvent(ctx context.Context, limit int64) ([]*pb.LoginEvent, error) {
	builder := r.Builder().
		Select(`
			system_event_login.id, COALESCE(system_event_login.account_id,0), system_event_login.ip,
			system_event_login.resp_code, system_event_login.created_at,
			COALESCE(system_account.id,0), COALESCE(system_account.username,'')
			`).
		From(tableNameSystemEventLogin).
		LeftJoin("system_account ON system_event_login.account_id = system_account.id").
		OrderBy("created_at DESC")
	if limit > 0 {
		builder = builder.Limit(uint64(limit))
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

	var result []*pb.LoginEvent
	for rows.Next() {
		event := pb.LoginEvent{}
		user := pb.User{
			Basic: &pb.BasicUser{},
		}
		var createdTime time.Time
		if err = rows.Scan(
			&event.Id, &user.Id, &event.Ip,
			&event.RespCode, &createdTime,
			&user.Id, &user.Basic.Username,
		); err != nil {
			return nil, err
		}
		if user.GetId() == 0 {
			user = pb.User{}
		}
		event.CreatedAt = timestamppb.New(createdTime)
		event.User = &user
		result = append(result, &event)
	}
	return result, nil
}

func (r *event) InsertShioajiEvent(ctx context.Context, event *pb.ShioajiEvent) error {
	builder := r.Builder().
		Insert(tableNameSystemEventShioaji).
		Columns("event_code, response, event, info, created_at").
		Values(
			event.GetEventCode(),
			event.GetRespCode(),
			event.GetEvent(),
			event.GetInfo(),
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
