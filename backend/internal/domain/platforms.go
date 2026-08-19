package domain

import "strings"

type PlatformDefinition struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	HealthGuard bool   `json:"health_guard"`
}

var CorePlatformDefinitions = []PlatformDefinition{
	{Code: PlatformAnthropic, Name: "Anthropic", Color: "#f97316", HealthGuard: true},
	{Code: PlatformOpenAI, Name: "OpenAI", Color: "#22c55e", HealthGuard: true},
	{Code: PlatformGemini, Name: "Gemini", Color: "#3b82f6", HealthGuard: true},
	{Code: PlatformAntigravity, Name: "Antigravity", Color: "#a855f7", HealthGuard: true},
	{Code: PlatformGrok, Name: "Grok", Color: "#71717a", HealthGuard: true},
	{Code: PlatformKimi, Name: "Kimi", Color: "#ec4899", HealthGuard: true},
	{Code: PlatformZhipu, Name: "Zhipu GLM", Color: "#6366f1", HealthGuard: true},
	{Code: PlatformDeepseek, Name: "DeepSeek", Color: "#14b8a6", HealthGuard: true},
	{Code: PlatformComposite, Name: "Composite", Color: "#06b6d4", HealthGuard: false},
}

var CorePlatformCodes = platformCodes(CorePlatformDefinitions)

func platformCodes(definitions []PlatformDefinition) []string {
	codes := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		codes = append(codes, definition.Code)
	}
	return codes
}

func IsCorePlatform(platform string) bool {
	normalized := strings.ToLower(strings.TrimSpace(platform))
	for _, definition := range CorePlatformDefinitions {
		if normalized == definition.Code {
			return true
		}
	}
	return false
}

func ListCorePlatformDefinitions() []PlatformDefinition {
	definitions := make([]PlatformDefinition, len(CorePlatformDefinitions))
	copy(definitions, CorePlatformDefinitions)
	return definitions
}
