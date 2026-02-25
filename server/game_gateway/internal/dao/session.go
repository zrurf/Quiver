package dao

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	AccessPrefix = "session:"
)

type SessionRepository struct {
	imdb *redis.Client
}

func NewSessionRepository(imdb *redis.Client) *SessionRepository {
	return &SessionRepository{
		imdb: imdb,
	}
}

// GetUidByAccessToken 通过访问令牌获取用户ID
func (r *SessionRepository) GetUidByAccessToken(ctx context.Context, token string) (int64, error) {
	return r.imdb.Get(ctx, AccessPrefix+token).Int64()
}

// HasAccessToken 检查访问令牌是否存在
func (r *SessionRepository) HasAccessToken(ctx context.Context, token string) (bool, error) {
	res, err := r.imdb.Exists(ctx, AccessPrefix+token).Result()
	return res != 0, err
}
