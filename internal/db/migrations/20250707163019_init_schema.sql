-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
                                     id BIGINT AUTO_INCREMENT PRIMARY KEY,
                                     telegram_id BIGINT UNIQUE,
                                     username VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    credits INT NOT NULL DEFAULT 0,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    uts BIGINT
    );

CREATE TABLE IF NOT EXISTS billing_transactions (
                                                    id INT AUTO_INCREMENT PRIMARY KEY,
                                                    user_id BIGINT,
                                                    credits_added INT,
                                                    amount_paid INT,
                                                    provider VARCHAR(50),
    payload VARCHAR(255),
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS user_logs (
                                         id INT AUTO_INCREMENT PRIMARY KEY,
                                         user_id BIGINT,
                                         action_type VARCHAR(50),
    prompt TEXT,
    success BOOLEAN,
    video_path TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );

CREATE TABLE IF NOT EXISTS pending_videos (
                                              id INT AUTO_INCREMENT PRIMARY KEY,
                                              user_id BIGINT,
                                              prompt TEXT,
                                              video_path TEXT,
                                              created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS generation_errors (
                                                 id INT AUTO_INCREMENT PRIMARY KEY,
                                                 user_id BIGINT NOT NULL,
                                                 prompt TEXT NOT NULL,
                                                 error_code INT,
                                                 support_code VARCHAR(64),
    error_message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS generation_errors;
DROP TABLE IF EXISTS pending_videos;
DROP TABLE IF EXISTS user_logs;
DROP TABLE IF EXISTS billing_transactions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
