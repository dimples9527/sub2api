# Task Plan: 供应商 Sub2API 同步与自动化任务

## Goal

在独立供应商管理模块中实现 Sub2API 登录 token Redis 缓存、账号/分组/余额/成本同步、本地持久化、自动化调度和定期清理，并将真实任务维护接入供应商自动化任务中心。

## Current Phase

Phase 5

## Phases

### Phase 1: Requirements & Discovery
- [x] 确认 Sub2API 使用 email + password 登录
- [x] 确认 token 使用 Redis 缓存并在 401 后重登一次
- [x] 确认业务数据落库并由页面查询本地数据
- [x] 确认同步与清理任务进入自动化任务中心
- **Status:** complete

### Phase 2: Planning & Structure
- [x] 完成设计文档
- [x] 映射后端与前端改动文件
- [x] 编写详细 TDD 实施计划
- **Status:** complete

### Phase 3: Backend Implementation
- [x] 增加数据库迁移和 Repository
- [x] 增加 Redis token 缓存
- [x] 增加 Sub2API 客户端和同步服务
- [x] 增加自动化任务服务和调度器
- [x] 增加后台管理接口和依赖注入
- **Status:** complete

### Phase 4: Frontend Implementation
- [x] 接入供应商同步操作
- [x] 重写自动化任务中心真实页面
- [x] 接入本地账号和分组查询
- **Status:** complete

### Phase 5: Testing & Verification
- [x] 运行后端定向测试
- [x] 运行前端类型检查和构建
- [x] 重编译并重启本地后端
- [ ] 浏览器验证主要操作
- **Status:** in_progress

### Phase 6: Delivery
- [ ] 检查任务相关 diff 和 UTF-8
- [ ] 检查敏感信息与无关改动
- [ ] 向用户说明结果和使用方式
- **Status:** pending

## Key Questions

1. 如何避免并发请求重复登录？使用 Redis 登录锁和 token 轮询。
2. 如何避免同步数据无限增长？当前状态覆盖、每日统计 upsert、历史表按保留周期批量清理。
3. 如何避免影响旧模块？新建供应商专用 service/repository/cache/handler，旧上游业务文件不修改。

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Redis 缓存 access token | 支持重启和多实例共享，TTL 自动清理 |
| 401 后清缓存并重试一次 | 恢复失效 token，同时避免无限重试 |
| 业务数据写入独立供应商表 | 页面查询稳定，不依赖上游实时可用性 |
| 同步与清理使用独立任务 | 两类任务频率和失败影响不同 |
| 前端中文直接写入 | 遵守用户不使用国际化的要求 |
| 不执行 git commit | 用户未明确要求提交 |

## Errors Encountered

| Error | Attempt | Resolution |
|-------|---------|------------|
| 旧本地后端未重编译导致相对接口地址校验仍失败 | 1 | 使用 `dev.bat restart-backend` 重编译并重启 |
| 设计文档目录被 `docs/*` 忽略 | 1 | 保留本地设计文档，不执行提交 |
| 实施计划大范围补强补丁上下文不匹配 | 1 | 改用按任务小段精确补丁，不重复提交原补丁 |
| 计划中的 `supplier_automation_scheduler.go` 文件不存在 | 1 | 调度器实现位于 `supplier_automation_service.go`，格式化时改用实际存在文件 |
| 内置浏览器工具无法执行 Node REPL 脚本 | 1 | 改用本地 HTTP/接口挂载验证，并在交付说明中保留限制 |
| 广域后端测试命令外层超时 | 1 | 使用 Go `-timeout=120s -v` 定位到既有非供应商失败/超时 |

## Notes

- 工作区已有大量本次供应商模块及其他未提交改动，不回退、不覆盖无关文件。
- 旧上游管理模块仅作为协议参考，禁止调用旧业务 service/API。
- 所有生产代码遵循先写失败测试、再实现的 TDD 顺序。
