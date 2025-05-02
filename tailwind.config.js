/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
    "./web/static/css/**/*.css"
  ],
  safelist: [
    'bg-background',
    'dark:bg-background-dark',
    'bg-surface',
    'dark:bg-surface-dark',
    'text-text',
    'dark:text-text-dark'
  ],
  theme: {
    extend: {
      colors: {
        // Color Palette from Alpine SaaS design
        teal: '#468189',
        bittersweet: '#bf4342',
        night: '#0c0c0c',
        nyanza: '#f0ffce',
        sage: '#d2cca1',
        
        // Component Colors
        primary: '#468189', // teal
        'primary-500': '#468189',
        'primary-600': '#468189',
        'primary-700': '#3a6b6f',
        'primary-dark': '#3a6b6f',
        danger: '#bf4342', // bittersweet-shimmer
        'danger-dark': '#bf4342',

        // Semantic colors for light/dark themes
        background: '#f0ffce', // nyanza
        'background-dark': '#0c0c0c', // night
        surface: '#ffffff',
        'surface-dark': '#181818', // slightly darker for more contrast
        text: '#0c0c0c', // night
        'text-dark': '#f0ffce', // nyanza
        accent: '#d2cca1', // sage
        'accent-dark': '#fffad1', // lighter accent for links, etc.
        muted: '#d2cca1', // sage
        'muted-dark': '#e6e6b3', // lighter for better contrast
      }
    }
  },
  plugins: [
    require('@tailwindcss/forms')
  ]
}
