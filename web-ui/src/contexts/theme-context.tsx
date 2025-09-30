"use client"

import React, { createContext, useContext, useState, useEffect } from 'react'

export interface Theme {
  id: string
  name: string
  description: string
  colors: {
    primary: string
    secondary: string
    accent: string
    success: string
    warning: string
    error: string
    background: {
      from: string
      via: string
      to: string
    }
    cards: {
      applications: { from: string; to: string }
      workflows: { from: string; to: string }
      resources: { from: string; to: string }
      users: { from: string; to: string }
    }
  }
}

export const themes: Theme[] = [
  {
    id: 'lemon-green',
    name: 'Lemon Green Professional',
    description: 'Modern lemon green with accessibility-focused design',
    colors: {
      primary: '[color:hsl(var(--primary))]',
      secondary: '[color:hsl(var(--secondary))]',
      accent: '[color:hsl(var(--accent))]',
      success: '[color:hsl(var(--success))]',
      warning: '[color:hsl(var(--warning))]',
      error: '[color:hsl(var(--error))]',
      background: {
        from: 'slate-50',
        via: 'gray-50',
        to: 'zinc-50'
      },
      cards: {
        applications: { from: 'blue-500/10', to: 'blue-600/5' },
        workflows: { from: 'emerald-500/10', to: 'emerald-600/5' },
        resources: { from: 'purple-500/10', to: 'purple-600/5' },
        users: { from: 'orange-500/10', to: 'orange-600/5' }
      }
    }
  },
  {
    id: 'blue-600',
    name: 'Ocean Breeze',
    description: 'Cool blues and purples for a professional look',
    colors: {
      primary: 'blue-600',
      secondary: 'slate-600',
      accent: 'purple-600',
      success: 'emerald-600',
      warning: 'amber-600',
      error: 'red-600',
      background: {
        from: 'slate-50',
        via: 'blue-50',
        to: 'indigo-100'
      },
      cards: {
        applications: { from: 'blue-500', to: 'blue-600' },
        workflows: { from: 'emerald-500', to: 'emerald-600' },
        resources: { from: 'purple-500', to: 'purple-600' },
        users: { from: 'amber-500', to: 'orange-500' }
      }
    }
  },
  {
    id: 'orange-600',
    name: 'Sunset Glow',
    description: 'Warm oranges and pinks for a vibrant feel',
    colors: {
      primary: 'orange-600',
      secondary: 'rose-600',
      accent: 'pink-600',
      success: 'emerald-600',
      warning: 'yellow-600',
      error: 'red-600',
      background: {
        from: 'orange-50',
        via: 'rose-50',
        to: 'pink-100'
      },
      cards: {
        applications: { from: 'orange-500', to: 'red-500' },
        workflows: { from: 'rose-500', to: 'pink-500' },
        resources: { from: 'purple-500', to: 'violet-500' },
        users: { from: 'amber-500', to: 'yellow-500' }
      }
    }
  },
  {
    id: 'emerald-600',
    name: 'Forest Serenity',
    description: 'Natural greens for a calm, organic atmosphere',
    colors: {
      primary: 'emerald-600',
      secondary: 'green-600',
      accent: 'teal-600',
      success: 'green-600',
      warning: 'amber-600',
      error: 'red-600',
      background: {
        from: 'emerald-50',
        via: 'green-50',
        to: 'teal-100'
      },
      cards: {
        applications: { from: 'emerald-500', to: 'green-600' },
        workflows: { from: 'teal-500', to: 'cyan-500' },
        resources: { from: 'green-500', to: 'lime-500' },
        users: { from: 'yellow-500', to: 'amber-500' }
      }
    }
  },
  {
    id: 'violet-600',
    name: 'Midnight Aurora',
    description: 'Deep purples and teals with cosmic vibes',
    colors: {
      primary: 'violet-600',
      secondary: 'indigo-600',
      accent: 'cyan-600',
      success: 'emerald-600',
      warning: 'amber-600',
      error: 'red-600',
      background: {
        from: 'violet-50',
        via: 'indigo-50',
        to: 'cyan-100'
      },
      cards: {
        applications: { from: 'violet-500', to: 'purple-600' },
        workflows: { from: 'indigo-500', to: 'blue-600' },
        resources: { from: 'cyan-500', to: 'teal-500' },
        users: { from: 'pink-500', to: 'rose-500' }
      }
    }
  },
  {
    id: 'gray-600',
    name: 'Minimal Gray',
    description: 'Clean grays with subtle accents for focus',
    colors: {
      primary: 'gray-600',
      secondary: 'slate-600',
      accent: 'zinc-600',
      success: 'emerald-600',
      warning: 'amber-600',
      error: 'red-600',
      background: {
        from: 'gray-50',
        via: 'slate-50',
        to: 'zinc-100'
      },
      cards: {
        applications: { from: 'gray-500', to: 'slate-600' },
        workflows: { from: 'slate-500', to: 'zinc-500' },
        resources: { from: 'zinc-500', to: 'neutral-500' },
        users: { from: 'stone-500', to: 'gray-500' }
      }
    }
  },
  {
    id: 'black-white',
    name: 'Black & White',
    description: 'Pure black and white monochrome theme',
    colors: {
      primary: 'black',
      secondary: 'white',
      accent: 'black',
      success: 'black',
      warning: 'black',
      error: 'black',
      background: {
        from: 'white',
        via: 'white',
        to: 'white'
      },
      cards: {
        applications: { from: 'black', to: 'black' },
        workflows: { from: 'black', to: 'black' },
        resources: { from: 'black', to: 'black' },
        users: { from: 'black', to: 'black' }
      }
    }
  }
]

