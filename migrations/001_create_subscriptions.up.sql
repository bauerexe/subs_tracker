CREATE TABLE IF NOT EXISTS subscriptions
(
    id           BIGSERIAL PRIMARY KEY,
    user_id      UUID    NOT NULL,
    service_name VARCHAR(100)    NOT NULL,
    cost         INT NOT NULL CHECK (cost >= 0),
    start_date   DATE    NOT NULL,
    end_date     DATE,

    CHECK (end_date IS NULL OR end_date >= start_date),
    CHECK (extract(DAY FROM start_date) = 1),
    CHECK (end_date IS NULL OR extract(day from end_date) = 1)
);

CREATE INDEX IF NOT EXISTS idx_subs_user     ON subscriptions (user_id);
CREATE INDEX IF NOT EXISTS idx_subs_service  ON subscriptions (service_name);
CREATE INDEX IF NOT EXISTS idx_subs_start    ON subscriptions (start_date);
CREATE INDEX IF NOT EXISTS idx_subs_end      ON subscriptions (end_date);