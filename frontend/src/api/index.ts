import axios from "axios";

// Create axios instance
const api = axios.create({ baseURL: "http://localhost:8080/v1" });

// Request interceptor
api.interceptors.request.use(
  (config) => {
    return config;
  },
  (error) => {
    return Promise.reject(error);
  },
);

// Response interceptor - Handle error 401/403
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response) {
      const { status } = error.response;

      // Redirect to login when authentication error
      if (status === 401 || status === 403) {
        console.warn(`Authentication error (${status}), redirecting to login`);
      }
    }

    return Promise.reject(error);
  },
);

export default api;
