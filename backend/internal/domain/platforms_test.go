package domain

import "testing"

// TestCorePlatformDefinitionsCoverAllPlatforms 把平台目录钉死，与前端
// platformOptions.spec.ts 对称：目录是 /admin/platforms 的唯一数据源，前端启动时
// 用它整体覆盖本地兜底列表。新增平台常量若忘了在这里登记，就会在各筛选下拉里静默消失
// （minimax / opencode_go 曾漏过），所以这条用例强制目录覆盖全部已知平台。
func TestCorePlatformDefinitionsCoverAllPlatforms(t *testing.T) {
	want := []string{
		PlatformAnthropic,
		PlatformOpenAI,
		PlatformGemini,
		PlatformAntigravity,
		PlatformGrok,
		PlatformKimi,
		PlatformZhipu,
		PlatformDeepseek,
		PlatformMiniMax,
		PlatformOpenCodeGo,
		PlatformComposite,
	}

	got := make([]string, 0, len(CorePlatformDefinitions))
	seen := make(map[string]bool, len(CorePlatformDefinitions))
	for _, def := range CorePlatformDefinitions {
		if def.Code == "" {
			t.Fatalf("catalog entry has empty code: %+v", def)
		}
		if def.Name == "" {
			t.Fatalf("catalog entry %q has empty name", def.Code)
		}
		if seen[def.Code] {
			t.Fatalf("catalog has duplicate code %q", def.Code)
		}
		seen[def.Code] = true
		got = append(got, def.Code)
	}

	if len(got) != len(want) {
		t.Fatalf("catalog code count = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("catalog code[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}
