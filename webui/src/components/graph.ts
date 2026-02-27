// Device Relationship Graph Visualization

import cytoscape from 'cytoscape';
// @ts-ignore - no types available
import panzoom from 'cytoscape-panzoom';
import type { Device, DeviceRelationship, Datacenter, DeviceStatus } from '../core/types';
import { api, RackdAPIError } from '../core/api';

// Register panzoom extension
panzoom(cytoscape);

interface LayoutOption {
  value: string;
  label: string;
}

interface GraphFilters {
  status: DeviceStatus[];
  relationshipTypes: string[];
  datacenterId: string;
  search: string;
}

interface GraphData {
  devices: Device[];
  relationships: DeviceRelationship[];
  datacenters: Datacenter[];
  loading: boolean;
  error: string;
  cy: cytoscape.Core | null;
  filters: GraphFilters;
  layouts: LayoutOption[];
  selectedLayout: string;
  showFilters: boolean;
  hoveredNode: Device | null;

  init(): Promise<void>;
  loadData(): Promise<void>;
  get filteredDevices(): Device[];
  get filteredRelationships(): DeviceRelationship[];
  renderGraph(): void;
  applyLayout(): void;
  resetFilters(): void;
  zoomIn(): void;
  zoomOut(): void;
  fitGraph(): void;
  resetView(): void;
  exportPNG(): void;
  exportSVG(): void;
  exportJSON(): void;
  downloadDataUrl(dataUrl: string, filename: string): void;
  getStatusColor(status: DeviceStatus): string;
  showNodeTooltip(device: Device): void;
  hideNodeTooltip(): void;
  destroy(): void;
}