interface ThemeContextType {
  currentTheme: Theme
  setTheme: (themeId: string) => void
  themes: Theme[]
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined)

export function CustomThemeProvider({ children }: { children: React.ReactNode }) {
  const [currentTheme, setCurrentTheme] = useState<Theme>(themes[0])

  // Apply theme by updating CSS custom properties
  const applyTheme = (theme: Theme) => {
    const root = document.documentElement

    // Map theme colors to actual CSS values (RGB format for CSS variables)
    const colorMap: Record<string, string> = {
      'lemon-green': '132 204 22', // lime-500
      'blue-600': '37 99 235',     // blue-600
      'orange-600': '234 88 12',   // orange-600
      'emerald-600': '5 150 105',  // emerald-600
      'violet-600': '124 58 237',  // violet-600
      'gray-600': '75 85 99',      // gray-600
      'black-white': '0 0 0',      // pure black
    }

    // Update primary color based on theme
    const primaryColor = colorMap[theme.id] || colorMap['lemon-green']
    root.style.setProperty('--primary', primaryColor)

    // For black-white theme, update additional properties for complete monochrome
    if (theme.id === 'black-white') {
      root.style.setProperty('--background', '255 255 255') // white
      root.style.setProperty('--foreground', '0 0 0') // black
      root.style.setProperty('--muted', '245 245 245') // very light gray
      root.style.setProperty('--muted-foreground', '64 64 64') // dark gray
      root.style.setProperty('--border', '229 229 229') // light gray
      root.style.setProperty('--card', '255 255 255') // white
      root.style.setProperty('--card-foreground', '0 0 0') // black

      // Add CSS class for complete black/white override
      root.classList.add('theme-black-white')
    } else {
      // Remove black-white theme class and reset to defaults
      root.classList.remove('theme-black-white')
      root.style.setProperty('--background', '255 255 255') // white
      root.style.setProperty('--foreground', '17 24 39') // gray-900
      root.style.setProperty('--muted', '248 250 252') // slate-50
      root.style.setProperty('--muted-foreground', '100 116 139') // slate-500
      root.style.setProperty('--border', '229 231 235') // gray-200
      root.style.setProperty('--card', '255 255 255') // white
      root.style.setProperty('--card-foreground', '17 24 39') // gray-900
    }
  }

  useEffect(() => {
    const savedTheme = localStorage.getItem('custom-theme')
    if (savedTheme) {
      const theme = themes.find(t => t.id === savedTheme)
      if (theme) {
        setCurrentTheme(theme)
        applyTheme(theme)
      }
    } else {
      // Default to lemon green theme
      setCurrentTheme(themes[0])
      applyTheme(themes[0])
    }
  }, [])

  useEffect(() => {
    applyTheme(currentTheme)
  }, [currentTheme])

  const setTheme = (themeId: string) => {
    const theme = themes.find(t => t.id === themeId)
    if (theme) {
      setCurrentTheme(theme)
      localStorage.setItem('custom-theme', themeId)
    }
  }

  return (
    <ThemeContext.Provider value={{ currentTheme, setTheme, themes }}>
      {children}
    </ThemeContext.Provider>
  )
}

export function useCustomTheme() {
  const context = useContext(ThemeContext)
  if (context === undefined) {
    throw new Error('useCustomTheme must be used within a CustomThemeProvider')
  }
  return context
}