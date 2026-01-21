package server

import (
	"net/http"
	"testing"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/mcp"
)

type mockFeature struct {
	name           string
	routesCalled   bool
	mcpCalled      bool
	uiCalled       bool
}

func (f *mockFeature) Name() string { return f.name }
func (f *mockFeature) RegisterRoutes(mux *http.ServeMux) { f.routesCalled = true }
func (f *mockFeature) RegisterMCPTools(s *mcp.Server) { f.mcpCalled = true }
func (f *mockFeature) ConfigureUI(b *api.UIConfigBuilder) { f.uiCalled = true }

func TestFeatureInterface(t *testing.T) {
	f := &mockFeature{name: "test-feature"}
	
	var feature Feature = f
	if feature.Name() != "test-feature" {
		t.Errorf("expected name 'test-feature', got %s", feature.Name())
	}
	
	feature.RegisterRoutes(http.NewServeMux())
	if !f.routesCalled {
		t.Error("RegisterRoutes not called")
	}
	
	feature.ConfigureUI(api.NewUIConfigBuilder())
	if !f.uiCalled {
		t.Error("ConfigureUI not called")
	}
}
