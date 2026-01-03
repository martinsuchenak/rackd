import Alpine from 'alpinejs';

Alpine.data('datacenterManager', () => ({
    datacenters: [],
    loading: false,
    saving: false,
    showModal: false,
    modalTitle: 'Add Datacenter',
    currentDatacenter: {},
    form: {
        id: '',
        name: '',
        location: '',
        description: ''
    },
    toast: { show: false, message: '', type: 'info' },

    async init() {
        await this.loadDatacenters();
    },

    async loadDatacenters() {
        this.loading = true;
        try {
            const response = await fetch('/api/datacenters');
            if (!response.ok) throw new Error('Failed to load datacenters');
            this.datacenters = await response.json();
        } catch (error) {
            this.showToast('Failed to load datacenters', 'error');
        } finally {
            this.loading = false;
        }
    },

    openAddModal() {
        this.modalTitle = 'Add Datacenter';
        this.resetForm();
        this.showModal = true;
    },

    closeModal() {
        this.showModal = false;
        this.resetForm();
    },

    resetForm() {
        this.form = {
            id: '',
            name: '',
            location: '',
            description: ''
        };
    },

    async saveDatacenter() {
        this.saving = true;
        try {
            const datacenter = {
                name: this.form.name,
                location: this.form.location || '',
                description: this.form.description || ''
            };

            const url = this.form.id
                ? `/api/datacenters/${this.form.id}`
                : '/api/datacenters';
            const method = this.form.id ? 'PUT' : 'POST';

            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(datacenter)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to save datacenter');
            }

            this.showToast(this.form.id ? 'Datacenter updated successfully' : 'Datacenter created successfully', 'success');
            this.closeModal();
            this.loadDatacenters();
            // Refresh device manager's datacenters list
            if (window.deviceManagerRefreshDatacenters) {
                window.deviceManagerRefreshDatacenters();
            }
        } catch (error) {
            this.showToast(error.message, 'error');
        } finally {
            this.saving = false;
        }
    },

    async editDatacenter(id) {
        try {
            const response = await fetch(`/api/datacenters/${id}`);
            if (!response.ok) throw new Error('Failed to load datacenter');
            const datacenter = await response.json();
            this.modalTitle = 'Edit Datacenter';
            this.form = {
                id: datacenter.id || '',
                name: datacenter.name || '',
                location: datacenter.location || '',
                description: datacenter.description || ''
            };
            this.showModal = true;
        } catch (error) {
            this.showToast('Failed to load datacenter', 'error');
        }
    },

    async deleteDatacenter(id) {
        const deviceCount = await this.getDatacenterDeviceCount(id);
        const message = deviceCount > 0
            ? `Are you sure you want to delete this datacenter? ${deviceCount} devices will lose their datacenter association.`
            : 'Are you sure you want to delete this datacenter?';

        if (!confirm(message)) return;

        try {
            const response = await fetch(`/api/datacenters/${id}`, {
                method: 'DELETE'
            });

            if (!response.ok) throw new Error('Failed to delete datacenter');

            this.showToast('Datacenter deleted successfully', 'success');
            this.loadDatacenters();
            // Refresh device manager's datacenters list
            if (window.deviceManagerRefreshDatacenters) {
                window.deviceManagerRefreshDatacenters();
            }
        } catch (error) {
            this.showToast('Failed to delete datacenter', 'error');
        }
    },

    async getDatacenterDeviceCount(datacenterId) {
        try {
            const response = await fetch(`/api/datacenters/${datacenterId}/devices`);
            if (!response.ok) return 0;
            const devices = await response.json();
            return devices.length;
        } catch {
            return 0;
        }
    },

    showToast(message, type = 'info') {
        this.toast = { show: true, message, type };
        setTimeout(() => { this.toast.show = false; }, 3000);
    }
}));

