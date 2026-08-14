package admin

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelSquareSyncAccountCandidate struct {
	ID                int64    `json:"id"`
	Name              string   `json:"name"`
	Platform          string   `json:"platform"`
	Type              string   `json:"type"`
	Status            string   `json:"status"`
	GroupIDs          []int64  `json:"group_ids,omitempty"`
	GroupNames        []string `json:"group_names,omitempty"`
	EffectivePlatform string   `json:"effective_platform,omitempty"`
}

func (h *AccountHandler) ListModelSquareSyncAccounts(c *gin.Context) {
	platform := strings.ToLower(strings.TrimSpace(c.Query("platform")))
	if platform == "" {
		response.BadRequest(c, "platform is required")
		return
	}
	if h.monitorGroupPlatformOverrideService == nil {
		response.ErrorFrom(c, fmt.Errorf("monitor group platform override service is not initialized"))
		return
	}
	if h.adminService == nil {
		response.ErrorFrom(c, fmt.Errorf("admin service is not initialized"))
		return
	}

	groups, err := h.adminService.GetAllGroupsIncludingInactive(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	groupIDs := make([]int64, 0, len(groups))
	groupNames := make(map[int64]string, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.ID)
		groupNames[group.ID] = group.Name
	}
	overrides := map[int64]service.MonitorGroupPlatformOverride{}
	if len(groupIDs) > 0 {
		overrides, err = h.monitorGroupPlatformOverrideService.ListByGroupIDs(c.Request.Context(), groupIDs)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	matchingGroupIDs := make(map[int64]struct{})
	for _, group := range groups {
		effective := strings.ToLower(strings.TrimSpace(group.Platform))
		if actual := strings.ToLower(strings.TrimSpace(overrides[group.ID].ActualPlatform)); actual != "" {
			effective = actual
		}
		if effective == platform {
			matchingGroupIDs[group.ID] = struct{}{}
		}
	}
	accounts, _, err := h.adminService.ListAccounts(c.Request.Context(), 1, 10000, "", "", "", "", 0, "", "name", "asc")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result := make([]ModelSquareSyncAccountCandidate, 0, len(accounts))
	seen := make(map[int64]struct{}, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if account == nil || account.ID <= 0 {
			continue
		}
		if !h.accountMatchesModelSquarePlatform(account, platform, matchingGroupIDs) {
			continue
		}
		if _, ok := seen[account.ID]; ok {
			continue
		}
		seen[account.ID] = struct{}{}
		candidate := ModelSquareSyncAccountCandidate{
			ID:       account.ID,
			Name:     account.Name,
			Platform: account.Platform,
			Type:     account.Type,
			Status:   account.Status,
		}
		if len(account.GroupIDs) > 0 {
			candidate.GroupIDs = append(candidate.GroupIDs, account.GroupIDs...)
		}
		candidate.GroupNames = modelSquareAccountGroupNames(account, groupNames)
		candidate.EffectivePlatform = platform
		result = append(result, candidate)
	}
	response.Success(c, result)
}

func modelSquareAccountGroupNames(account *service.Account, groupNames map[int64]string) []string {
	if account == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(account.GroupIDs)+len(account.Groups))
	names := make([]string, 0, len(account.GroupIDs)+len(account.Groups))
	for _, groupID := range account.GroupIDs {
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		if name := strings.TrimSpace(groupNames[groupID]); name != "" {
			names = append(names, name)
		}
	}
	for _, group := range account.Groups {
		if group == nil {
			continue
		}
		if _, ok := seen[group.ID]; ok {
			continue
		}
		seen[group.ID] = struct{}{}
		if name := strings.TrimSpace(group.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (h *AccountHandler) accountMatchesModelSquarePlatform(account *service.Account, platform string, matchingGroupIDs map[int64]struct{}) bool {
	if account == nil {
		return false
	}
	hasGroups := len(account.GroupIDs) > 0 || len(account.Groups) > 0
	for _, groupID := range account.GroupIDs {
		if _, ok := matchingGroupIDs[groupID]; ok {
			return true
		}
	}
	for _, group := range account.Groups {
		if group == nil {
			continue
		}
		if _, ok := matchingGroupIDs[group.ID]; ok {
			return true
		}
	}
	return !hasGroups && strings.ToLower(strings.TrimSpace(account.Platform)) == platform
}
