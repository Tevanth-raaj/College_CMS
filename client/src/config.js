// API Configuration
const API_BASE_URL =
  process.env.REACT_APP_API_URL || "https://academics.bitsathy.ac.in/api" || "http://localhost:8080/api";

const GOOGLE_CLIENT_ID =
  "880469513355-ndrhbbo7m85kete7010vj4fh3nabc3g2.apps.googleusercontent.com" || process.env.REACT_APP_GOOGLE_CLIENT_ID

export { API_BASE_URL, GOOGLE_CLIENT_ID };
