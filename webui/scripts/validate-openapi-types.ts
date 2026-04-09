import { readFileSync } from 'fs';
import { resolve } from 'path';

type PropertyInfo = {
  name: string;
  required: boolean;
};

type InterfaceInfo = {
  name: string;
  properties: Map<string, PropertyInfo>;
};

type SchemaInfo = {
  name: string;
  properties: Map<string, PropertyInfo>;
};

type Mapping = {
  typeName: string;
  schemaName: string;
};

const mappings: Mapping[] = [
  { typeName: 'Datacenter', schemaName: 'Datacenter' },
  { typeName: 'Network', schemaName: 'Network' },
  { typeName: 'NetworkUtilization', schemaName: 'NetworkUtilization' },
  { typeName: 'ServiceInfo', schemaName: 'ServiceInfo' },
  { typeName: 'DiscoveredDevice', schemaName: 'DiscoveredDevice' },
  { typeName: 'DiscoveryScan', schemaName: 'DiscoveryScan' },
  { typeName: 'ScanProfile', schemaName: 'ScanProfile' },
  { typeName: 'ScheduledScan', schemaName: 'ScheduledScan' },
  { typeName: 'User', schemaName: 'User' },
  { typeName: 'Role', schemaName: 'Role' },
  { typeName: 'Permission', schemaName: 'Permission' },
  { typeName: 'Webhook', schemaName: 'Webhook' },
  { typeName: 'WebhookDelivery', schemaName: 'WebhookDelivery' },
  { typeName: 'CustomFieldDefinition', schemaName: 'CustomField' },
  { typeName: 'Circuit', schemaName: 'Circuit' },
  { typeName: 'NATMapping', schemaName: 'NATRule' },
  { typeName: 'DNSProvider', schemaName: 'DNSProvider' },
  { typeName: 'DNSZone', schemaName: 'DNSZone' },
  { typeName: 'DNSRecord', schemaName: 'DNSRecord' },
  { typeName: 'APIKey', schemaName: 'APIKey' },
];

