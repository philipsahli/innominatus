import type { Config } from 'tailwindcss';
import tailwindcssAnimate from 'tailwindcss-animate';
import typography from '@tailwindcss/typography';

const config: Config = {
  darkMode: 'class',
  content: [
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  prefix: '',
  theme: {
    container: {
      center: true,
      padding: '2rem',
      screens: {
        '2xl': '1400px',
      },
    },
    extend: {
      colors: {
        border: 'rgb(229 231 235)', // gray-200
        input: 'rgb(229 231 235)', // gray-200
        ring: 'rgb(132 204 22)', // lime-500
        background: 'rgb(255 255 255)', // white
        foreground: 'rgb(17 24 39)', // gray-900
        primary: {
          DEFAULT: 'rgb(132 204 22)', // lime-500 - our lemon green
          foreground: 'rgb(255 255 255)', // white
        },
        secondary: {
          DEFAULT: 'rgb(248 250 252)', // slate-50
          foreground: 'rgb(15 23 42)', // slate-900
        },
        destructive: {
          DEFAULT: 'rgb(239 68 68)', // red-500
          foreground: 'rgb(255 255 255)', // white
        },
        muted: {
          DEFAULT: 'rgb(248 250 252)', // slate-50
          foreground: 'rgb(100 116 139)', // slate-500
        },
        accent: {
          DEFAULT: 'rgb(248 250 252)', // slate-50
          foreground: 'rgb(15 23 42)', // slate-900
        },
        popover: {
          DEFAULT: 'rgb(255 255 255)', // white
          foreground: 'rgb(17 24 39)', // gray-900
        },
        card: {
          DEFAULT: 'rgb(255 255 255)', // white
          foreground: 'rgb(17 24 39)', // gray-900
        },
        // Status colors for IDP operations
        success: {
          DEFAULT: 'hsl(var(--success))',
          foreground: 'hsl(var(--success-foreground))',
          muted: 'hsl(var(--success-muted))',
        },
        warning: {
          DEFAULT: 'hsl(var(--warning))',
          foreground: 'hsl(var(--warning-foreground))',
          muted: 'hsl(var(--warning-muted))',
        },
        error: {
          DEFAULT: 'hsl(var(--error))',
          foreground: 'hsl(var(--error-foreground))',
          muted: 'hsl(var(--error-muted))',
        },
        info: {
          DEFAULT: 'hsl(var(--info))',
          foreground: 'hsl(var(--info-foreground))',
          muted: 'hsl(var(--info-muted))',
        },
        // IDP-specific semantic colors
        workflow: {
          running: 'hsl(var(--workflow-running))',
          completed: 'hsl(var(--workflow-completed))',
          failed: 'hsl(var(--workflow-failed))',
          pending: 'hsl(var(--workflow-pending))',
        },
        resource: {
          active: 'hsl(var(--resource-active))',
          provisioning: 'hsl(var(--resource-provisioning))',
          degraded: 'hsl(var(--resource-degraded))',
          terminated: 'hsl(var(--resource-terminated))',
        },
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Consolas', 'monospace'],
      },
      animation: {
        'accordion-down': 'accordion-down 0.2s ease-out',
        'accordion-up': 'accordion-up 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'bounce-soft': 'bounce-soft 1s ease-in-out infinite',
      },
      keyframes: {
        'accordion-down': {
          from: { height: '0' },
          to: { height: 'var(--radix-accordion-content-height)' },
        },
        'accordion-up': {
          from: { height: 'var(--radix-accordion-content-height)' },
          to: { height: '0' },
        },
        'bounce-soft': {
          '0%, 100%': {
            transform: 'translateY(-5%)',
            animationTimingFunction: 'cubic-bezier(0.8, 0, 1, 1)',
          },
          '50%': {
            transform: 'translateY(0)',
            animationTimingFunction: 'cubic-bezier(0, 0, 0.2, 1)',
          },
        },
      },
      spacing: {
        '18': '4.5rem',
        '88': '22rem',
      },
      backdropBlur: {
        xs: '2px',
      },
    },
  },
  plugins: [tailwindcssAnimate, typography],
} satisfies Config;

export default config;
