/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}'
  ],
  theme: {
    extend: {
      colors: {
        primary: '#0066cc',
        secondary: '#004c99',
        accent: '#00aaff',
        dark: '#1a1a2e',
        light: '#f0f8ff',
        'gray-dark': '#333344',
        'gray-medium': '#666688',
        'gray-light': '#e0e6ed'
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif']
      },
      boxShadow: {
        card: '0 4px 20px rgba(0, 0, 0, 0.08)',
        hover: '0 8px 30px rgba(0, 102, 204, 0.15)'
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite'
      }
    }
  }
}

