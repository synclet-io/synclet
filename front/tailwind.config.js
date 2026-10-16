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
          'DEFAULT': 'var(--color-primary)',
          'hover': 'var(--color-primary-hover)',
          'light': 'var(--color-primary-light)',
          'muted': 'var(--color-primary-muted)',
          'solid': 'var(--color-primary-solid)',
          'solid-hover': 'var(--color-primary-solid-hover)',
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
        'text-disabled': 'var(--color-text-disabled)',
        'on-primary': 'var(--color-text-on-primary)',
        'border': {
          DEFAULT: 'var(--color-border-default)',
          subtle: 'var(--color-border-subtle)',
          strong: 'var(--color-border-strong)',
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
          border: 'var(--color-status-success-border)',
        },
        'danger': {
          'DEFAULT': 'var(--color-status-danger)',
          'bg': 'var(--color-status-danger-bg)',
          'border': 'var(--color-status-danger-border)',
          'solid': 'var(--color-status-danger-solid)',
          'solid-hover': 'var(--color-status-danger-solid-hover)',
        },
        'warning': {
          DEFAULT: 'var(--color-status-warning)',
          bg: 'var(--color-status-warning-bg)',
          border: 'var(--color-status-warning-border)',
        },
        'info': {
          DEFAULT: 'var(--color-status-info)',
          bg: 'var(--color-status-info-bg)',
          border: 'var(--color-status-info-border)',
        },
        'accent': {
          DEFAULT: 'var(--color-accent)',
          bg: 'var(--color-accent-bg)',
        },
        'tooltip': {
          DEFAULT: 'var(--color-tooltip-bg)',
          text: 'var(--color-tooltip-text)',
        },
        'code': {
          'DEFAULT': 'var(--color-code-bg)',
          'text': 'var(--color-code-text)',
          'muted': 'var(--color-code-muted)',
          'control': 'var(--color-code-control)',
          'control-hover': 'var(--color-code-control-hover)',
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
