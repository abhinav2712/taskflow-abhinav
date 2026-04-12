export interface User {
  id: string;
  name: string;
  email: string;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

export interface ApiErrorResponse {
  error: string;
  fields?: Record<string, string>;
}

export type TaskStatus = "todo" | "in_progress" | "done";
export type TaskPriority = "low" | "medium" | "high";

export interface Task {
  id: string;
  title: string;
  description?: string | null;
  status: TaskStatus;
  priority: TaskPriority;
  project_id: string;
  assignee_id?: string | null;
  creator_id: string;
  due_date?: string | null;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string | null;
  owner_id: string;
  created_at: string;
  updated_at: string;
  tasks?: Task[];
}

export interface ProjectsResponse {
  projects: Project[];
}

export interface TasksResponse {
  tasks: Task[];
}

export interface UsersResponse {
  users: User[];
}

export interface CreateProjectData {
  name: string;
  description?: string;
}

export interface UpdateProjectData {
  name?: string;
  description?: string;
}

export interface CreateTaskData {
  title: string;
  description?: string;
  priority: TaskPriority;
  assignee_id?: string;
  due_date?: string;
}

export interface UpdateTaskData {
  title?: string;
  description?: string;
  status?: TaskStatus;
  priority?: TaskPriority;
  assignee_id?: string;
  due_date?: string;
}
