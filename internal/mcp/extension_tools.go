package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/pkg/extension"
)

// ToolRegistry registers an extension's MCP tools. It implements
// extension.ToolRegistry.
type ToolRegistry struct {
	s       *Server
	feature string
	errs    []error
}

// ExtensionTools returns a registry for the named extension's tools.
func (s *Server) ExtensionTools(feature string) *ToolRegistry {
	return &ToolRegistry{s: s, feature: feature}
}

// Register implements extension.ToolRegistry.
func (r *ToolRegistry) Register(tool *mcp.ToolBuilder, handler mcp.ToolHandler, opts ...extension.Option) {
	name := tool.Name()
	if _, exists := r.s.mcpServer.ToolSource(name); exists {
		r.errs = append(r.errs, fmt.Errorf("tool %q is already registered", name))
		return
	}
	if perm := extension.ApplyOptions(opts).Permission; perm != nil {
		next := handler
		handler = func(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
			if err := r.s.svc.Authorize(ctx, perm.Resource, perm.Action); err != nil {
				return nil, mcp.NewToolErrorInternal(err.Error())
			}
			return next(ctx, req)
		}
	}
	defer func() {
		if rec := recover(); rec != nil {
			r.errs = append(r.errs, fmt.Errorf("tool %q: %v", name, rec))
		}
	}()
	r.s.mcpServer.RegisterTool(tool, handler)
}

// Err reports the tools that could not be registered.
func (r *ToolRegistry) Err() error {
	if len(r.errs) == 0 {
		return nil
	}
	return fmt.Errorf("extension %s: %w", r.feature, errors.Join(r.errs...))
}

// AddInstructions appends a line to the server instructions, telling assistants
// what an extension's tools are for.
func (s *Server) AddInstructions(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	s.instructions = append(s.instructions, line)
	s.mcpServer.SetInstructions(s.base + "\n" + strings.Join(s.instructions, "\n"))
}
