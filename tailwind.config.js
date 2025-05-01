/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./web/templates/**/*.html",
    "./web/static/js/**/*.js",
    "./web/static/css/**/*.css" // include CSS sources for @apply
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#f0f9ff',
          100: '#e0f2fe',
          200: '#b9e6fe',
          300: '#7dd3fc',
          400: '#38bdf8',
          500: '#0ea5e9',
          600: '#0284c7',
          700: '#0369a1',
          800: '#075985',
          900: '#0c4a6e',
        },
      },
    },
  },
  corePlugins: {
    // Ensure padding utilities are enabled
    padding: true,
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
