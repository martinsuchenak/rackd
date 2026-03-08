package storage

import "github.com/martinsuchenak/rackd/internal/model"

// appendPagination adds LIMIT and OFFSET clauses to a query.
// If p is nil, default pagination is applied.
func appendPagination(query string, args []interface{}, p *model.Pagination) (string, []interface{}) {
	if p == nil {
		p = &model.Pagination{}
	}
	p.Clamp()
	query += " LIMIT ?"
	args = append(args, p.Limit)
	if p.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, p.Offset)
	}
	return query, args
}
