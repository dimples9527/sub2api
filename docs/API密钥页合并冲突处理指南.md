# API 密钥页合并冲突处理指南

## 背景

为降低 `frontend/src/views/user/KeysView.vue` 与主分支合并时的冲突面，密钥相关能力已拆为：

| 模块 | 路径 | 职责 |
|------|------|------|
| 页面编排 | `frontend/src/views/user/KeysView.vue` | 列表、筛选、分页、打开弹窗/浮层、调用删除/重置等流程 |
| 创建/编辑弹窗 | `frontend/src/components/keys/KeyFormModal.vue` | 创建与编辑表单 UI、平台过滤、分组倍率排序、提交保存 |
| 列表切换分组浮层 | `frontend/src/components/keys/KeyGroupSelectorPopover.vue` | 列表点击分组后的平台过滤、搜索、分组选择 |
| 过滤排序工具 | `frontend/src/utils/keyFormGroupOptions.ts` | 平台过滤（含 composite）、倍率升序、平台选项构建 |

**原则：业务规则优先改工具函数；UI 交互优先改对应组件；KeysView 只做编排。**

## 合并前先判断冲突落点

拿到冲突后，先看冲突文件，不要一上来全量保留某一方：

1. 只冲突 `keyFormGroupOptions.ts`
   - 优先手工合并函数逻辑
   - 以“选中平台时包含 composite + 倍率升序”为产品底线
2. 只冲突 `KeyFormModal.vue` / `KeyGroupSelectorPopover.vue`
   - 通常与主分支 KeysView 无关
   - 按组件内 UI/交互解决，再回归对应场景
3. 冲突在 `KeysView.vue`
   - 先看是不是“引用组件的那几行”或“open/close 编排”
   - 不要把已经抽走的表单/浮层大段 UI 重新粘回 KeysView
4. 冲突在 i18n
   - `keys.platformLabel` / `keys.allPlatforms` / `keys.selectPlatform` 等 key 两边都加时，保留双方新增项即可

## 常见冲突场景与处理

### 场景 A：主分支也改了创建/编辑弹窗

**现象：** `KeysView.vue` 里出现大段表单冲突，或主分支仍内联 `BaseDialog` 表单。

**处理：**

1. 以本分支的 `KeyFormModal.vue` 为承载方，把主分支新增的表单字段/校验迁入该组件
2. `KeysView.vue` 只保留：

```vue
<KeyFormModal
  :show="showCreateModal || showEditModal"
  :mode="showEditModal ? 'edit' : 'create'"
  :group-options="groupOptions"
  :editing-key="selectedKey"
  @close="closeKeyFormModal"
  @saved="onKeyFormSaved"
  @request-reset-quota="showResetQuotaDialog = true"
  @request-reset-rate-limit="showResetRateLimitDialog = true"
/>
```

3. 不要在 KeysView 恢复 `formData` / `handleSubmit` 大段逻辑

### 场景 B：主分支也改了列表分组切换浮层

**现象：** 列表点击分组相关模板或 `openGroupSelector` 冲突。

**处理：**

1. 浮层 UI 与平台下拉样式以 `KeyGroupSelectorPopover.vue` 为准
2. KeysView 只负责：
   - 根据按钮位置计算 `dropdownPosition`
   - 维护 `groupSelectorKeyId`
   - 在 `@select` 里调用 `changeGroup`
3. 若主分支改了点击外部关闭逻辑，合并时保留：
   - 点触发器/浮层内不关闭
   - 点浮层内非平台区域时，通过组件 `handleInsideClick` 收起平台菜单

### 场景 C：双方都改了平台过滤/排序规则

**现象：** `keyFormGroupOptions.ts` 冲突。

**处理顺序：**

1. 先对齐产品规则：
   - 选中具体平台时，结果需包含 `composite`
   - 分组默认按 `rate`（倍率）升序，相同再按名称
2. 再合并实现，避免页面和浮层各写一套
3. 跑工具单测：`frontend/src/utils/__tests__/keyFormGroupOptions.spec.ts`

### 场景 D：主分支尚未抽离，本分支已抽离

**现象：** 主分支改的是旧内联代码，本分支文件已不存在对应片段。

**处理：**

1. 不要用主分支大段覆盖本分支 KeysView
2. 阅读主分支 diff 的“意图”（新字段、新校验、文案、接口参数）
3. 把意图移植到：
   - 表单相关 -> `KeyFormModal.vue`
   - 列表切换分组 -> `KeyGroupSelectorPopover.vue`
   - 过滤排序规则 -> `keyFormGroupOptions.ts`
4. KeysView 仅补传参或事件编排

## 推荐合并步骤

```bash
# 1. 更新主分支
git fetch origin
git merge origin/main
# 或
git rebase origin/main

# 2. 查看冲突文件
git status

# 3. 按模块逐个解决，优先组件/工具，最后 KeysView

# 4. 解决后检查是否误把大段 UI 粘回 KeysView
git diff -- frontend/src/views/user/KeysView.vue
```

解决后至少执行：

```bash
cd frontend
pnpm exec vitest run src/utils/__tests__/keyFormGroupOptions.spec.ts src/components/keys/__tests__/KeyGroupSelectorPopover.spec.ts src/views/user/__tests__/KeysView.spec.ts
```

如本机 `__tests__` 被 gitignore，测试文件需：

```bash
git add -f frontend/src/utils/__tests__/keyFormGroupOptions.spec.ts
git add -f frontend/src/components/keys/__tests__/KeyGroupSelectorPopover.spec.ts
```

## 人工回归清单

- [ ] 创建密钥：平台过滤、分组倍率升序、提交成功
- [ ] 编辑密钥：回填正确；切换平台后非法分组被清空
- [ ] 列表点击分组：平台下拉样式、选项列表样式、搜索、切换成功
- [ ] 选中具体平台时，列表中仍能看到 composite 分组
- [ ] 深色模式下平台下拉与分组列表可读
- [ ] 删除/重置额度/重置速率限制弹窗仍可用（它们仍由 KeysView 编排）

## 什么时候继续抽离

若后续主分支仍频繁改 KeysView，可继续下沉：

1. 列表列配置与工具栏
2. 删除/重置确认流程
3. CCS 客户端选择弹窗

目标始终是：**KeysView 越来越薄，冲突越来越局部。**
