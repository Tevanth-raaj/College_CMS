// API Configuration
const DEFAULT_API_BASE_URL =
  process.env.NODE_ENV === 'development'
    ? 'http://localhost:8080/api'
    : 'https://academics.bitsathy.ac.in/api'

const API_BASE_URL = (process.env.REACT_APP_API_URL || DEFAULT_API_BASE_URL).replace(/\/+$/, '')

const GOOGLE_CLIENT_ID =
  process.env.REACT_APP_GOOGLE_CLIENT_ID || "880469513355-ndrhbbo7m85kete7010vj4fh3nabc3g2.apps.googleusercontent.com" 

export { API_BASE_URL, GOOGLE_CLIENT_ID };
