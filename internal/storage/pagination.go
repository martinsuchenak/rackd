package storage

import "github.com/martinsuchenak/rackd/pkg/model"

// appendPagination adds LIMIT and OFFSET clauses to a query.
//
// A nil or zero Pagination means no page was asked for, and every row is
// returned: internal callers rely on that to see all records, and the API's
// list endpoints document it. A caller that wants a page sets Limit (capped at
// model.MaxPageSize) or Offset; an Offset alone gets the default page size.
func appendPagination(query string, args []interface{}, p *model.Pagination) (string, []interface{}) {
	if p == nil || (p.Limit <= 0 && p.Offset <= 0) {
		return query, args
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
