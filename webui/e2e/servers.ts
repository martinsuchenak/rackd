// Rackd with the example extension in examples/hello, for the extension UI
// tests. playwright.config.ts starts it next to the main server.
export const extensionBaseURL = process.env.E2E_EXTENSION_BASE_URL || 'http://127.0.0.1:18081';
