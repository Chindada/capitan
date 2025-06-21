package repo

import (
	"context"
	"errors"
	"time"

	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
	"github.com/jackc/pgx/v5"
)

//go:generate mockgen -source=user_postgres.go -destination=./mocks/mocks_user_postgres_test.go -package=mocks

type UserRepo interface {
	InsertUser(ctx context.Context, t *pb.User) error
	UpdateUser(ctx context.Context, t *pb.User) error
	UpdateUserPassword(ctx context.Context, t *pb.User) error
	SelectAllUser(ctx context.Context) (*pb.UserList, error)
	SelectUserByUsername(ctx context.Context, username string) (*pb.User, error)
	SelectUserByID(ctx context.Context, id int64) (*pb.User, error)
	SelectUserIDByUsername(ctx context.Context, username string) (int64, error)
	DeleteUser(ctx context.Context, username string) error

	ActivateUserTotp(ctx context.Context, t *pb.User, totp *pb.Totp) error
	SelectTotpByID(ctx context.Context, id int64) (*pb.Totp, error)
}

type user struct {
	client.PGClient
}

func NewUserRepo(pg client.PGClient) UserRepo {
	return &user{pg}
}

func (r *user) insertTotp(ctx context.Context, tx pgx.Tx, t *pb.Totp) (*pb.Totp, error) {
	sql, args, err := r.Builder().
		Insert(tableNameSystemTotp).
		Columns("secret, qr_code").
		Values(t.GetSecret(), t.GetQrCode()).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return nil, err
	}

	if row := tx.QueryRow(ctx, sql, args...); row == nil {
		return nil, errInsertFail
	} else if err = row.Scan(&t.Id); err != nil {
		return nil, err
	}
	return t, nil
}

func (r *user) deleteTotpByID(ctx context.Context, tx pgx.Tx, id int64) error {
	sql, args, err := r.Builder().
		Delete(tableNameSystemTotp).
		Where("id = ?", id).
		ToSql()
	if err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return nil
}

func (r *user) SelectTotpByID(ctx context.Context, id int64) (*pb.Totp, error) {
	sql, arg, err := r.Builder().
		Select("secret, qr_code").
		From(tableNameSystemTotp).
		Where("id = ?", id).
		ToSql()
	if err != nil {
		return nil, err
	}

	row := r.Pool().QueryRow(ctx, sql, arg...)
	e := pb.Totp{}
	if err = row.Scan(&e.Secret, &e.QrCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &pb.Totp{}, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *user) InsertUser(ctx context.Context, t *pb.User) error {
	sql, args, err := r.Builder().Insert(tableNameSystemAccount).
		Columns("username, password, email, role, created_at, updated_at").
		Values(
			t.GetBasic().GetUsername(), t.GetBasic().GetPassword(),
			t.GetBasic().GetEmail(), t.GetBasic().GetRole(),
			time.Now(), time.Now(),
		).
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

func (r *user) UpdateUser(ctx context.Context, t *pb.User) error {
	sql, args, err := r.Builder().
		Update(tableNameSystemAccount).
		Set("email", t.GetBasic().GetEmail()).
		Set("role", t.GetBasic().GetRole()).
		Set("updated_at", time.Now()).
		Where("username = ?", t.GetBasic().GetUsername()).
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

func (r *user) UpdateUserPassword(ctx context.Context, t *pb.User) error {
	sql, args, err := r.Builder().Update(tableNameSystemAccount).
		Set("password", t.GetBasic().GetPassword()).
		Set("updated_at", time.Now()).
		Where("username = ?", t.GetBasic().GetUsername()).
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

func (r *user) ActivateUserTotp(ctx context.Context, t *pb.User, totp *pb.Totp) error {
	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer r.Rollback(ctx, tx)

	err = r.deleteTotpByID(ctx, tx, t.GetTotpId())
	if err != nil {
		return err
	}
	result, err := r.insertTotp(ctx, tx, totp)
	if err != nil {
		return err
	}
	sql, args, err := r.Builder().Update(tableNameSystemAccount).
		Set("enable_totp", true).
		Set("totp_id", result.GetId()).
		Set("updated_at", time.Now()).
		Where("username = ?", t.GetBasic().GetUsername()).
		ToSql()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *user) SelectUserByUsername(ctx context.Context, username string) (*pb.User, error) {
	sql, arg, err := r.Builder().
		Select("id, username, password, email, role, enable_totp, COALESCE(totp_id,0)").
		From(tableNameSystemAccount).
		Where("username = ?", username).
		ToSql()
	if err != nil {
		return nil, err
	}

	row := r.Pool().QueryRow(ctx, sql, arg...)
	e := pb.User{
		Basic: &pb.BasicUser{},
	}
	if err = row.Scan(
		&e.Id,
		&e.Basic.Username, &e.Basic.Password, &e.Basic.Email, &e.Basic.Role,
		&e.EnableTotp, &e.TotpId,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &pb.User{}, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *user) SelectUserByID(ctx context.Context, id int64) (*pb.User, error) {
	sql, arg, err := r.Builder().
		Select("id, username, password, email, role, enable_totp, COALESCE(totp_id,0)").
		From(tableNameSystemAccount).
		Where("id = ?", id).
		ToSql()
	if err != nil {
		return nil, err
	}

	row := r.Pool().QueryRow(ctx, sql, arg...)
	e := pb.User{
		Basic: &pb.BasicUser{},
	}
	if err = row.Scan(
		&e.Id,
		&e.Basic.Username, &e.Basic.Password, &e.Basic.Email, &e.Basic.Role,
		&e.EnableTotp, &e.TotpId,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &pb.User{}, nil
		}
		return nil, err
	}
	return &e, nil
}

func (r *user) SelectAllUser(ctx context.Context) (*pb.UserList, error) {
	sql, arg, err := r.Builder().
		Select("id, username, email, role, enable_totp, COALESCE(totp_id,0)").
		From(tableNameSystemAccount).
		OrderBy("role DESC").
		OrderBy("username ASC").
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.Pool().Query(ctx, sql, arg...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*pb.User
	for rows.Next() {
		e := pb.User{
			Basic: &pb.BasicUser{},
		}
		if err = rows.Scan(
			&e.Id, &e.Basic.Username, &e.Basic.Email, &e.Basic.Role,
			&e.EnableTotp, &e.TotpId,
		); err != nil {
			return nil, err
		}
		result = append(result, &e)
	}
	return &pb.UserList{List: result}, nil
}

func (r *user) DeleteUser(ctx context.Context, username string) error {
	user, err := r.SelectUserByUsername(ctx, username)
	if err != nil {
		return err
	}

	sql, arg, err := r.Builder().
		Delete(tableNameSystemAccount).
		Where("username = ?", user.GetBasic().GetUsername()).
		ToSql()
	if err != nil {
		return err
	}

	tx, err := r.Pool().Begin(ctx)
	if err != nil {
		return err
	}
	defer r.Rollback(ctx, tx)

	if user.GetTotpId() != 0 {
		if err = r.deleteTotpByID(ctx, tx, user.GetTotpId()); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, sql, arg...); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *user) SelectUserIDByUsername(ctx context.Context, username string) (int64, error) {
	sql, arg, err := r.Builder().
		Select("id").
		From(tableNameSystemAccount).
		Where("username = ?", username).
		ToSql()
	if err != nil {
		return 0, err
	}

	row := r.Pool().QueryRow(ctx, sql, arg...)
	var id int64
	if err = row.Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return id, nil
}
