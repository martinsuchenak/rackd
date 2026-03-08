package mcp

import (
	"context"

	"github.com/paularlott/mcp"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (s *Server) registerDiscoveryTools() {
	s.mcpServer.RegisterTool(
		mcp.NewTool("discovery_scan", "Start a network discovery scan",
			mcp.String("network_id", "Network ID to scan", mcp.Required()),
			mcp.String("scan_type", "Scan type: quick, full, deep"),
		).Discoverable("discovery", "scan", "network", "probe", "nmap", "detect"),
		s.handleStartScan,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("discovery_list", "List discovered devices",
			mcp.String("network_id", "Filter by network ID"),
		).Discoverable("discovery", "scan", "list", "found", "detected"),
		s.handleListDiscovered,
	)

	s.mcpServer.RegisterTool(
		mcp.NewTool("discovery_promote", "Promote a discovered device to inventory",
			mcp.String("discovered_id", "Discovered device ID", mcp.Required()),
			mcp.String("name", "Device name", mcp.Required()),
		).Discoverable("discovery", "promote", "import", "inventory", "add"),
		s.handlePromoteDevice,
	)
}

func (s *Server) handleStartScan(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	networkID, _ := req.String("network_id")
	scanType := req.StringOr("scan_type", model.ScanTypeQuick)

	if scanType != model.ScanTypeQuick && scanType != model.ScanTypeFull && scanType != model.ScanTypeDeep {
		scanType = model.ScanTypeQuick
	}

	scan, err := s.svc.Discovery.StartScan(ctx, networkID, scanType)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(scan), nil
}

func (s *Server) handleListDiscovered(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	networkID := req.StringOr("network_id", "")
	devices, err := s.svc.Discovery.ListDevices(ctx, networkID)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(devices), nil
}

func (s *Server) handlePromoteDevice(ctx context.Context, req *mcp.ToolRequest) (*mcp.ToolResponse, error) {
	discoveredID, _ := req.String("discovered_id")
	name, _ := req.String("name")

	device := &model.Device{
		Name:    name,
		Tags:    []string{},
		Domains: []string{},
	}

	promoted, err := s.svc.Discovery.PromoteDevice(ctx, discoveredID, device)
	if err != nil {
		return nil, mcp.NewToolErrorInternal(err.Error())
	}
	return mcp.NewToolResponseJSON(promoted), nil
}
