package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/zrurf/quiver/server/game/internal/model"
)

type PlayerDAO struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewPlayerDAO(db *pgxpool.Pool, rdb *redis.Client) *PlayerDAO {
	return &PlayerDAO{db: db, rdb: rdb}
}

// Load 从数据库加载玩家数据，若不存在则创建默认记录
func (d *PlayerDAO) Load(ctx context.Context, uid int64) (*model.Player, error) {
	var level int
	var exp, coins, kills, deaths, playTime int64
	err := d.db.QueryRow(ctx, `
        SELECT level, exp, coins, kills, deaths, play_time
        FROM player_stats WHERE uid = $1
    `, uid).Scan(&level, &exp, &coins, &kills, &deaths, &playTime)
	if err != nil {
		if err == pgx.ErrNoRows {
			// 插入默认记录
			_, err = d.db.Exec(ctx, `
                INSERT INTO player_stats (uid, level, exp, coins, kills, deaths, play_time)
                VALUES ($1, 1, 0, 0, 0, 0, 0)
            `, uid)
			if err != nil {
				return nil, err
			}
			_, err = d.db.Exec(ctx, `
                INSERT INTO player_rating (uid, rating, rating_deviation, volatility)
                VALUES ($1, 1500.0, 350.0, 0.06)
            `, uid)
			if err != nil {
				return nil, err
			}
			level = 1
		} else {
			return nil, err
		}
	}

	// 加载 rating
	var rating, rd, vol float64
	err = d.db.QueryRow(ctx, `
        SELECT rating, rating_deviation, volatility FROM player_rating WHERE uid = $1
    `, uid).Scan(&rating, &rd, &vol)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	// 加载 buffs
	var buffs []uint8
	data, err := d.rdb.Get(ctx, "buff:"+fmt.Sprint(uid)).Bytes()
	if err == nil {
		json.Unmarshal(data, &buffs)
	}

	p := model.NewPlayer(uid)
	p.Level = level
	p.Exp = exp
	p.Coins = coins
	p.Kills = kills
	p.Deaths = deaths
	p.PlayTime = playTime
	p.Rating = rating
	p.RatingDeviation = rd
	p.Volatility = vol
	p.SetBuffs(buffs)
	return p, nil
}

// Save 保存玩家数据
func (d *PlayerDAO) Save(ctx context.Context, p *model.Player) error {
	// 更新 player_stats
	_, err := d.db.Exec(ctx, `
		UPDATE player_stats
		SET level = $2, exp = $3, coins = $4, kills = $5, deaths = $6, play_time = $7, update_at = NOW()
		WHERE uid = $1
	`, p.UID, p.Level, p.Exp, p.Coins, p.Kills, p.Deaths, p.PlayTime)
	if err != nil {
		return err
	}

	// 保存 buffs 到 IMDB
	buffs := p.GetBuffs()
	if len(buffs) > 0 {
		data, _ := json.Marshal(buffs)
		return d.rdb.Set(ctx, "buff:"+fmt.Sprint(p.UID), data, 0).Err()
	}
	return nil
}

// UpdateRoomRating 更新房间排名信息
func (d *PlayerDAO) UpdateRoomRating(ctx context.Context, roomID uint64, avgRating float64, playerCnt int) error {
	data := map[string]interface{}{
		"addr":       "", // 地址不需要更新
		"avg_rating": avgRating,
		"player_cnt": playerCnt,
		"updated_at": time.Now().Unix(),
	}
	jsonData, _ := json.Marshal(data)
	return d.rdb.Set(ctx, "room:"+strconv.FormatUint(roomID, 10), jsonData, 0).Err()
}
