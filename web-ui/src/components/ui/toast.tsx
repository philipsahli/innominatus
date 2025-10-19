'use client';

import React, { useEffect, useState } from 'react';
import { X } from 'lucide-react';

export interface ToastProps {
  id: string;
  title: string;
  description?: string;
  variant?: 'default' | 'destructive';
  onClose: () => void;
}

export function Toast({ id, title, description, variant = 'default', onClose }: ToastProps) {
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    // Trigger enter animation
    setTimeout(() => setIsVisible(true), 10);

    // Auto-dismiss after 5 seconds
    const timer = setTimeout(() => {
      setIsVisible(false);
      setTimeout(onClose, 300); // Wait for exit animation
    }, 5000);

    return () => clearTimeout(timer);
  }, [onClose]);

  const handleClose = () => {
    setIsVisible(false);
    setTimeout(onClose, 300);
  };

  const variantStyles = {
    default: 'bg-blue-600 border-blue-500',
    destructive: 'bg-red-600 border-red-500',
  };

  return (
    <div
      className={`
        pointer-events-auto relative flex w-full max-w-md rounded-lg border-2 p-4 shadow-lg
        transition-all duration-300 ease-in-out
        ${variantStyles[variant]}
        ${isVisible ? 'translate-x-0 opacity-100' : 'translate-x-full opacity-0'}
      `}
    >
      <div className="flex-1">
        <div className="font-semibold text-white">{title}</div>
        {description && (
          <div className="mt-1 text-sm text-gray-100 opacity-90">
            {description}
          </div>
        )}
      </div>
      <button
        onClick={handleClose}
        className="ml-4 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md text-white opacity-70 hover:opacity-100 transition-opacity"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
