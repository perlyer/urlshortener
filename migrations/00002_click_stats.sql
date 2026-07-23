-- +goose Up
-- Обобщённая таблица агрегатов: одна строка на (ссылка, измерение, значение).
-- dimension ∈ {total, hour, device, browser, os, referer, language, country}.
CREATE TABLE click_stats (
    code      TEXT   NOT NULL,
    dimension TEXT   NOT NULL,
    value     TEXT   NOT NULL,
    count     BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (code, dimension, value)
);

-- +goose Down
DROP TABLE click_stats;
