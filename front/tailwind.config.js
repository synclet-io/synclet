import defaultTheme from 'tailwindcss/defaultTheme'

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', ...defaultTheme.fontFamily.sans],
      },
      colors: {
        'primary': {
          DEFAULT: 'var(--color-primary)',
          hover: 'var(--color-primary-hover)',
          light: 'var(--color-primary-light)',
          muted: 'var(--color-primary-muted)',
          50: '#eff6ff',
          100: '#dbeafe',
          200: '#bfdbfe',
          300: '#93c5fd',
          400: '#60a5fa',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
          800: '#1e40af',
          900: '#1e3a8a',
        },
        'surface': {
          DEFAULT: 'var(--color-bg-surface)',
          raised: 'var(--color-bg-surface-raised)',
          hover: 'var(--color-bg-surface-hover)',
        },
        'page': 'var(--color-bg-page)',
        'heading': 'var(--color-text-heading)',
        'text-primary': 'var(--color-text-primary)',
        'text-secondary': 'var(--color-text-secondary)',
        'text-muted': 'var(--color-text-muted)',
        'on-primary': 'var(--color-text-on-primary)',
        'border': {
          DEFAULT: 'var(--color-border-default)',
          subtle: 'var(--color-border-subtle)',
        },
        'sidebar': {
          'DEFAULT': 'var(--color-sidebar-bg)',
          'text': 'var(--color-sidebar-text)',
          'text-active': 'var(--color-sidebar-text-active)',
          'hover': 'var(--color-sidebar-hover)',
          'active': 'var(--color-sidebar-active)',
          'border': 'var(--color-sidebar-border)',
        },
        'success': {
          DEFAULT: 'var(--color-status-success)',
          bg: 'var(--color-status-success-bg)',
          50: '#f0fdf4',
          100: '#dcfce7',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
        },
        'danger': {
          DEFAULT: 'var(--color-status-danger)',
          bg: 'var(--color-status-danger-bg)',
          50: '#fef2f2',
          100: '#fee2e2',
          200: '#fecaca',
          300: '#fca5a5',
          500: '#ef4444',
          600: '#dc2626',
          700: '#b91c1c',
        },
        'warning': {
          DEFAULT: 'var(--color-status-warning)',
          bg: 'var(--color-status-warning-bg)',
          50: '#fffbeb',
          100: '#fef3c7',
          500: '#f59e0b',
          600: '#d97706',
        },
        'info': {
          DEFAULT: 'var(--color-status-info)',
          bg: 'var(--color-status-info-bg)',
        },
      },
      borderRadius: {
        'DEFAULT': '0.5rem',
        'xl': '0.75rem',
        '2xl': '1rem',
      },
      boxShadow: {
        xs: 'var(--shadow-xs)',
        soft: 'var(--shadow-soft)',
        raised: 'var(--shadow-raised)',
        overlay: 'var(--shadow-overlay)',
      },
    },
  },
  plugins: [],
}
