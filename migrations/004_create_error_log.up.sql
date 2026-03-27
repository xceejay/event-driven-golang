CREATE TABLE IF NOT EXISTS event_error_log (
  id            CHAR(36)    NOT NULL,
  event_id      CHAR(36)    NOT NULL,
  error_message TEXT        NULL,
  stack_trace   LONGTEXT    NULL,
  occurred_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  INDEX idx_error_log_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
