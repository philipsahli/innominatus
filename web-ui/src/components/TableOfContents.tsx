'use client';

import { useState, useEffect } from 'react';
import { List } from 'lucide-react';
import { TocItem } from '@/lib/extract-toc';

interface TableOfContentsProps {
  items: TocItem[];
}

export function TableOfContents({ items }: TableOfContentsProps) {
  const [activeId, setActiveId] = useState<string>('');
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    // Track active section on scroll
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setActiveId(entry.target.id);
          }
        });
      },
      {
        rootMargin: '-80px 0px -80% 0px',
      }
    );

    // Observe all heading elements
    items.forEach(({ id }) => {
      const element = document.getElementById(id);
      if (element) {
        observer.observe(element);
      }
    });

    return () => observer.disconnect();
  }, [items]);

  const scrollToSection = (id: string) => {
    const element = document.getElementById(id);
    if (element) {
      const yOffset = -80; // Account for sticky header
      const y = element.getBoundingClientRect().top + window.pageYOffset + yOffset;
      window.scrollTo({ top: y, behavior: 'smooth' });
      setIsOpen(false);
    }
  };

  if (items.length === 0) {
    return null;
  }

  return (
    <>
      {/* Mobile Toggle Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="lg:hidden fixed bottom-6 right-6 p-3 bg-indigo-600 text-white rounded-full shadow-lg hover:bg-indigo-700 transition-colors z-50"
        aria-label="Toggle table of contents"
      >
        <List className="w-5 h-5" />
      </button>

      {/* Mobile Overlay */}
      {isOpen && (
        <div
          className="lg:hidden fixed inset-0 bg-black/50 z-40"
          onClick={() => setIsOpen(false)}
        />
      )}

      {/* TOC Container */}
      <aside className="hidden lg:block w-64 flex-shrink-0">
        <nav
          className="sticky top-6 h-fit max-h-[calc(100vh-3rem)] overflow-y-auto bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg p-4"
          aria-label="Table of contents"
        >
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100 uppercase tracking-wide mb-3">
            On This Page
          </h2>
          <ul className="space-y-2">
            {items.map(({ id, text, level }) => (
              <li key={id}>
                <button
                  onClick={() => scrollToSection(id)}
                  className={`
                    w-full text-left text-sm transition-colors
                    ${level === 3 ? 'pl-4' : ''}
                    ${
                      activeId === id
                        ? 'text-indigo-600 dark:text-indigo-400 font-medium'
                        : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'
                    }
                  `}
                >
                  {text}
                </button>
              </li>
            ))}
          </ul>
        </nav>
      </aside>

      {/* Mobile TOC Panel */}
      <nav
        className={`
          lg:hidden fixed top-0 left-0
          w-80 max-w-[85vw] h-full
          bg-white dark:bg-gray-800
          border-r border-gray-200 dark:border-gray-700
          shadow-2xl
          overflow-y-auto
          transition-transform duration-300
          z-50
          ${isOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
        aria-label="Table of contents"
      >
        <div className="p-6">
          <h2 className="text-sm font-semibold text-gray-900 dark:text-gray-100 uppercase tracking-wide mb-4">
            On This Page
          </h2>
          <ul className="space-y-2">
            {items.map(({ id, text, level }) => (
              <li key={id}>
                <button
                  onClick={() => scrollToSection(id)}
                  className={`
                    w-full text-left text-sm transition-colors py-1
                    ${level === 3 ? 'pl-4' : ''}
                    ${
                      activeId === id
                        ? 'text-indigo-600 dark:text-indigo-400 font-medium'
                        : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-100'
                    }
                  `}
                >
                  {text}
                </button>
              </li>
            ))}
          </ul>
        </div>
      </nav>
    </>
  );
}
