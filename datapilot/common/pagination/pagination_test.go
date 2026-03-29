package pagination_test

// Feature: datapilot-platform, Property 6: Pagination applies correct LIMIT and OFFSET
// Feature: datapilot-platform, Property 7: Pagination caps limit at 100
// Feature: datapilot-platform, Property 8: Pagination defaults sub-1 page to page 1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"datapilot/common/pagination"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// scopeDB creates a bare *gorm.DB with no dialector — sufficient for
// inspecting the Statement fields set by Limit/Offset scopes.
func scopeDB() *gorm.DB {
	db, _ := gorm.Open(nil, &gorm.Config{})
	return db
}

// extractLimitOffset calls Paginate and reads the Limit/Offset values that
// were stored in the LIMIT clause by the scope functions.
func extractLimitOffset(page, limit int) (gotLimit, gotOffset int) {
	db := scopeDB()
	scoped := pagination.Paginate(db, page, limit)
	stmt := scoped.Statement
	if stmt == nil {
		return -1, -1
	}
	c, ok := stmt.Clauses["LIMIT"]
	if !ok {
		return -1, -1
	}
	lc, ok := c.Expression.(clause.Limit)
	if !ok {
		return -1, -1
	}
	if lc.Limit == nil {
		return -1, lc.Offset
	}
	return *lc.Limit, lc.Offset
}

// --- Unit tests ---

func TestPaginate_NormalCase(t *testing.T) {
	gotLimit, gotOffset := extractLimitOffset(2, 10)
	if gotLimit != 10 {
		t.Errorf("expected LIMIT 10, got %d", gotLimit)
	}
	if gotOffset != 10 {
		t.Errorf("expected OFFSET 10, got %d", gotOffset)
	}
}

func TestPaginate_FirstPage(t *testing.T) {
	gotLimit, gotOffset := extractLimitOffset(1, 20)
	if gotLimit != 20 {
		t.Errorf("expected LIMIT 20, got %d", gotLimit)
	}
	if gotOffset != 0 {
		t.Errorf("expected OFFSET 0, got %d", gotOffset)
	}
}

func TestPaginate_PageZeroDefaultsToOne(t *testing.T) {
	_, gotOffset := extractLimitOffset(0, 10)
	if gotOffset != 0 {
		t.Errorf("expected OFFSET 0 for page=0, got %d", gotOffset)
	}
}

func TestPaginate_NegativePageDefaultsToOne(t *testing.T) {
	_, gotOffset := extractLimitOffset(-5, 10)
	if gotOffset != 0 {
		t.Errorf("expected OFFSET 0 for page=-5, got %d", gotOffset)
	}
}

func TestPaginate_LimitCappedAt100(t *testing.T) {
	gotLimit, _ := extractLimitOffset(1, 500)
	if gotLimit != 100 {
		t.Errorf("expected LIMIT 100 for limit=500, got %d", gotLimit)
	}
}

func TestPaginate_LimitExactly100(t *testing.T) {
	gotLimit, _ := extractLimitOffset(1, 100)
	if gotLimit != 100 {
		t.Errorf("expected LIMIT 100, got %d", gotLimit)
	}
}

func TestPaginate_LimitExactly101Capped(t *testing.T) {
	gotLimit, _ := extractLimitOffset(1, 101)
	if gotLimit != 100 {
		t.Errorf("expected LIMIT 100 for limit=101, got %d", gotLimit)
	}
}

func TestParseParams_Defaults(t *testing.T) {
	c, _ := newGinContext("/items")
	page, limit := pagination.ParseParams(c)
	if page != 1 {
		t.Errorf("expected default page=1, got %d", page)
	}
	if limit != 20 {
		t.Errorf("expected default limit=20, got %d", limit)
	}
}

func TestParseParams_ValidValues(t *testing.T) {
	c, _ := newGinContext("/items?page=3&limit=50")
	page, limit := pagination.ParseParams(c)
	if page != 3 {
		t.Errorf("expected page=3, got %d", page)
	}
	if limit != 50 {
		t.Errorf("expected limit=50, got %d", limit)
	}
}

func TestParseParams_InvalidStrings(t *testing.T) {
	c, _ := newGinContext("/items?page=abc&limit=xyz")
	page, limit := pagination.ParseParams(c)
	if page != 1 {
		t.Errorf("expected default page=1, got %d", page)
	}
	if limit != 20 {
		t.Errorf("expected default limit=20, got %d", limit)
	}
}

func TestParseParams_ZeroValues(t *testing.T) {
	c, _ := newGinContext("/items?page=0&limit=0")
	page, limit := pagination.ParseParams(c)
	if page != 1 {
		t.Errorf("expected default page=1 for page=0, got %d", page)
	}
	if limit != 20 {
		t.Errorf("expected default limit=20 for limit=0, got %d", limit)
	}
}

func TestPagedResponse_Fields(t *testing.T) {
	resp := pagination.PagedResponse{
		Total: 42,
		Page:  2,
		Limit: 10,
		Data:  []string{"a", "b"},
	}
	if resp.Total != 42 || resp.Page != 2 || resp.Limit != 10 {
		t.Errorf("unexpected PagedResponse fields: %+v", resp)
	}
}

// --- Property tests ---

// Property 6: Pagination applies correct LIMIT and OFFSET
// Validates: Requirements 5.1
func TestProperty6_PaginateLimitOffset(t *testing.T) {
	p := gopter.DefaultTestParameters()
	p.MinSuccessfulTests = 20
	params := gopter.NewProperties(p)

	params.Property("LIMIT and OFFSET match page/limit inputs", prop.ForAll(
		func(page, limit int) bool {
			gotLimit, gotOffset := extractLimitOffset(page, limit)
			expectedOffset := (page - 1) * limit
			return gotLimit == limit && gotOffset == expectedOffset
		},
		gen.IntRange(1, 100), // page ≥ 1
		gen.IntRange(1, 100), // limit 1–100
	))

	params.TestingRun(t, gopter.ConsoleReporter(false))
}

// Property 7: Pagination caps limit at 100
// Validates: Requirements 5.2
func TestProperty7_PaginateCapsLimit(t *testing.T) {
	p := gopter.DefaultTestParameters()
	p.MinSuccessfulTests = 20
	params := gopter.NewProperties(p)

	params.Property("limit > 100 is capped to 100", prop.ForAll(
		func(limit int) bool {
			gotLimit, _ := extractLimitOffset(1, limit)
			return gotLimit == 100
		},
		gen.IntRange(101, 10000),
	))

	params.TestingRun(t, gopter.ConsoleReporter(false))
}

// Property 8: Pagination defaults sub-1 page to page 1
// Validates: Requirements 5.3
func TestProperty8_PaginateSubOnePage(t *testing.T) {
	p := gopter.DefaultTestParameters()
	p.MinSuccessfulTests = 20
	params := gopter.NewProperties(p)

	params.Property("page ≤ 0 produces OFFSET 0", prop.ForAll(
		func(page, limit int) bool {
			_, gotOffset := extractLimitOffset(page, limit)
			return gotOffset == 0
		},
		gen.IntRange(-1000, 0), // page ≤ 0
		gen.IntRange(1, 100),   // valid limit
	))

	params.TestingRun(t, gopter.ConsoleReporter(false))
}

// --- helpers ---

func newGinContext(url string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	c.Request = req
	return c, w
}
