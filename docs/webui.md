# Web UI Documentation

The Rackd Web UI provides a modern, responsive interface for managing devices, networks, IP addresses, and datacenter resources. Built with Alpine.js and TailwindCSS, it offers a complete single-page application experience.

## UI Overview

The web interface features:
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Dark/Light Theme**: Automatic system preference detection with manual toggle
- **Accessibility**: WCAG AAA compliant with keyboard navigation and screen reader support
- **Real-time Updates**: Live data updates without page refreshes
- **Global Search**: Search across devices, networks, and datacenters

## Navigation

### Sidebar Navigation
The left sidebar provides access to all main sections:
- **Dashboard** - Overview statistics and quick access
- **Devices** - Device inventory management
- **Networks** - Network and subnet management  
- **Datacenters** - Physical location tracking
- **Discovery** - Network scanning and device discovery

### Top Bar
- **Mobile Menu Toggle** - Hamburger menu for mobile devices
- **Global Search** - Search across all resources
- **Theme Toggle** - Switch between light, dark, and system themes

## Core Pages

### Dashboard (`/`)
Overview page displaying key statistics:
- Total device count with link to devices page
- Network count with link to networks page
- Datacenter count with link to datacenters page
- Discovered devices count with link to discovery page

### Devices (`/devices`)
Device inventory management:
- **List View**: Paginated table of all devices
- **Search & Filter**: Text search and datacenter filtering
- **Add Device**: Modal form for creating new devices
- **Device Details**: Detailed view with relationships and addresses

**Device Detail Page** (`/devices/detail?id=<id>`):
- Device information and specifications
- Network addresses and assignments
- Device relationships (dependencies, connections)
- Edit and delete actions

### Networks (`/networks`)
Network and subnet management:
- **List View**: All networks with subnet and VLAN information
- **Search & Filter**: Text search and datacenter filtering
- **Add Network**: Modal form for creating networks
- **Network Details**: IP pool management and device assignments

**Network Detail Page** (`/networks/detail?id=<id>`):
- Network configuration and settings
- IP address pool management
- Connected devices
- Subnet utilization statistics

### Datacenters (`/datacenters`)
Physical location management:
- **List View**: All datacenter locations
- **Add Datacenter**: Modal form for new locations
- **Datacenter Details**: Associated devices and networks

**Datacenter Detail Page** (`/datacenters/detail?id=<id>`):
- Location information and contact details
- Associated devices and networks
- Resource utilization overview

### Discovery (`/discovery`)
Network scanning and device discovery:
- **Discovered Devices**: List of found but unmanaged devices
- **Scan Controls**: Start new network scans
- **Promote Devices**: Convert discovered devices to managed devices
- **Scan History**: Previous scan results and status

## Internal Pages

### Pools (`/pools`)
IP address pool management (when available):
- Pool configuration and IP ranges
- Address allocation and availability
- Pool utilization statistics

### Credentials (`/credentials`)
Authentication credential management:
- SNMP community strings
- SSH/Telnet credentials
- Credential assignment to devices

### Scan Profiles (`/scan-profiles`)
Network scan configuration:
- Scan type definitions (Quick, Full, Deep)
- Port ranges and protocols
- Timeout and retry settings

### Scheduled Scans (`/scheduled-scans`)
Automated scanning configuration:
- Recurring scan schedules
- Target network selection
- Scan profile assignment

## Key Features

### Global Search
- **Scope**: Searches across devices, networks, and datacenters
- **Real-time**: Results appear as you type
- **Navigation**: Click results to navigate directly to details
- **Keyboard**: Full keyboard navigation support

### Theme Management
Three theme modes:
- **Light**: High contrast light theme
- **Dark**: High contrast dark theme  
- **System**: Follows OS preference automatically

### Responsive Design
- **Mobile-first**: Optimized for mobile devices
- **Breakpoints**: Responsive layouts for all screen sizes
- **Touch-friendly**: Minimum 44px touch targets
- **Accessible**: Screen reader and keyboard navigation

### Modal Forms
Consistent modal interface for:
- Adding/editing devices, networks, datacenters
- Confirmation dialogs for destructive actions
- Scan configuration and promotion workflows

## Alpine.js Components

### Core Components

**Router** (`router`):
- Single-page application routing
- Browser history management
- Sidebar state management

