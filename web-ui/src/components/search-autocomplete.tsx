'use client';

import React, { useState, useRef, useEffect } from 'react';
import { Input } from '@/components/ui/input';
import { Search, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { GraphNode } from '@/lib/api';
import { Badge } from '@/components/ui/badge';

interface SearchAutocompleteProps {
  nodes: GraphNode[];
  value: string;
  onChange: (value: string) => void;
  onSelectNode?: (node: GraphNode) => void;
  placeholder?: string;
}

export function SearchAutocomplete({
  nodes,
  value,
  onChange,
  onSelectNode,
  placeholder = 'Search nodes by name...',
}: SearchAutocompleteProps) {
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const inputRef = useRef<HTMLInputElement>(null);
  const suggestionsRef = useRef<HTMLDivElement>(null);

  // Filter nodes based on search term
  const suggestions = value
    ? nodes
        .filter((node) => {
          const searchLower = value.toLowerCase();
          return (
            node.name.toLowerCase().includes(searchLower) ||
            node.type.toLowerCase().includes(searchLower) ||
            node.metadata?.resource_type?.toLowerCase().includes(searchLower) ||
            node.metadata?.provider_id?.toLowerCase().includes(searchLower)
          );
        })
        .slice(0, 10) // Limit to 10 suggestions
    : [];

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    onChange(e.target.value);
    setShowSuggestions(true);
    setSelectedIndex(-1);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (!showSuggestions || suggestions.length === 0) return;

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((prev) => Math.min(prev + 1, suggestions.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((prev) => Math.max(prev - 1, -1));
        break;
      case 'Enter':
        e.preventDefault();
        if (selectedIndex >= 0 && suggestions[selectedIndex]) {
          handleSelectSuggestion(suggestions[selectedIndex]);
        }
        break;
      case 'Escape':
        setShowSuggestions(false);
        setSelectedIndex(-1);
        break;
    }
  };

  const handleSelectSuggestion = (node: GraphNode) => {
    onChange(node.name);
    setShowSuggestions(false);
    setSelectedIndex(-1);
    if (onSelectNode) {
      onSelectNode(node);
    }
  };

  const handleClear = () => {
    onChange('');
    setShowSuggestions(false);
    setSelectedIndex(-1);
    inputRef.current?.focus();
  };

  // Close suggestions when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        suggestionsRef.current &&
        !suggestionsRef.current.contains(event.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(event.target as Node)
      ) {
        setShowSuggestions(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Scroll selected suggestion into view
  useEffect(() => {
    if (selectedIndex >= 0 && suggestionsRef.current) {
      const selectedElement = suggestionsRef.current.children[selectedIndex] as HTMLElement;
      if (selectedElement) {
        selectedElement.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    }
  }, [selectedIndex]);

  const getNodeIcon = (type: string) => {
    switch (type) {
      case 'spec':
        return '📦';
      case 'workflow':
        return '🔄';
      case 'step':
        return '▶️';
      case 'resource':
        return '🗄️';
      case 'provider':
        return '👥';
      default:
        return '•';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'completed':
      case 'succeeded':
        return 'bg-green-500';
      case 'running':
      case 'provisioning':
      case 'in_progress':
        return 'bg-blue-500';
      case 'failed':
      case 'error':
        return 'bg-red-500';
      default:
        return 'bg-gray-400';
    }
  };

  return (
    <div className="relative w-full">
      {/* Search Input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
        <Input
          ref={inputRef}
          type="text"
          placeholder={placeholder}
          value={value}
          onChange={handleInputChange}
          onKeyDown={handleKeyDown}
          onFocus={() => value && setShowSuggestions(true)}
          className="pl-10 pr-10"
        />
        {value && (
          <Button
            variant="ghost"
            size="sm"
            onClick={handleClear}
            className="absolute right-2 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
          >
            <X className="w-4 h-4" />
          </Button>
        )}
      </div>

      {/* Suggestions Dropdown */}
      {showSuggestions && suggestions.length > 0 && (
        <div
          ref={suggestionsRef}
          className="absolute top-full mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg max-h-80 overflow-y-auto z-50"
        >
          {suggestions.map((node, index) => (
            <div
              key={node.id}
              onClick={() => handleSelectSuggestion(node)}
              className={`px-4 py-3 cursor-pointer border-b border-gray-100 dark:border-gray-700 last:border-b-0 ${
                index === selectedIndex
                  ? 'bg-blue-50 dark:bg-blue-900/30'
                  : 'hover:bg-gray-50 dark:hover:bg-gray-700'
              }`}
            >
              <div className="flex items-center gap-3">
                {/* Icon and Status Indicator */}
                <div className="flex items-center gap-2">
                  <span className="text-lg">{getNodeIcon(node.type)}</span>
                  <div className={`w-2 h-2 rounded-full ${getStatusColor(node.status)}`} />
                </div>

                {/* Node Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-sm truncate">{node.name}</span>
                    <Badge variant="outline" className="text-xs capitalize">
                      {node.type}
                    </Badge>
                  </div>

                  {/* Additional metadata */}
                  <div className="flex items-center gap-2 mt-1 text-xs text-muted-foreground">
                    {node.metadata?.resource_type && (
                      <span className="truncate">{node.metadata.resource_type}</span>
                    )}
                    {node.metadata?.provider_id && (
                      <span className="truncate">• {node.metadata.provider_id}</span>
                    )}
                    {node.status && <span className="capitalize">• {node.status}</span>}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* No results message */}
      {showSuggestions && value && suggestions.length === 0 && (
        <div className="absolute top-full mt-1 w-full bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg p-4 text-center text-sm text-muted-foreground z-50">
          No matching nodes found
        </div>
      )}
    </div>
  );
}
