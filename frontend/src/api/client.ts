import axios from "axios";

import { useAuthStore } from "store/auth";
import type { ApiErrorResponse, AuthResponse } from "types";

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().token;

  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }

  return config;
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().clearAuth();

      if (window.location.pathname !== "/login" && window.location.pathname !== "/register") {
        window.location.href = "/login";
      }
    }

    return Promise.reject(error);
  },
);

export const authApi = {
  login: async (payload: { email: string; password: string }) => {
    const response = await api.post<AuthResponse>("/auth/login", payload);
    return response.data;
  },
  register: async (payload: { name: string; email: string; password: string }) => {
    const response = await api.post<AuthResponse>("/auth/register", payload);
    return response.data;
  },
};

export function getApiErrorMessage(error: unknown): ApiErrorResponse {
  if (axios.isAxiosError<ApiErrorResponse>(error) && error.response?.data) {
    return error.response.data;
  }

  return { error: "Something went wrong. Please try again." };
}