function parseInterfaces(source: string): Map<string, InterfaceInfo> {
  const interfaces = new Map<string, InterfaceInfo>();
  const lines = source.split('\n');
  let current: InterfaceInfo | null = null;

  for (const line of lines) {
    const startMatch = line.match(/^export interface (\w+)(?: extends [^{]+)? \{$/);
    if (startMatch) {
      current = {
        name: startMatch[1],
        properties: new Map<string, PropertyInfo>(),
      };
      interfaces.set(current.name, current);
      continue;
    }

    if (current && line.trim() === '}') {
      current = null;
      continue;
    }

    if (!current) {
      continue;
    }

    const propertyMatch = line.match(/^\s{2}([A-Za-z0-9_]+)(\??):/);
    if (!propertyMatch) {
      continue;
    }

    current.properties.set(propertyMatch[1], {
      name: propertyMatch[1],
      required: propertyMatch[2] !== '?',
    });
  }

  return interfaces;
}

function parseInlineRequiredList(raw: string): string[] {
  return raw
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean);
}

function parseSchemas(source: string): Map<string, SchemaInfo> {
  const schemas = new Map<string, SchemaInfo>();
  const lines = source.split('\n');
  let current: SchemaInfo | null = null;
  let inProperties = false;
  let inRequiredBlock = false;

  for (const line of lines) {
    const schemaMatch = line.match(/^ {4}([A-Z][A-Za-z0-9]+):\s*$/);
    if (schemaMatch) {
      current = {
        name: schemaMatch[1],
        properties: new Map<string, PropertyInfo>(),
      };
      schemas.set(current.name, current);
      inProperties = false;
      inRequiredBlock = false;
      continue;
    }

    if (!current) {
      continue;
    }

    if (/^ {4}[A-Z][A-Za-z0-9]+:\s*$/.test(line)) {
      inProperties = false;
      inRequiredBlock = false;
      continue;
    }

    if (/^ {6}required:\s*\[(.*)\]\s*$/.test(line)) {
      const match = line.match(/^ {6}required:\s*\[(.*)\]\s*$/);
      if (match) {
        for (const property of parseInlineRequiredList(match[1])) {
          const existing = current.properties.get(property);
          current.properties.set(property, {
            name: property,
            required: true,
          });
          if (existing) {
            existing.required = true;
            current.properties.set(property, existing);
          }
        }
      }
      inRequiredBlock = false;
      continue;
    }

    if (/^ {6}required:\s*$/.test(line)) {
      inRequiredBlock = true;
      inProperties = false;
      continue;
    }

    if (inRequiredBlock) {
      const requiredMatch = line.match(/^ {8}- ([A-Za-z0-9_]+)\s*$/);
      if (requiredMatch) {
        const property = requiredMatch[1];
        const existing = current.properties.get(property);
        current.properties.set(property, {
          name: property,
          required: true,
        });
        if (existing) {
          existing.required = true;
          current.properties.set(property, existing);
        }
        continue;
      }
      if (!/^ {8}/.test(line)) {
        inRequiredBlock = false;
      }
    }

    if (/^ {6}properties:\s*$/.test(line)) {
      inProperties = true;
      continue;
    }

    if (inProperties) {
      const propertyMatch = line.match(/^ {8}([A-Za-z0-9_]+):/);
      if (propertyMatch) {
        const property = propertyMatch[1];
        const existing = current.properties.get(property);
        current.properties.set(property, {
          name: property,
          required: existing?.required ?? false,
        });
        continue;
      }

      if (!/^ {8}/.test(line) && line.trim() !== '') {
        inProperties = false;
      }
    }
  }

  return schemas;
}

function formatList(values: string[]): string {
  return values.length === 0 ? 'none' : values.join(', ');
}

const repoRoot = resolve(import.meta.dir, '..', '..');
const typesPath = resolve(repoRoot, 'webui/src/core/types.ts');
const openAPIPath = resolve(repoRoot, 'api/openapi.yaml');

const interfaceMap = parseInterfaces(readFileSync(typesPath, 'utf8'));
const schemaMap = parseSchemas(readFileSync(openAPIPath, 'utf8'));

const errors: string[] = [];

for (const mapping of mappings) {
  const iface = interfaceMap.get(mapping.typeName);
  const schema = schemaMap.get(mapping.schemaName);

  if (!iface) {
    errors.push(`Missing TypeScript interface: ${mapping.typeName}`);
    continue;
  }

  if (!schema) {
    errors.push(`Missing OpenAPI schema: ${mapping.schemaName}`);
    continue;
  }

  const interfaceProperties = Array.from(iface.properties.keys()).sort();
  const schemaProperties = Array.from(schema.properties.keys()).sort();

  const interfaceRequired = Array.from(iface.properties.values())
    .filter((property) => property.required)
    .map((property) => property.name)
    .sort();
  const schemaRequired = Array.from(schema.properties.values())
    .filter((property) => property.required)
    .map((property) => property.name)
    .sort();

  const missingFromSchema = interfaceProperties.filter((property) => !schema.properties.has(property));
  const missingFromInterface = schemaProperties.filter((property) => !iface.properties.has(property));
  const requiredMissingFromSchema = interfaceRequired.filter((property) => !schemaRequired.includes(property));
  const requiredMissingFromInterface = schemaRequired.filter((property) => !interfaceRequired.includes(property));

  if (
    missingFromSchema.length > 0 ||
    missingFromInterface.length > 0 ||
    requiredMissingFromSchema.length > 0 ||
    requiredMissingFromInterface.length > 0
  ) {
    errors.push(
      [
        `Contract mismatch for ${mapping.typeName} <-> ${mapping.schemaName}`,
        `  missing from OpenAPI: ${formatList(missingFromSchema)}`,
        `  missing from TypeScript: ${formatList(missingFromInterface)}`,
        `  required in TypeScript only: ${formatList(requiredMissingFromSchema)}`,
        `  required in OpenAPI only: ${formatList(requiredMissingFromInterface)}`,
      ].join('\n'),
    );
  }
}

if (errors.length > 0) {
  console.error('OpenAPI/frontend type validation failed.\n');
  for (const error of errors) {
    console.error(error);
    console.error('');
  }
  process.exit(1);
}

console.log(`Validated ${mappings.length} OpenAPI/type mappings successfully.`);
