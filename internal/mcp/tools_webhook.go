package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerWebhookTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("webhook_list", "List webhooks",
		).Discoverable("webhook", "notification", "event", "callback", "http"),
		s.handleWebhookList,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("webhook_get", "Get a webhook by ID",
			mcp.String("id", "Webhook ID", mcp.Required()),
		).Discoverable("webhook", "notification", "event"),
		s.handleWebhookGet,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("webhook_save", "Create or update a webhook",
			mcp.String("id", "Webhook ID (omit for new)"),
			mcp.String("name", "Webhook name", mcp.Required()),
			mcp.String("url", "Target URL", mcp.Required()),
			mcp.String("secret", "HMAC signing secret"),
			mcp.StringArray("events", "Event types to subscribe to"),
			mcp.Boolean("active", "Whether the webhook is active"),
			mcp.String("description", "Description"),
		).Discoverable("webhook", "create", "update", "notification", "event", "callback"),
		s.handleWebhookSave,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("webhook_delete", "Delete a webhook",
			mcp.String("id", "Webhook ID", mcp.Required()),
		).Discoverable("webhook", "delete", "remove"),
		s.handleWebhookDelete,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("webhook_ping", "Send a test ping to a webhook",
			mcp.String("id", "Webhook ID", mcp.Required()),
		).Discoverable("webhook", "ping", "test", "check"),
		s.handleWebhookPing,
	)
}

func (s *Server) handleWebhookList(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	webhooks, err := s.svc.Webhooks.List(ctx, &model.WebhookFilter{})
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(webhooks), nil
}

func (s *Server) handleWebhookGet(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	webhook, err := s.svc.Webhooks.Get(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(webhook), nil
}

func (s *Server) handleWebhookSave(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id := req.StringOr("id", "")
	name, _ := req.String("name")
	url, _ := req.String("url")

	// Convert string slice to EventType slice
	eventStrs := req.StringSliceOr("events", []string{})
	events := make([]model.EventType, 0, len(eventStrs))
	for _, e := range eventStrs {
		events = append(events, model.EventType(e))
	}

	if id == "" {
		createReq := &model.CreateWebhookRequest{
			Name:        name,
			URL:         url,
			Secret:      req.StringOr("secret", ""),
			Events:      events,
			Active:      req.BoolOr("active", true),
			Description: req.StringOr("description", ""),
		}
		webhook, err := s.svc.Webhooks.Create(ctx, createReq)
		if err != nil {
			return nil, mcp.NewToolErrorInternal(err.Error())
		}
		return mcp.NewToolResponseJSON(webhook), nil
	}

	active := req.BoolOr("active", true)
	updateReq := &model.UpdateWebhookRequest{
		Name:   &name,
		URL:    &url,
		Active: &active,
	}
	if v := req.StringOr("description", ""); v != "" {
		updateReq.Description = &v
	}
	if v := req.StringOr("secret", ""); v != "" {
		updateReq.Secret = &v
	}
	if len(events) > 0 {
		updateReq.Events = &events
	}

	webhook, err := s.svc.Webhooks.Update(ctx, id, updateReq)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(webhook), nil
}

func (s *Server) handleWebhookDelete(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	if err := s.svc.Webhooks.Delete(ctx, id); err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(map[string]string{"status": "deleted", "id": id}), nil
}

func (s *Server) handleWebhookPing(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	id, _ := req.String("id")
	delivery, err := s.svc.Webhooks.Ping(ctx, id)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(delivery), nil
}
