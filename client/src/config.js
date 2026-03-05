// API Configuration
const API_BASE_URL =
  process.env.REACT_APP_API_URL ||
  "http://localhost:5000/api"

const GOOGLE_CLIENT_ID =
  process.env.REACT_APP_GOOGLE_CLIENT_ID || ""

export { API_BASE_URL, GOOGLE_CLIENT_ID };