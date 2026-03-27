INSERT INTO event_lifecycle_config (event_type, flow_type, max_attempts, event_lifespan_seconds, is_suspended, attempt_lifecycle_configs)
VALUES
  (
    'FLOW_A_STEP_1_REQUESTED',
    'FLOW_A',
    3,
    300,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 30),
      JSON_OBJECT('attempt_number', 3, 'delay_seconds', 60)
    )
  ),
  (
    'FLOW_A_STEP_1_EXPIRED',
    'FLOW_A',
    1,
    60,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0)
    )
  ),
  (
    'FLOW_A_STEP_2_REQUESTED',
    'FLOW_A',
    5,
    3600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 60),
      JSON_OBJECT('attempt_number', 3, 'delay_seconds', 120),
      JSON_OBJECT('attempt_number', 4, 'delay_seconds', 300),
      JSON_OBJECT('attempt_number', 5, 'delay_seconds', 600)
    )
  ),
  (
    'FLOW_B_NOTIFICATION_REQUESTED',
    'FLOW_B',
    3,
    600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 60),
      JSON_OBJECT('attempt_number', 3, 'delay_seconds', 120)
    )
  ),
  (
    'FLOW_B_NOTIFICATION_FAILED',
    'FLOW_B',
    1,
    120,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0)
    )
  );
