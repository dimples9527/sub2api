UPDATE supplier_provider_types
SET recharge_url = '/api/v1/redeem/history?timezone=Asia%2FShanghai'
WHERE code = 'sub2api'
  AND recharge_url = '/v1/redeem/history?timezone=Asia%2FShanghai';

UPDATE supplier_providers
SET recharge_url = '/api/v1/redeem/history?timezone=Asia%2FShanghai'
WHERE provider_type = 'sub2api'
  AND recharge_url = '/v1/redeem/history?timezone=Asia%2FShanghai';
