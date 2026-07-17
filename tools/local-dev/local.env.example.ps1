$Sub2ApiDev = @{
    PostgresContainer = 'pgsql-local-5433'
    RedisContainer = 'redis-local'
    DatabaseAdminUser = 'postgres'
    DatabaseHost = '127.0.0.1'
    DatabasePort = 5433
    DatabaseName = 'sub2api_dev'
    DatabaseUser = 'sub2api_dev'
    DatabasePassword = ''
    RedisHost = '127.0.0.1'
    RedisPort = 6379
    RedisPassword = ''
    BackendHost = '127.0.0.1'
    BackendPort = 4000
    FrontendHost = '127.0.0.1'
    FrontendPort = 5173
    AdminEmail = 'admin@sub2api.local'
    AdminPassword = ''
    # 固定 AES-256-GCM 加密密钥，64 位 hex。请在本机 local.env.ps1 中设置，避免重启后已保存凭据无法解密。
    TotpEncryptionKey = ''
    GoExe = ''
}
