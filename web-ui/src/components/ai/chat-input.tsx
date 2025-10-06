'use client';

import React, { useState, KeyboardEvent } from 'react';
import { Button } from '@/components/ui/button';
import { Send, Sparkles } from 'lucide-react';

export interface ChatInputProps {
  onSend: (message: string) => void;
  onGenerateSpec?: (description: string) => void;
  disabled?: boolean;
  placeholder?: string;
}

export function ChatInput({ onSend, onGenerateSpec, disabled, placeholder }: ChatInputProps) {
  const [message, setMessage] = useState('');

  const handleSend = () => {
    if (message.trim() && !disabled) {
      onSend(message);
      setMessage('');
    }
  };

  const handleGenerateSpec = () => {
    if (message.trim() && !disabled && onGenerateSpec) {
      onGenerateSpec(message);
      setMessage('');
    }
  };

  const handleKeyPress = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4">
      <div className="flex gap-2">
        <textarea
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyPress={handleKeyPress}
          placeholder={placeholder || 'Ask me anything about innominatus...'}
          disabled={disabled}
          className="flex-1 resize-none rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 dark:text-gray-100 disabled:opacity-50"
          rows={3}
        />
        <div className="flex flex-col gap-2">
          <Button
            onClick={handleSend}
            disabled={disabled || !message.trim()}
            size="sm"
            className="bg-blue-500 hover:bg-blue-600 text-white"
          >
            <Send className="w-4 h-4" />
          </Button>
          {onGenerateSpec && (
            <Button
              onClick={handleGenerateSpec}
              disabled={disabled || !message.trim()}
              size="sm"
              variant="outline"
              title="Generate Score specification"
            >
              <Sparkles className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>
      <p className="text-xs text-muted-foreground mt-2">
        Press Enter to send, Shift+Enter for new line
        {onGenerateSpec && ' • Click ✨ to generate Score spec'}
      </p>
    </div>
  );
}
