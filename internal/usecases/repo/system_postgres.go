package repo

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:generate mockgen -source=system_postgres.go -destination=./mocks/mocks_system_postgres_test.go -package=mocks

type SystemRepo interface {
	SelectSetting(ctx context.Context, key pb.SettingKey) (*pb.SystemSetting, error)
	InsertSetting(ctx context.Context, s *pb.SystemSetting) error
	UpdateSetting(ctx context.Context, s *pb.SystemSetting) error

	InsertLoginEvent(ctx context.Context, events []*pb.LoginEvent) error
	SelectLoginEvent(ctx context.Context, limit int64) ([]*pb.LoginEvent, error)
}

type system struct {
	client.PGClient
}

func NewSystemRepo(pg client.PGClient) SystemRepo {
	return &system{pg}
}

func (r *system) SelectSetting(ctx context.Context, key pb.SettingKey) (*pb.SystemSetting, error) {
	sql, arg, err := r.Builder().
		Select("setting").
		From(tableNameSystemSetting).
		Where(squirrel.Eq{"key": key}).
		ToSql()
	if err != nil {
		return nil, err
	}
	rows := r.Pool().QueryRow(ctx, sql, arg...)
	var content []byte
	if err = rows.Scan(&content); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &pb.SystemSetting{}, nil
		}
		return nil, err
	}
	var s pb.SystemSetting
	if err = proto.Unmarshal(content, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *system) InsertSetting(ctx context.Context, s *pb.SystemSetting) error {
	data, err := proto.Marshal(s)
	if err != nil {
		return err
	}
	sql, args, err := r.Builder().
		Insert(tableNameSystemSetting).
		Columns("key, setting, updated_at").
		Values(s.GetKey(), data, time.Now()).
		ToSql()
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

func (r *system) UpdateSetting(ctx context.Context, s *pb.SystemSetting) error {
	data, err := proto.Marshal(s)
	if err != nil {
		return err
	}
	sql, args, err := r.Builder().
		Update(tableNameSystemSetting).
		Set("setting", data).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"key": s.GetKey()}).
		ToSql()
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

func (r *system) InsertLoginEvent(ctx context.Context, events []*pb.LoginEvent) error {
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

func (r *system) SelectLoginEvent(ctx context.Context, limit int64) ([]*pb.LoginEvent, error) {
	sql, args, err := r.Builder().
		Select(`
			system_event_login.id, COALESCE(system_event_login.account_id,0), system_event_login.ip,
			system_event_login.resp_code, system_event_login.created_at,
			COALESCE(system_account.id,0), COALESCE(system_account.username,'')
			`).
		From(tableNameSystemEventLogin).
		LeftJoin("system_account ON system_event_login.account_id = system_account.id").
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
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
