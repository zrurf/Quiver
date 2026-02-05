package dao

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: pool,
	}
}

// 保存Opaque Record
func (r *UserRepository) SaveUserRecord(ctx context.Context, username string, record []byte) error {
	log.Debug().Str("uname", username).Any("record", record).Msg("save user record")
	sql := `INSERT INTO users (name, opaque_record) VALUES ($1,$2) ON CONFLICT (name) DO NOTHING`
	_, err := r.db.Exec(ctx, sql, username, record)
	return err
}

// 获取Opaque Record
func (r *UserRepository) GetUserRecord(ctx context.Context, username string) (int64, []byte, error) {
	var id int64
	var record []byte
	sql := `SELECT id, opaque_record FROM users WHERE name = $1 LIMIT 1`
	err := r.db.QueryRow(ctx, sql, username).Scan(&id, &record)
	log.Debug().Str("uname", username).Any("record", record).Msg("read user record")
	return id, record, err
}

// 更新用户最后登录时间
func (r *UserRepository) UpdateLastLogin(ctx context.Context, uid int64) error {
	sql := `UPDATE users SET last_login = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, sql, uid)
	return err
}

// 检查用户名是否存在
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	sql := `SELECT EXISTS(SELECT 1 FROM users WHERE name = $1 LIMIT 1)`
	err := r.db.QueryRow(ctx, sql, username).Scan(&exists)
	return exists, err
}
