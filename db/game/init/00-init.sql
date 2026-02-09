CREATE TABLE IF NOT EXISTS "player_stats" (
    "uid" BIGINT PRIMARY KEY,               -- 用户UID
    "level" INT NOT NULL DEFAULT 1,         -- 等级
    "exp" BIGINT NOT NULL DEFAULT 0,        -- 经验值
    "coins" BIGINT NOT NULL DEFAULT 0,      -- 金币数
    "kills" BIGINT NOT NULL DEFAULT 0,      -- 击杀数
    "deaths" BIGINT NOT NULL DEFAULT 0,     -- 死亡数
    "play_time" BIGINT NOT NULL DEFAULT 0,  -- 游玩时间（秒）
    "create_at" TIMESTAMP NOT NULL DEFAULT NOW(),
    "update_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS "player_rating" (
    "uid" BIGINT PRIMARY KEY REFERENCES "player_stats"("uid") ON DELETE CASCADE,
    "rating" DOUBLE PRECISION NOT NULL DEFAULT 1500.0, -- Glicko-2评分
    "rating_deviation" DOUBLE PRECISION NOT NULL DEFAULT 350.0, -- RD值
    "volatility" DOUBLE PRECISION NOT NULL DEFAULT 0.06, -- 波动率σ
    "update_at" TIMESTAMP NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_rating CHECK (rating BETWEEN 0 AND 3000),
    CONSTRAINT valid_rd CHECK (rating_deviation BETWEEN 30 AND 350),
    CONSTRAINT valid_volatility CHECK (volatility > 0)
);

CREATE TABLE IF NOT EXISTS "player_buff_slots" (
    "uid" BIGINT PRIMARY KEY REFERENCES "player_stats"("uid") ON DELETE CASCADE,
    "slots" JSONB NOT NULL DEFAULT '[]'::jsonb COMPRESSION zstd,
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW()
);