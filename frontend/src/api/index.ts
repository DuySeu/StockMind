import axios from "axios";

const axiosClient = axios.create({
  baseURL: "http://localhost:8080/v1/",
  headers: {
    "Content-Type": "application/json",
  },
});

axiosClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem("authToken");
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    // Handle request errors
    return Promise.reject(error);
  }
);
axiosClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error?.response?.status === 401) {
      window.localStorage.clear();
      window.location.href = "/login";
    }
    return Promise.reject(error?.response?.data);
  }
);

export default axiosClient;
