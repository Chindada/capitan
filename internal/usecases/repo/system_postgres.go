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
)

//go:generate mockgen -source=system_postgres.go -destination=./mocks/mocks_system_postgres_test.go -package=mocks

type SystemRepo interface {
	SelectSetting(ctx context.Context, key pb.SettingKey) (*pb.SystemSetting, error)
	InsertSetting(ctx context.Context, s *pb.SystemSetting) error
	UpdateSetting(ctx context.Context, s *pb.SystemSetting) error
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
