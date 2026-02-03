// Device Relationship Graph Visualization

import cytoscape from 'cytoscape';
import type { Device, DeviceRelationship } from '../core/types';
import { api, RackdAPIError } from '../core/api';

interface GraphData {
  devices: Device[];
  relationships: DeviceRelationship[];
  loading: boolean;
  error: string;
  cy: cytoscape.Core | null;
  init(): Promise<void>;
  loadData(): Promise<void>;
  renderGraph(): void;
}

export function deviceGraph(): GraphData {
  return {
    devices: [],
    relationships: [],
    loading: true,
    error: '',
    cy: null,

    async init(): Promise<void> {
      await this.loadData();
      // Wait for DOM to be ready
      setTimeout(() => this.renderGraph(), 100);
    },

    async loadData(): Promise<void> {
      this.loading = true;
      try {
        const [devices, relationships] = await Promise.all([
          api.listDevices({}),
          api.getAllRelationships()
        ]);
        
        this.devices = devices;
        this.relationships = relationships || [];
      } catch (e) {
        console.error('Graph load error:', e);
        this.error = e instanceof RackdAPIError ? e.message : 'Failed to load data';
      } finally {
        this.loading = false;
      }
    },

    renderGraph(): void {
      const container = document.getElementById('graph-container');
      if (!container) return;

      // Prepare nodes
      const nodes = this.devices.map(d => ({
        data: { id: d.id, label: d.name, device: d }
      }));

      // Prepare edges
      const edges = this.relationships.map((r, i) => ({
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
          {
            selector: 'node',
            style: {
              'background-color': '#3B82F6',
              'label': 'data(label)',
              'color': '#fff',
              'text-valign': 'center',
              'text-halign': 'center',
              'font-size': '12px',
              'width': '60px',
              'height': '60px'
            }
          },
          {
            selector: 'edge',
            style: {
              'width': 2,
              'line-color': '#94A3B8',
              'target-arrow-color': '#94A3B8',
              'target-arrow-shape': 'triangle',
              'curve-style': 'bezier'
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
          }
        ],
        layout: {
          name: 'cose',
          animate: false,
          nodeRepulsion: 8000,
          idealEdgeLength: 100
        }
      });

      // Handle node clicks
      this.cy.on('tap', 'node', (evt) => {
        const deviceId = evt.target.id();
        window.dispatchEvent(new CustomEvent('nav', { detail: `/devices/detail?id=${deviceId}` }));
      });
    }
  };
}
