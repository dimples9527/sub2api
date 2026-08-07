ALTER TABLE supplier_provider_auth_events
    DROP CONSTRAINT IF EXISTS supplier_provider_auth_events_event_type_check;

ALTER TABLE supplier_provider_auth_events
    ADD CONSTRAINT supplier_provider_auth_events_event_type_check CHECK (event_type IN (
        'cache_hit',
        'cache_miss',
        'login_success',
        'login_failed',
        'refresh_success',
        'refresh_failed',
        'cache_invalidated',
        'cache_error'
    ));