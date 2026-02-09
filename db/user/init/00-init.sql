-- 用户ID序列（从10000001开始，8位数，缓存20）
CREATE SEQUENCE IF NOT EXISTS "uid_seq"
    START WITH 10000001
    MINVALUE 10000001
    MAXVALUE 99999999
    INCREMENT BY 1
    CACHE 20
    NO CYCLE;

-- 用户状态
CREATE TYPE "user_status" AS ENUM (
    'ACTIVE', 'INACTIVE', 'SUSPENDED', 'BANNED', 'ARCHIVED'
);

-- 用户表
CREATE TABLE IF NOT EXISTS "users" (
    "id" BIGINT PRIMARY KEY DEFAULT nextval('uid_seq'),  -- UID
    "name" VARCHAR(64) NOT NULL UNIQUE,                  -- 用户名
    "status" "user_status" NOT NULL DEFAULT 'ACTIVE',    -- 状态
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW(),       -- 创建时间
    "updated_at" TIMESTAMP NOT NULL DEFAULT NOW(),       -- 更新时间
    "last_login" TIMESTAMP NOT NULL DEFAULT NOW(),       -- 最后登录时间
    "opaque_record" BYTEA NOT NULL                       -- OPAQUE注册记录
);

-- 登录日志表
CREATE TABLE IF NOT EXISTS "auth_login_log" (
    "id"        BIGSERIAL PRIMARY KEY,  -- 记录ID
    "uid"       BIGINT REFERENCES "users"("id") ON DELETE SET NULL, -- UID
    "username"  TEXT NOT NULL,          -- 用户名
    "success"   BOOLEAN NOT NULL,       -- 状态
    "reason"    TEXT COMPRESSION zstd,  -- 失败原因
    "ip"        INET,                   -- 登录IP
    "user_agent" TEXT COMPRESSION zstd, -- UA
    "created_at" TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS "idx_users_name" ON "users"("name");
CREATE INDEX IF NOT EXISTS "idx_login_log_uid" ON "auth_login_log"("uid");

ALTER SEQUENCE "uid_seq" OWNED BY "users"."id";