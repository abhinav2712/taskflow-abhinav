import axios from "axios";

import { useAuthStore } from "store/auth";
import type {
  ApiErrorResponse,
  AuthResponse,
  CreateProjectData,
  CreateTaskData,
  Project,
  ProjectsResponse,
  TasksResponse,
  Task,
  UpdateProjectData,
  UpdateTaskData,
  UsersResponse,
} from "types";

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

export const projectsApi = {
  list: async () => {
    const response = await api.get<ProjectsResponse>("/projects");
    return response.data;
  },
  create: async (payload: CreateProjectData) => {
    const response = await api.post<Project>("/projects", payload);
    return response.data;
  },
  get: async (id: string) => {
    const response = await api.get<Project>(`/projects/${id}`);
    return response.data;
  },
  update: async (id: string, payload: UpdateProjectData) => {
    const response = await api.patch<Project>(`/projects/${id}`, payload);
    return response.data;
  },
  delete: async (id: string) => {
    await api.delete(`/projects/${id}`);
  },
};

export const tasksApi = {
  list: async (projectId: string, filters?: { status?: string; assignee?: string }) => {
    const response = await api.get<TasksResponse>(`/projects/${projectId}/tasks`, {
      params: {
        status: filters?.status || undefined,
        assignee: filters?.assignee || undefined,
      },
    });

    return response.data;
  },
  create: async (projectId: string, payload: CreateTaskData) => {
    const response = await api.post<Task>(`/projects/${projectId}/tasks`, payload);
    return response.data;
  },
  update: async (id: string, payload: UpdateTaskData) => {
    const response = await api.patch<Task>(`/tasks/${id}`, payload);
    return response.data;
  },
  delete: async (id: string) => {
    await api.delete(`/tasks/${id}`);
  },
};

export const usersApi = {
  list: async () => {
    const response = await api.get<UsersResponse>("/users");
    return response.data;
  },
};

export function getApiErrorMessage(error: unknown): ApiErrorResponse {
  if (axios.isAxiosError<ApiErrorResponse>(error) && error.response?.data) {
    return error.response.data;
  }

  return { error: "Something went wrong. Please try again." };
}
