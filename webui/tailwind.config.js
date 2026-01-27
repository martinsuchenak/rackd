/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/**/*.{html,js,ts}',
    './index.html'
  ],
  theme: {
    extend: {
      colors: {
        'deep-slate': '#0F172A',
        'core-blue': '#3B82F6',
        'accent-cyan': '#06B6D4',
        'light-gray': '#F8FAFC',
      },
      fontFamily: {
        primary: ['Inter', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'monospace'],
      },
      fontSize: {
        'logo': '56px',
        'main-title': '64px',
        'subtitle': '24px',
        'section-title': '16px',
        'body': '15px',
        'small': '12px',
      },
      fontWeight: {
        bold: '700',
        'extra-bold': '800',
        'semi-bold': '600',
        medium: '500',
        normal: '400',
      },
    },
  },
  plugins: [],
}
