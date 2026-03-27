CREATE TABLE IF NOT EXISTS event_lifecycle_config (
  id                        BIGINT       NOT NULL AUTO_INCREMENT,
  event_type                VARCHAR(255) NOT NULL UNIQUE,
  flow_type                 VARCHAR(100) NOT NULL,
  max_attempts              INT          NOT NULL,
  event_lifespan_seconds    BIGINT       NOT NULL,
  is_suspended              BOOLEAN      NOT NULL DEFAULT FALSE,
  attempt_lifecycle_configs JSON         NOT NULL,
  created_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
