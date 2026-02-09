package dao

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	AccessPrefix  = "session:"
	RefreshPrefix = "refresh:"
)

type SessionRepository struct {
	imdb *redis.Client
}

func NewSessionRepository(imdb *redis.Client) *SessionRepository {
	return &SessionRepository{
		imdb: imdb,
	}
}

// SaveAccessToken 保存访问令牌
func (r *SessionRepository) SaveAccessToken(ctx context.Context, token string, uid int64, expireSec int) error {
	key := AccessPrefix + token
	return r.imdb.Set(ctx, key, uid, time.Second*time.Duration(expireSec)).Err()
}

// SaveRefreshToken 保存刷新令牌
func (r *SessionRepository) SaveRefreshToken(ctx context.Context, token string, uid int64, expireSec int) error {
	key := RefreshPrefix + token
	return r.imdb.Set(ctx, key, uid, time.Second*time.Duration(expireSec)).Err()
}

// GetUidByAccessToken 通过访问令牌获取用户ID
func (r *SessionRepository) GetUidByAccessToken(ctx context.Context, token string) (int64, error) {
	return r.imdb.Get(ctx, AccessPrefix+token).Int64()
}

// GetUidByRefreshToken 通过刷新令牌获取用户ID
func (r *SessionRepository) GetUidByRefreshToken(ctx context.Context, token string) (int64, error) {
	return r.imdb.Get(ctx, RefreshPrefix+token).Int64()
}

// DelAccessToken 删除访问令牌
func (r *SessionRepository) DelAccessToken(ctx context.Context, token string) error {
	return r.imdb.Del(ctx, AccessPrefix+token).Err()
}

// DelRefreshToken 删除刷新令牌
func (r *SessionRepository) DelRefreshToken(ctx context.Context, token string) error {
	return r.imdb.Del(ctx, RefreshPrefix+token).Err()
}

// HasAccessToken 检查访问令牌是否存在
func (r *SessionRepository) HasAccessToken(ctx context.Context, token string) (bool, error) {
	res, err := r.imdb.Exists(ctx, AccessPrefix+token).Result()
	return res != 0, err
}

// HasRefreshToken 检查刷新令牌是否存在
func (r *SessionRepository) HasRefreshToken(ctx context.Context, token string) (bool, error) {
	res, err := r.imdb.Exists(ctx, RefreshPrefix+token).Result()
	return res != 0, err
}
