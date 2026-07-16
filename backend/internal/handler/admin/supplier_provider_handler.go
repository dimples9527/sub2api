package admin

import (
	"strconv"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const supplierProviderMaxPageSize = 200

type SupplierProviderHandler struct {
	service *service.SupplierProviderService
}

type SupplierProviderTypeHandler struct {
	service *service.SupplierProviderTypeService
}

func NewSupplierProviderHandler(service *service.SupplierProviderService) *SupplierProviderHandler {
	return &SupplierProviderHandler{service: service}
}

func NewSupplierProviderTypeHandler(service *service.SupplierProviderTypeService) *SupplierProviderTypeHandler {
	return &SupplierProviderTypeHandler{service: service}
}

func (h *SupplierProviderHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	if pageSize > supplierProviderMaxPageSize {
		pageSize = supplierProviderMaxPageSize
	}

	result, err := h.service.List(c.Request.Context(), service.SupplierProviderListParams{
		Search:   strings.TrimSpace(c.Query("search")),
		Enabled:  parseSupplierProviderEnabled(c.Query("enabled")),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *SupplierProviderHandler) Get(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	provider, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, provider)
}

func (h *SupplierProviderHandler) Create(c *gin.Context) {
	var req service.SupplierProviderUpsertParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	provider, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, provider)
}

func (h *SupplierProviderHandler) Update(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	var req service.SupplierProviderUpsertParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	provider, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, provider)
}

func (h *SupplierProviderHandler) Delete(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "供应商已删除"})
}

func (h *SupplierProviderHandler) SetDefault(c *gin.Context) {
	id, ok := parseSupplierProviderID(c)
	if !ok {
		return
	}
	provider, err := h.service.SetDefault(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, provider)
}

func parseSupplierProviderID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_SUPPLIER_PROVIDER_ID", "invalid supplier provider id"))
		return 0, false
	}
	return id, true
}

func parseSupplierProviderEnabled(raw string) *bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes":
		value := true
		return &value
	case "false", "0", "no":
		value := false
		return &value
	default:
		return nil
	}
}

func (h *SupplierProviderTypeHandler) List(c *gin.Context) {
	enabledOnly := parseSupplierProviderEnabled(c.Query("enabled_only"))
	items, err := h.service.List(c.Request.Context(), enabledOnly != nil && *enabledOnly)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *SupplierProviderTypeHandler) Get(c *gin.Context) {
	id, ok := parseSupplierProviderTypeID(c)
	if !ok {
		return
	}
	providerType, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, providerType)
}

func (h *SupplierProviderTypeHandler) Create(c *gin.Context) {
	var req service.SupplierProviderTypeUpsertParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	providerType, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, providerType)
}

func (h *SupplierProviderTypeHandler) Update(c *gin.Context) {
	id, ok := parseSupplierProviderTypeID(c)
	if !ok {
		return
	}
	var req service.SupplierProviderTypeUpsertParams
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	providerType, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, providerType)
}

func (h *SupplierProviderTypeHandler) Delete(c *gin.Context) {
	id, ok := parseSupplierProviderTypeID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "供应商类型已删除"})
}

func parseSupplierProviderTypeID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.ErrorFrom(c, infraerrors.BadRequest("INVALID_SUPPLIER_PROVIDER_TYPE_ID", "invalid supplier provider type id"))
		return 0, false
	}
	return id, true
}