Alpine.data('deviceManager', () => ({
    devices: [],
    datacenters: [],
    loading: false,
    saving: false,
    showModal: false,
    showViewModal: false,
    searchQuery: '',
    modalTitle: 'Add Device',
    currentDevice: {},
    form: {
        id: '',
        name: '',
        description: '',
        make_model: '',
        os: '',
        datacenter_id: '',
        tagsInput: '',
        domainsInput: '',
        addresses: [{ ip: '', port: '', type: 'ipv4', label: '' }]
    },
    toast: { show: false, message: '', type: 'info' },

    async init() {
        await Promise.all([this.loadDevices(), this.loadDatacenters()]);
        // Register refresh function for datacenter manager
        window.deviceManagerRefreshDatacenters = () => this.loadDatacenters();
    },

    async loadDatacenters() {
        try {
            const response = await fetch('/api/datacenters');
            if (!response.ok) throw new Error('Failed to load datacenters');
            this.datacenters = await response.json();
        } catch (error) {
            console.error('Failed to load datacenters:', error);
        }
    },

    async loadDevices() {
        this.loading = true;
        try {
            const url = this.searchQuery
                ? `/api/search?q=${encodeURIComponent(this.searchQuery)}`
                : '/api/devices';
            const response = await fetch(url);
            if (!response.ok) throw new Error('Failed to load devices');
            this.devices = await response.json();
            // Enrich devices with datacenter names
            this.enrichDevicesWithDatacenters();
        } catch (error) {
            this.showToast('Failed to load devices', 'error');
        } finally {
            this.loading = false;
        }
    },

    enrichDevicesWithDatacenters() {
        this.devices = this.devices.map(device => ({
            ...device,
            datacenter_name: this.datacenters.find(dc => dc.id === device.datacenter_id)?.name || null
        }));
    },

    clearSearch() {
        this.searchQuery = '';
        this.loadDevices();
    },

    openAddModal() {
        this.modalTitle = 'Add Device';
        this.resetForm();
        this.showModal = true;
    },

    closeModal() {
        this.showModal = false;
        this.resetForm();
    },

    resetForm() {
        this.form = {
            id: '',
            name: '',
            description: '',
            make_model: '',
            os: '',
            datacenter_id: '',
            tagsInput: '',
            domainsInput: '',
            addresses: [{ ip: '', port: '', type: 'ipv4', label: '' }]
        };
    },

    addAddress() {
        this.form.addresses.push({ ip: '', port: '', type: 'ipv4', label: '' });
    },

    removeAddress(index) {
        this.form.addresses.splice(index, 1);
        if (this.form.addresses.length === 0) {
            this.addAddress();
        }
    },

    async saveDevice() {
        this.saving = true;
        try {
            // Clean up addresses - convert empty ports to null/omit them
            const addresses = this.form.addresses
                .filter(a => a.ip)
                .map(a => ({
                    ip: a.ip,
                    port: a.port && a.port !== '' ? parseInt(a.port, 10) : 0,
                    type: a.type || 'ipv4',
                    label: a.label || ''
                }));

            const device = {
                name: this.form.name,
                description: this.form.description || '',
                make_model: this.form.make_model || '',
                os: this.form.os || '',
                datacenter_id: this.form.datacenter_id || '',
                tags: this.form.tagsInput.split(',').map(t => t.trim()).filter(t => t),
                domains: this.form.domainsInput.split(',').map(t => t.trim()).filter(t => t),
                addresses: addresses
            };

            const url = this.form.id ? `/api/devices/${this.form.id}` : '/api/devices';
            const method = this.form.id ? 'PUT' : 'POST';

            const response = await fetch(url, {
                method,
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(device)
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to save device');
            }

            this.showToast(this.form.id ? 'Device updated successfully' : 'Device created successfully', 'success');
            this.closeModal();
            this.loadDevices();
        } catch (error) {
            this.showToast(error.message, 'error');
        } finally {
            this.saving = false;
        }
    },

    async viewDevice(id) {
        try {
            const response = await fetch(`/api/devices/${id}`);
            if (!response.ok) throw new Error('Failed to load device');
            const device = await response.json();
            // Enrich with datacenter name
            device.datacenter_name = this.datacenters.find(dc => dc.id === device.datacenter_id)?.name || null;
            this.currentDevice = device;
            this.showViewModal = true;
        } catch (error) {
            this.showToast('Failed to load device', 'error');
        }
    },

    closeViewModal() {
        this.showViewModal = false;
        this.currentDevice = {};
    },

    editCurrentDevice() {
        const device = this.currentDevice;
        this.modalTitle = 'Edit Device';
        this.form = {
            id: device.id || '',
            name: device.name || '',
            description: device.description || '',
            make_model: device.make_model || '',
            os: device.os || '',
            datacenter_id: device.datacenter_id || '',
            tagsInput: (device.tags || []).join(', '),
            domainsInput: (device.domains || []).join(', '),
            addresses: device.addresses && device.addresses.length > 0
                ? [...device.addresses]
                : [{ ip: '', port: '', type: 'ipv4', label: '' }]
        };
        this.closeViewModal();
        this.showModal = true;
    },

    async editDevice(id) {
        try {
            const response = await fetch(`/api/devices/${id}`);
            if (!response.ok) throw new Error('Failed to load device');
            const device = await response.json();
            this.modalTitle = 'Edit Device';
            this.form = {
                id: device.id || '',
                name: device.name || '',
                description: device.description || '',
                make_model: device.make_model || '',
                os: device.os || '',
                datacenter_id: device.datacenter_id || '',
                tagsInput: (device.tags || []).join(', '),
                domainsInput: (device.domains || []).join(', '),
                addresses: device.addresses && device.addresses.length > 0
                    ? [...device.addresses]
                    : [{ ip: '', port: '', type: 'ipv4', label: '' }]
            };
            this.showModal = true;
        } catch (error) {
            this.showToast('Failed to load device', 'error');
        }
    },

    async deleteDeviceFromList(id) {
        if (!confirm('Are you sure you want to delete this device?')) return;

        try {
            const response = await fetch(`/api/devices/${id}`, {
                method: 'DELETE'
            });

            if (!response.ok) throw new Error('Failed to delete device');

            this.showToast('Device deleted successfully', 'success');
            this.loadDevices();
        } catch (error) {
            this.showToast('Failed to delete device', 'error');
        }
    },

    async deleteCurrentDevice() {
        if (!confirm('Are you sure you want to delete this device?')) return;

        try {
            const response = await fetch(`/api/devices/${this.currentDevice.id}`, {
                method: 'DELETE'
            });

            if (!response.ok) throw new Error('Failed to delete device');

            this.showToast('Device deleted successfully', 'success');
            this.closeViewModal();
            this.loadDevices();
        } catch (error) {
            this.showToast('Failed to delete device', 'error');
        }
    },

    showToast(message, type = 'info') {
        this.toast = { show: true, message, type };
        setTimeout(() => { this.toast.show = false; }, 3000);
    }
}));

Alpine.start();
