package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type RoomRepository struct {
	db   *pgxpool.Pool
	imdb *redis.Client
}

type RoomInfo struct {
	ID          uint64
	Addr        string
	AvgRating   float64
	PlayerCount int
}

func NewRoomRepository(db *pgxpool.Pool, imdb *redis.Client) *RoomRepository {
	return &RoomRepository{
		db:   db,
		imdb: imdb,
	}
}

func (r *RoomRepository) SaveRoom(ctx context.Context, roomID uint64, addr string, avgRating float64, playerCnt int) error {
	data := map[string]interface{}{
		"addr":       addr,
		"avg_rating": avgRating,
		"player_cnt": playerCnt,
		"updated_at": time.Now().Unix(),
	}
	jsonData, _ := json.Marshal(data)
	return r.imdb.Set(ctx, roomKey(roomID), jsonData, 0).Err()
}

func (r *RoomRepository) GetPlayerRating(ctx context.Context, uid int64) (float64, float64, error) {
	var rating float64
	var rd float64
	sql := `SELECT rating, rating_deviation FROM player_rating WHERE uid = $1 LIMIT 1`
	err := r.db.QueryRow(ctx, sql, uid).Scan(&rating, &rd)
	return rating, rd, err
}

func roomKey(id uint64) string {
	return "room:" + fmt.Sprint(id)
}

func (r *RoomRepository) GetActiveRooms(ctx context.Context) ([]RoomInfo, error) {
	keys, err := r.imdb.Keys(ctx, "room:*").Result()
	if err != nil {
		return nil, err
	}
	rooms := make([]RoomInfo, 0, len(keys))
	for _, key := range keys {
		data, err := r.imdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var info struct {
			Addr      string  `json:"addr"`
			AvgRating float64 `json:"avg_rating"`
			PlayerCnt int     `json:"player_cnt"`
			UpdatedAt int64   `json:"updated_at"`
		}
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}
		roomID, _ := strconv.ParseUint(strings.TrimPrefix(key, "room:"), 10, 64)
		rooms = append(rooms, RoomInfo{
			ID:          roomID,
			Addr:        info.Addr,
			AvgRating:   info.AvgRating,
			PlayerCount: info.PlayerCnt,
		})
	}
	return rooms, nil
}
