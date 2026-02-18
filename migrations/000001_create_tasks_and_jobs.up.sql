CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title VARCHAR(255) NOT NULL,
    description TEXT NULL,
    status ENUM('new', 'in_progress', 'done') NOT NULL DEFAULT 'new',
    priority TINYINT UNSIGNED NOT NULL DEFAULT 3,
    due_at DATETIME NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_tasks_status (status),
    INDEX idx_tasks_due_at (due_at),
    INDEX idx_tasks_created_at (created_at),
    INDEX idx_tasks_priority (priority)
);

CREATE TABLE IF NOT EXISTS jobs (
    id CHAR(36) NOT NULL,
    type VARCHAR(64) NOT NULL,
    payload JSON NOT NULL,
    status ENUM('queued', 'running', 'done', 'failed') NOT NULL DEFAULT 'queued',
    result JSON NULL,
    error TEXT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME NULL,
    finished_at DATETIME NULL,
    PRIMARY KEY (id),
    INDEX idx_jobs_status (status),
    INDEX idx_jobs_type (type),
    INDEX idx_jobs_created_at (created_at)
);