**Navigation** (`nav`):
- Dynamic navigation items
- Feature detection

**Global Search** (`globalSearch`):
- Debounced search input
- Multi-resource search results
- Keyboard navigation

**Theme Toggle** (`themeToggle`):
- Theme persistence
- System preference detection
- CSS class management

### Resource Components

**Device List** (`deviceList`):
- Paginated device listing
- Search and filtering
- Bulk operations
- Modal management

**Device Detail** (`deviceDetail`):
- Device information display
- Address management
- Relationship tracking
- Edit capabilities

**Device Form** (`deviceForm`):
- Add/edit device modal
- Form validation
- Datacenter selection
- Address assignment

**Network List** (`networkList`):
- Network listing and search
- Datacenter filtering
- Pool integration

**Network Detail** (`networkDetail`):
- Network configuration
- IP pool management
- Device assignments
- Utilization tracking

**Discovery List** (`discoveryList`):
- Discovered device listing
- Scan management
- Device promotion
- Status tracking

### Additional Components

**Credentials** (`credentialsList`, `credentialForm`):
- Credential management
- Type-specific forms
- Device assignment

**Profiles** (`profileList`, `profileForm`):
- Scan profile configuration
- Parameter management
- Template system

**Scheduled Scans** (`scheduledScansList`, `scheduledScanForm`):
- Schedule management
- Cron expression handling
- Profile assignment

## Accessibility Features

### WCAG AAA Compliance
- **Color Contrast**: 7:1 minimum contrast ratios
- **Focus Indicators**: Visible focus rings on all interactive elements
- **Keyboard Navigation**: Full keyboard accessibility
- **Screen Readers**: Proper ARIA labels and roles

### Keyboard Navigation
- **Tab Order**: Logical tab sequence
- **Skip Links**: Skip to main content
- **Modal Traps**: Focus management in modals
- **Escape Handling**: Close modals with Escape key

### Touch Accessibility
- **Target Size**: Minimum 44px touch targets
- **Spacing**: Adequate spacing between interactive elements
- **Gestures**: No complex gestures required

## Browser Support

### Supported Browsers
- **Chrome/Chromium**: 90+
- **Firefox**: 88+
- **Safari**: 14+
- **Edge**: 90+

### Required Features
- ES2020 support
- CSS Custom Properties
- CSS Grid and Flexbox
- Fetch API
- History API

## Performance

### Optimization Features
- **Lazy Loading**: Components loaded on demand
- **Debounced Search**: Reduced API calls
- **Pagination**: Large datasets split into pages
- **Caching**: API response caching
- **Minimal Bundle**: No external dependencies

### Loading States
- **Skeleton Loading**: Placeholder content during loads
- **Progress Indicators**: Visual feedback for operations
- **Error Handling**: Graceful error display and recovery

## Customization

### CSS Variables
The UI uses CSS custom properties for theming:
```css
:root {
  --core-blue: #3B82F6;
  --accent-cyan: #06B6D4;
  --text-primary: #111827;
  /* ... */
}
```

### Component Registration
Add custom Alpine.js components:
```javascript
Alpine.data('customComponent', () => ({
  // Component logic
}));
```

## Development

### File Structure
```
webui/src/
├── index.html          # Main HTML template
├── app.ts             # Application entry point
├── styles.css         # Global styles and theme
├── components/        # Alpine.js components
│   ├── devices.ts     # Device management
│   ├── networks.ts    # Network management
│   ├── search.ts      # Global search
│   └── ...
├── partials/          # HTML templates
│   ├── pages/         # Page templates
│   └── modals/        # Modal templates
└── core/              # Core utilities
    ├── api.ts         # API client
    ├── types.ts       # TypeScript types
    └── utils.ts       # Utility functions
```

### Build Process
The UI is built using Bun and bundled into static assets:
```bash
cd webui
bun install
bun run build
```

### API Contract Validation
The shared API-facing TypeScript types are checked against [`api/openapi.yaml`](/Users/martinsuchenak/Devel/projects/rackd/api/openapi.yaml):
```bash
cd webui
bun run validate:types
```
This validator covers the frontend's central API contract types and is intended to catch schema drift before UI code starts relying on mismatched fields.

### Hot Reload
Development server with hot reload:
```bash
bun run dev
```
