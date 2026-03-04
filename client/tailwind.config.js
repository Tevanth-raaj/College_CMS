/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,jsx,ts,tsx}"],
  theme: {
    extend: {
      colors: {
        primary: "#7D53F6",
        secondary: "#FBFAFF",
        primary_light:"rgb(125 83 246 / 5%)",
        primary_dim:"rgb(125 83 246 / 14%)",
        primary_medium:"rgb(125 83 246 / 65%)",
        background: "rgb(238 241 249/1)",
        "sec-background": "#ECE8FE",
        "sec-dim": "rgb(253 253 253 / 75%)",
        dark: "#1e1e1e",
        iconColor: "#5F6388",
      },

      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },

      boxShadow: {
        soft: '0 2px 15px -3px rgba(0, 0, 0, 0.07), 0 10px 20px -2px rgba(0, 0, 0, 0.04)',
        card: '0 1px 3px 0 rgba(0, 0, 0, 0.1), 0 1px 2px 0 rgba(0, 0, 0, 0.06)',
      },
    },
  },

  plugins: [require("daisyui")],

  daisyui: {
    themes: [
      {
        light: {
          primary: "rgb(143 121 250)",
          secondary: "#ffffff",
          accent: "#0369a1",
          neutral: "#1f2937",
          "base-100": "#ffffff",
          "base-200": "#f9fafb",
          "base-300": "#f3f4f6",
          info: "#3b82f6",
          success: "#10b981",
          warning: "#f59e0b",
          error: "#ef4444",
        },
      },
    ],
    base: true,
    styled: true,
    utils: true,
  },
}
