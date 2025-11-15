import yaml from 'js-yaml';

/**
 * Format data as YAML with preserved key order
 * @param data - Data to format
 * @returns YAML string with original key order preserved
 */
export function formatAsYAML(data: any): string {
  try {
    return yaml.dump(data, {
      indent: 2, // 2 spaces per indent level
      lineWidth: -1, // No line wrapping
      noRefs: true, // Don't use anchors/references
      sortKeys: false, // Preserve original key order (IMPORTANT!)
    });
  } catch (error) {
    console.error('Failed to format as YAML:', error);
    // Fallback to JSON if YAML formatting fails
    return JSON.stringify(data, null, 2);
  }
}