export function deviceGraph(): GraphData {
  return {
    devices: [],
    relationships: [],
    datacenters: [],
    loading: true,
    error: '',
    cy: null,
    filters: {
      status: [],
      relationshipTypes: [],
      datacenterId: '',
      search: ''
    },
    layouts: [
      { value: 'cose', label: 'Force-directed' },
      { value: 'circle', label: 'Circle' },
      { value: 'concentric', label: 'Concentric' },
      { value: 'grid', label: 'Grid' },
      { value: 'breadthfirst', label: 'Hierarchy' }
    ],
    selectedLayout: 'cose',
    showFilters: false,
    hoveredNode: null,

    async init(): Promise<void> {
      await this.loadData();
      // Wait for DOM to be ready
      setTimeout(() => this.renderGraph(), 100);
    },

    async loadData(): Promise<void> {
      this.loading = true;
      try {
        const [devices, relationships, datacenters] = await Promise.all([
          api.listDevices({}),
          api.getAllRelationships(),
          api.listDatacenters()
        ]);

        this.devices = devices || [];
        this.relationships = relationships || [];
        this.datacenters = datacenters || [];
      } catch (e) {
        console.error('Graph load error:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load data';
      } finally {
        this.loading = false;
      }
    },

    get filteredDevices(): Device[] {
      let filtered = this.devices;

      // Filter by status
      if (this.filters.status.length > 0) {
        filtered = filtered.filter(d => this.filters.status.includes(d.status));
      }

      // Filter by datacenter
      if (this.filters.datacenterId) {
        filtered = filtered.filter(d => d.datacenter_id === this.filters.datacenterId);
      }

      // Filter by search
      if (this.filters.search) {
        const search = this.filters.search.toLowerCase();
        filtered = filtered.filter(d =>
          d.name.toLowerCase().includes(search) ||
          (d.hostname && d.hostname.toLowerCase().includes(search))
        );
      }

      return filtered;
    },

    get filteredRelationships(): DeviceRelationship[] {
      let filtered = this.relationships;

      // Filter by relationship type
      if (this.filters.relationshipTypes.length > 0) {
        filtered = filtered.filter(r => this.filters.relationshipTypes.includes(r.type));
      }

      // Only include relationships where both devices are in filtered set
      const deviceIds = new Set(this.filteredDevices.map(d => d.id));
      filtered = filtered.filter(r =>
        deviceIds.has(r.parent_id) && deviceIds.has(r.child_id)
      );

      return filtered;
    },

    renderGraph(): void {
      const container = document.getElementById('graph-container');
      if (!container) return;

      // Destroy existing graph
      if (this.cy) {
        this.cy.destroy();
      }

      // Prepare nodes
      const nodes = this.filteredDevices.map(d => ({
        data: {
          id: d.id,
          label: d.name,
          device: d,
          status: d.status
        }
      }));

      // Prepare edges
      const edges = this.filteredRelationships.map((r, i) => ({
        data: {
          id: `edge-${i}`,
          source: r.parent_id,
          target: r.child_id,
          type: r.type,
          notes: r.notes
        }
      }));

      this.cy = cytoscape({
        container,
        elements: { nodes, edges },
        style: [
          // Node styles by status
          {
            selector: 'node',
            style: {
              'label': 'data(label)',
              'color': '#fff',
              'text-valign': 'center',
              'text-halign': 'center',
              'font-size': '11px',
              'font-weight': 'bold',
              'width': '50px',
              'height': '50px',
              'shape': 'roundrectangle',
              'text-wrap': 'wrap',
              'text-max-width': '45px',
              'border-width': 2,
              'border-color': '#fff'
            }
          },
          {
            selector: 'node[status="active"]',
            style: {
              'background-color': '#10B981',
              'border-color': '#059669'
            }
          },
          {
            selector: 'node[status="planned"]',
            style: {
              'background-color': '#3B82F6',
              'border-color': '#2563EB'
            }
          },
          {
            selector: 'node[status="maintenance"]',
            style: {
              'background-color': '#F59E0B',
              'border-color': '#D97706'
            }
          },
          {
            selector: 'node[status="decommissioned"]',
            style: {
              'background-color': '#6B7280',
              'border-color': '#4B5563'
            }
          },
          // Edge styles
          {
            selector: 'edge',
            style: {
              'width': 2,
              'line-color': '#94A3B8',
              'target-arrow-color': '#94A3B8',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier',
              'arrow-scale': 0.8
            }
          },
          {
            selector: 'edge[type="contains"]',
            style: {
              'line-color': '#10B981',
              'target-arrow-color': '#10B981'
            }
          },
          {
            selector: 'edge[type="connected_to"]',
            style: {
              'line-color': '#3B82F6',
              'target-arrow-color': '#3B82F6'
            }
          },
          {
            selector: 'edge[type="depends_on"]',
            style: {
              'line-color': '#A855F7',
              'target-arrow-color': '#A855F7'
            }
          },
          // Hover state
          {
            selector: 'node:selected',
            style: {
              'border-width': 4,
              'border-color': '#F59E0B'
            }
          }
        ],
        layout: {
          name: this.selectedLayout,
          animate: false,  // Disable animation on initial render to avoid timing issues
          // COSE-specific options
          nodeRepulsion: () => 8000,
          idealEdgeLength: () => 100,
          // Circle/concentric options
          radius: 200,
          startAngle: 0,
          // Grid options
          spacingFactor: 1.5
        },
        minZoom: 0.2,
        maxZoom: 3,
        wheelSensitivity: 0.3
      });

      // Fit graph to viewport
      this.cy.fit(undefined, 50);

      // Handle node clicks
      this.cy.on('tap', 'node', (evt) => {
        const deviceId = evt.target.id();
        window.dispatchEvent(new CustomEvent('nav', { detail: `/devices/detail?id=${deviceId}` }));
      });

      // Handle node hover for tooltips
      this.cy.on('mouseover', 'node', (evt) => {
        const device = evt.target.data('device') as Device;
        this.showNodeTooltip(device);
        container.style.cursor = 'pointer';
      });

      this.cy.on('mouseout', 'node', () => {
        this.hideNodeTooltip();
        container.style.cursor = 'default';
      });
    },

    applyLayout(): void {
      if (!this.cy) return;

      const layoutOptions: Record<string, object> = {
        cose: {
          name: 'cose',
          animate: true,
          animationDuration: 500,
          nodeRepulsion: () => 8000,
          idealEdgeLength: () => 100
        },
        circle: {
          name: 'circle',
          animate: true,
          animationDuration: 500,
          radius: 250
        },
        concentric: {
          name: 'concentric',
          animate: true,
          animationDuration: 500,
          minNodeSpacing: 30
        },
        grid: {
          name: 'grid',
          animate: true,
          animationDuration: 500,
          spacingFactor: 1.5
        },
        breadthfirst: {
          name: 'breadthfirst',
          animate: true,
          animationDuration: 500,
          directed: true,
          spacingFactor: 1.5
        }
      };

      this.cy.layout(layoutOptions[this.selectedLayout] as any || layoutOptions.cose as any).run();
    },

    resetFilters(): void {
      this.filters = {
        status: [],
        relationshipTypes: [],
        datacenterId: '',
        search: ''
      };
      this.renderGraph();
    },

    zoomIn(): void {
      if (this.cy) {
        this.cy.zoom(this.cy.zoom() * 1.2);
      }
    },

    zoomOut(): void {
      if (this.cy) {
        this.cy.zoom(this.cy.zoom() / 1.2);
      }
    },

    fitGraph(): void {
      if (this.cy) {
        this.cy.fit(undefined, 50);
      }
    },

    resetView(): void {
      if (this.cy) {
        this.cy.fit(undefined, 50);
        this.cy.zoom(1);
      }
    },

    exportPNG(): void {
      if (!this.cy) return;
      const png = this.cy.png({ full: true, scale: 2, bg: '#ffffff' });
      this.downloadDataUrl(png, 'topology.png');
    },

    exportSVG(): void {
      if (!this.cy) return;
      // Cytoscape doesn't have built-in SVG export, use PNG instead
      const png = this.cy.png({ full: true, scale: 2, bg: '#ffffff' });
      this.downloadDataUrl(png, 'topology.png');
    },

    exportJSON(): void {
      if (!this.cy) return;
      const data = {
        nodes: this.filteredDevices.map(d => ({
          id: d.id,
          name: d.name,
          hostname: d.hostname,
          status: d.status,
          datacenter_id: d.datacenter_id
        })),
        edges: this.filteredRelationships.map(r => ({
          source: r.parent_id,
          target: r.child_id,
          type: r.type,
          notes: r.notes
        }))
      };
      const json = 'data:application/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(data, null, 2));
      this.downloadDataUrl(json, 'topology.json');
    },

    downloadDataUrl(dataUrl: string, filename: string): void {
      const link = document.createElement('a');
      link.href = dataUrl;
      link.download = filename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
    },

    getStatusColor(status: DeviceStatus): string {
      const colors: Record<DeviceStatus, string> = {
        active: '#10B981',
        planned: '#3B82F6',
        maintenance: '#F59E0B',
        decommissioned: '#6B7280'
      };
      return colors[status] || '#6B7280';
    },

    showNodeTooltip(device: Device): void {
      this.hoveredNode = device;
    },

    hideNodeTooltip(): void {
      this.hoveredNode = null;
    },

    // Cleanup on component destroy
    destroy(): void {
      if (this.cy) {
        this.cy.destroy();
        this.cy = null;
      }
    }
  };
}
