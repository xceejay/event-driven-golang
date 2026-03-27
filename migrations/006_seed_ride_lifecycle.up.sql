INSERT IGNORE INTO event_lifecycle_config (event_type, flow_type, max_attempts, event_lifespan_seconds, is_suspended, attempt_lifecycle_configs)
VALUES
  -- Ride requested: find a driver
  (
    'RIDE_REQUESTED',
    'FLOW_B',
    3,
    600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 5),
      JSON_OBJECT('attempt_number', 3, 'delay_seconds', 10)
    )
  ),
  -- Driver matched
  (
    'DRIVER_MATCHED',
    'FLOW_B',
    3,
    600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 5),
      JSON_OBJECT('attempt_number', 3, 'delay_seconds', 10)
    )
  ),
  -- Driver arrived
  (
    'DRIVER_ARRIVED',
    'FLOW_B',
    2,
    300,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 5)
    )
  ),
  -- Trip started
  (
    'TRIP_STARTED',
    'FLOW_B',
    1,
    3600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0)
    )
  ),
  -- Trip completed
  (
    'TRIP_COMPLETED',
    'FLOW_B',
    1,
    600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0)
    )
  ),
  -- Payment processed
  (
    'PAYMENT_PROCESSED',
    'FLOW_B',
    3,
    600,
    FALSE,
    JSON_ARRAY(
      JSON_OBJECT('attempt_number', 1, 'delay_seconds', 0),
      JSON_OBJECT('attempt_number', 2, 'delay_seconds', 10),
      JSON_OBJECT('attempt_number', 3, 'delay_seconds', 20)
    )
  );
