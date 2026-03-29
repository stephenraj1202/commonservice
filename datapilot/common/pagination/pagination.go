package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// Paginate applies LIMIT and OFFSET to the given GORM DB scope based on the
// supplied page and limit values.
//   - Page numbers less than 1 are defaulted to 1.
//   - Limit values greater than 100 are capped at 100.
//   - OFFSET = (page - 1) * limit
func Paginate(db *gorm.DB, page, limit int) *gorm.DB {
	if page < 1 {
		page = 1
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset := (page - 1) * limit
	return db.Limit(limit).Offset(offset)
}

// ParseParams reads the "page" and "limit" query parameters from the Gin
// context and returns safe integer values. Missing or invalid values fall back
// to page=1 and limit=20.
func ParseParams(c *gin.Context) (page, limit int) {
	page = defaultPage
	limit = defaultLimit

	if p, err := strconv.Atoi(c.Query("page")); err == nil && p >= 1 {
		page = p
	}
	if l, err := strconv.Atoi(c.Query("limit")); err == nil && l >= 1 {
		limit = l
	}
	return page, limit
}

// PagedResponse is the standard envelope returned by all paginated list
// endpoints. It carries the total record count alongside the current page,
// limit, and the slice of records.
type PagedResponse struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Limit int         `json:"limit"`
	Data  interface{} `json:"data"`
}
