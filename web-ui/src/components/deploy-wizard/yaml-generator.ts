import { WizardData } from './types';

/**
 * Generates a Score v1b1 YAML specification from wizard data
 */
export function generateScoreYaml(data: WizardData): string {
  // Validate required fields
  const trimmedAppName = data.appName.trim();
  if (!trimmedAppName) {
    throw new Error('Application name is required and cannot be empty');
  }

  // Validate DNS-compliant name
  const namePattern = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
  if (!namePattern.test(trimmedAppName)) {
    throw new Error('Application name must contain only lowercase letters, numbers, and hyphens');
  }

  if (!data.container.image.trim()) {
    throw new Error('Container image is required and cannot be empty');
  }

  const lines: string[] = [];

  // API version and metadata
  lines.push('apiVersion: score.dev/v1b1');
  lines.push('metadata:');
  lines.push(`  name: ${trimmedAppName}`);
  lines.push('  labels:');
  lines.push(`    environment: ${data.environment}`);
  lines.push(`    ttl: ${data.ttl}`);
  lines.push('');

  // Containers
  lines.push('containers:');
  lines.push('  main:');
  lines.push(`    image: ${data.container.image}`);

  // Environment variables
  if (Object.keys(data.container.envVars).length > 0) {
    lines.push('    variables:');
    Object.entries(data.container.envVars).forEach(([key, value]) => {
      // Escape quotes in values
      const escapedValue = value.replace(/"/g, '\\"');
      lines.push(`      ${key}: "${escapedValue}"`);
    });
  }

  // Resources (CPU/Memory)
  if (data.container.cpuRequest || data.container.memoryRequest) {
    lines.push('    resources:');
    lines.push('      requests:');
    if (data.container.cpuRequest) {
      lines.push(`        cpu: ${data.container.cpuRequest}`);
    }
    if (data.container.memoryRequest) {
      lines.push(`        memory: ${data.container.memoryRequest}`);
    }
  }

  lines.push('');

  // Service (expose container port)
  lines.push('service:');
  lines.push('  ports:');
  lines.push('    http:');
  lines.push(`      port: ${data.container.port}`);
  lines.push(`      targetPort: ${data.container.port}`);
  lines.push('');

  // Resources section
  if (data.resources.length > 0) {
    lines.push('resources:');
    data.resources.forEach((resource) => {
      lines.push(`  ${resource.name}:`);
      lines.push(`    type: ${resource.type}`);

      if (Object.keys(resource.properties).length > 0) {
        lines.push('    properties:');
        Object.entries(resource.properties).forEach(([key, value]) => {
          if (value !== undefined && value !== null && value !== '') {
            // Format value based on type
            const formattedValue = formatYamlValue(value);
            lines.push(`      ${key}: ${formattedValue}`);
          }
        });
      }
    });
  }

  return lines.join('\n');
}

/**
 * Format a value for YAML output
 */
function formatYamlValue(value: any): string {
  if (typeof value === 'string') {
    // Check if string contains special characters that need quoting
    if (value.includes(':') || value.includes('#') || value.includes('\n') || value.includes('"')) {
      return `"${value.replace(/"/g, '\\"')}"`;
    }
    return value;
  }

  if (typeof value === 'number') {
    return value.toString();
  }

  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }

  if (Array.isArray(value)) {
    return `[${value.map(formatYamlValue).join(', ')}]`;
  }

  if (typeof value === 'object' && value !== null) {
    // For nested objects, use JSON-style formatting
    return JSON.stringify(value);
  }

  return String(value);
}
