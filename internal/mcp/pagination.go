package mcp

import (
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/paularlott/mcp"
)

// mcpPagination reads limit/offset from an MCP tool request and returns a clamped Pagination.
func mcpPagination(req *mcp.ToolRequest) model.Pagination {
	p := model.Pagination{
		Limit:  req.IntOr("limit", model.DefaultPageSize),
		Offset: req.IntOr("offset", 0),
	}
	p.Clamp()
	return p
}

// paginatedResponse wraps a list result with pagination metadata so the
// AI agent knows whether more results are available.
func paginatedResponse(items interface{}, count int, pg model.Pagination) map[string]interface{} {
	return map[string]interface{}{
		"items":    items,
		"count":    count,
		"limit":    pg.Limit,
		"offset":   pg.Offset,
		"has_more": count >= pg.Limit,
	}
}
