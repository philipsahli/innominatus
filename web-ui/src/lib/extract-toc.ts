/**
 * Extract Table of Contents from markdown content
 * Parses H2 and H3 headings and generates a navigable TOC structure
 */

export interface TocItem {
  id: string;
  text: string;
  level: 2 | 3;
}

/**
 * Extracts H2 and H3 headings from markdown content
 * Uses the same slug generation as rehype-slug for consistency
 */
export function extractTableOfContents(markdown: string): TocItem[] {
  const headingRegex = /^(#{2,3})\s+(.+)$/gm;
  const toc: TocItem[] = [];
  let match;

  while ((match = headingRegex.exec(markdown)) !== null) {
    const level = match[1].length as 2 | 3;
    const text = match[2].trim();

    // Generate slug similar to rehype-slug
    // Remove markdown formatting, convert to lowercase, replace spaces with hyphens
    const id = text
      .toLowerCase()
      .replace(/[`*_[\]]/g, '') // Remove markdown formatting
      .replace(/[^\w\s-]/g, '') // Remove special chars except word chars, spaces, hyphens
      .replace(/\s+/g, '-') // Replace spaces with hyphens
      .replace(/-+/g, '-') // Replace multiple hyphens with single
      .replace(/^-|-$/g, ''); // Remove leading/trailing hyphens

    toc.push({ id, text, level });
  }

  return toc;
}
