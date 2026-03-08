package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerCustomFieldTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("custom_field_list", "List custom field definitions",
		).Discoverable("custom", "field", "definition", "metadata", "attribute"),
		s.handleCustomFieldList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("custom_field_get", "Get a custom field definition by ID",
			mcp.String("id", "Custom field definition ID", mcp.Required()),
		).Discoverable("custom", "field", "definition"),
		s.handleCustomFieldGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("custom_field_save", "Create or update a custom field definition",
			mcp.String("id", "Definition ID (omit for new)"),
			mcp.String("name", "Display name", mcp.Required()),
			mcp.String("key", "Unique key for API/queries", mcp.Required()),
			mcp.String("type", "Field type (text, number, boolean, select)", mcp.Required()),
			mcp.Boolean("required", "Whether the field is required"),
			mcp.StringArray("options", "Options for select type"),
			mcp.String("description", "Help text"),
		).Discoverable("custom", "field", "create", "update", "definition", "schema"),
		s.handleCustomFieldSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("custom_field_delete", "Delete a custom field definition",
			mcp.String("id", "Definition ID", mcp.Required()),
		).Discoverable("custom", "field", "delete", "remove"),
		s.handleCustomFieldDelete,
	)
}

func (s *Server) handleCustomFieldList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	if s.svc.CustomFields == nil {
		return mcp.NewToolResponseJSON([]interface{}{}), nil
	}
	fields, err := s.svc.CustomFields.ListDefinitions(ctx, nil)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(fields), nil
}

func (s *Server) handleCustomFieldGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	if s.svc.CustomFields == nil {
		return nil, mcp.NewToolErrorInternal("custom fields not configured")
	}
	id, _ := req.String("id")
	field, err := s.svc.CustomFields.GetDefinition(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(field), nil
}

func (s *Server) handleCustomFieldSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	if s.svc.CustomFields == nil {
		return nil, mcp.NewToolErrorInternal("custom fields not configured")
	}

	id := req.StringOr("id", "")
	name, _ := req.String("name")
	key, _ := req.String("key")
	fieldType := model.CustomFieldType(req.StringOr("type", ""))

	if id == "" {
		createReq := &model.CreateCustomFieldDefinitionRequest{
			Name:        name,
			Key:         key,
			Type:        fieldType,
			Required:    req.BoolOr("required", false),
			Options:     req.StringSliceOr("options", []string{}),
			Description: req.StringOr("description", ""),
		}
		field, err := s.svc.CustomFields.CreateDefinition(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(field), nil
	}

	updateReq := &model.UpdateCustomFieldDefinitionRequest{
		Name: &name,
		Key:  &key,
		Type: &fieldType,
	}
	if v := req.StringOr("description", ""); v != "" {
		updateReq.Description = &v
	}
	required := req.BoolOr("required", false)
	updateReq.Required = &required

	field, err := s.svc.CustomFields.UpdateDefinition(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(field), nil
}

func (s *Server) handleCustomFieldDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	if s.svc.CustomFields == nil {
		return nil, mcp.NewToolErrorInternal("custom fields not configured")
	}
	id, _ := req.String("id")
	if err := s.svc.CustomFields.DeleteDefinition(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}
