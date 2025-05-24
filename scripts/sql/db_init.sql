CREATE DATABASE jotice WITH ENCODING = 'UTF8';

CREATE TYPE callback_status AS ENUM ('init', 'pending', 'succeeded', 'failed');
CREATE TABLE callback_log
(
    id              BIGINT PRIMARY KEY,
    notification_id BIGINT          NOT NULL UNIQUE,         -- 等待回调通知的 id
    retry_times     SMALLINT        NOT NULL DEFAULT 0,      -- 重试次数
    next_retry_at   BIGINT          NOT NULL DEFAULT 0,      -- 下次重试时间戳（秒）
    status          callback_status NOT NULL DEFAULT 'init', -- 回调状态
    created_at      BIGINT,
    updated_at      BIGINT
);

COMMENT
ON COLUMN callback_log.notification_id IS '等待回调通知的 id';
COMMENT
ON COLUMN callback_log.retry_times IS '重试次数';
COMMENT
ON COLUMN callback_log.next_retry_at IS '下次重试时间戳（秒）';
COMMENT
ON COLUMN callback_log.status IS '回调状态';

CREATE INDEX idx_status_create_at ON callback_log(status, created_at);
