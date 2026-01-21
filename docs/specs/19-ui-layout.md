# UI Layout and Design

This document outlines the user interface layout, design philosophy, and key components for the Rackd Web UI, based on the architecture described in `08-web-ui.md`.

## 1. Design Philosophy

The UI will adhere to the following core principles:

- **Responsive**: The layout must adapt seamlessly to various screen sizes, from mobile devices to large desktop monitors.
- **Modular**: The design will be built using a system of reusable components and a grid-based layout for consistency and maintainability.
- **Accessible**: The interface will comply with **WCAG 2.1 AA** standards to ensure it is usable by people with a wide range of disabilities.
- **Themable**: Both **light and dark themes** will be supported, with an option to automatically follow the user's system preference.

## 2. Layout Structure

The main application interface will consist of a two-column layout:

1.  **Persistent Sidebar (Navigation)**: A fixed navigation bar on the left side of the screen providing access to all major sections of the application.
    - On larger viewports (desktops, tablets), it will be persistently visible.
    - On smaller viewports (mobile), it will collapse into a "hamburger" menu to maximize content visibility.
2.  **Main Content Area**: The primary area on the right where all content, data tables, forms, and visualizations are displayed. This area will use a flexible grid system to arrange components.

### Text Wireframe: Main Layout

```
+-----------------------------------------------------------------------------+
| Top Bar (Global Search, User Menu, Theme Toggle)                            |
+--------------------------------+--------------------------------------------+
|                                |                                            |
|  Sidebar Navigation            |  Main Content Area                         |
|                                |                                            |
|  - Dashboard                   |  +--------------------------------------+  |
|  - Devices                     |  | Page Title                           |  |
|  - Networks                    |  +--------------------------------------+  |
|  - Datacenters                 |                                            |
|  - Discovery                   |  (Content for the selected page,         |
|  - Settings                    |   e.g., a data table or a form,          |
|                                |   is displayed here)                     |
|                                |                                            |
|                                |                                            |
+--------------------------------+--------------------------------------------+
```

## 3. Theming

- **Mechanism**: Theming will be implemented using CSS variables and TailwindCSS's dark mode variant (`dark:`). A class (e.g., `dark`) will be toggled on the `<html>` element to switch between themes.
- **Theme Toggle**: A user-facing control will be provided (e.g., in the top bar) with three options:
    - Light
    - Dark
    - System (default)
- **Persistence**: The user's theme preference will be saved in `localStorage`.

## 4. Accessibility (WCAG 2.1 AA)

Accessibility is a primary requirement. The following practices must be implemented:

- **Semantic HTML**: Use HTML5 elements (`<nav>`, `<main>`, `<header>`, `<section>`, etc.) to define the structure of the page.
- **ARIA Attributes**: Use ARIA (Accessible Rich Internet Applications) attributes where necessary to provide additional context for screen readers, especially for dynamic components.
- **Keyboard Navigation**: All interactive elements (links, buttons, form fields, menus) must be reachable and operable using the Tab key. Focus states (`:focus-visible`) must be clearly visible.
- **Color Contrast**: Text and background colors must meet a contrast ratio of at least 4.5:1 (or 3:1 for large text).
- **Forms**: All form inputs must be associated with a `<label>`. Error messages must be programmatically associated with their respective inputs.
- **Images & Icons**: All non-decorative images and icons must have alternative text (`alt` attribute).

## 5. Page Wireframes

### Device List Page

```
+-----------------------------------------------------------------------------+
| Top Bar: [Global Search...] [User Menu] [Theme]                             |
+--------------------------------+--------------------------------------------+
| Sidebar Nav (Active: Devices)  |  Devices                                  |
|                                |  +--------------------------------------+  |
|                                |  | [Filter] [Filter]   [Add New Device] |  |
|                                |  +--------------------------------------+  |
|                                |  |                                      |  |
|                                |  |  +----------------------------------+  |  |
|                                |  |  | Name | IP Address | Model | Status|  |  |
|                                |  |  +----------------------------------+  |  |
|                                |  |  | srv-01 | 10.1.1.5 | R740  | Online|  |  |
|                                |  |  | srv-02 | 10.1.1.6 | R740  | Online|  |  |
|                                |  |  | ...    | ...      | ...   | ...   |  |  |
|                                |  |  +----------------------------------+  |  |
|                                |  |  | [Pagination: < 1 2 3 >]          |  |  |
|                                |  +--------------------------------------+  |  |
+--------------------------------+--------------------------------------------+
```

### Device Detail Page

```
+-----------------------------------------------------------------------------+
| Top Bar: [Global Search...] [User Menu] [Theme]                             |
+--------------------------------+--------------------------------------------+
| Sidebar Nav (Active: Devices)  |  < Devices / srv-01                       |
|                                |  +--------------------------------------+  |
|                                |  | srv-01              [Edit] [Delete]  |  |
|                                |  | Status: Online                         |  |
|                                |  +--------------------------------------+  |
|                                |                                            |
|                                |  [Details] [Addresses] [Relationships]     |
|                                |  +--------------------------------------+  |
|                                |  | General Information                  |  |
|                                |  |  Make/Model: Dell R740               |  |
|                                |  |  OS:         Ubuntu 22.04            |  |
|                                |  |  Datacenter: DC-West-01              |  |
|                                |  |  Tags:       [web] [prod]            |  |
|                                |  +--------------------------------------+  |
+--------------------------------+--------------------------------------------+
```

## 6. Component Inventory

The UI will be composed of the following key reusable components:

- **Data Table**: A responsive table for displaying lists of entities. Supports sorting, filtering, and pagination.
- **Forms**: Standardized forms for creating and editing entities, with built-in validation.
- **Modals**: For confirmations (e.g., delete actions) and displaying focused information.
- **Search Bar**: A global search component in the top bar.
- **Filter Controls**: Dropdowns, text inputs, and toggles for filtering data tables.
- **Navigation Sidebar**: The primary navigation component.
- **Tabs**: For organizing content on detail pages (e.g., Device Details, Addresses, etc.).
- **Buttons**: A consistent set of buttons for primary, secondary, and destructive actions.
- **Badges/Tags**: For displaying status labels and tags.
- **Alerts/Toasts**: For providing user feedback (e.g., "Device saved successfully").
