import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { getApiErrorMessage, projectsApi, tasksApi } from "api/client";
import Navbar from "components/Navbar";
import TaskCard from "components/TaskCard";
import TaskModal from "components/TaskModal";
import { useAuthStore } from "store/auth";
import type { Project, Task, TaskStatus } from "types";

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>();
  const user = useAuthStore((state) => state.user);
  const currentUserId = user?.id ?? "";

  const [project, setProject] = useState<Project | null>(null);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [tasksLoading, setTasksLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");
  const [assigneeFilter, setAssigneeFilter] = useState("");
  const [taskModal, setTaskModal] = useState<{ open: boolean; task?: Task }>({ open: false });
  const [feedback, setFeedback] = useState<string | null>(null);

  function matchesActiveFilters(task: Task) {
    const matchesStatus = !statusFilter || task.status === statusFilter;
    const matchesAssignee = !assigneeFilter || task.assignee_id === assigneeFilter;
    return matchesStatus && matchesAssignee;
  }

  useEffect(() => {
    if (!id) {
      setError("Project not found.");
      setLoading(false);
      return;
    }

    const projectId = id;
    let cancelled = false;

    async function loadProject() {
      setLoading(true);
      setError(null);

      try {
        const projectResponse = await projectsApi.get(projectId);
        if (cancelled) {
          return;
        }
        setProject(projectResponse);
        setTasks(projectResponse.tasks ?? []);
      } catch (loadError) {
        if (!cancelled) {
          setError(getApiErrorMessage(loadError).error);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadProject();

    return () => {
      cancelled = true;
    };
  }, [id]);

  useEffect(() => {
    if (!id || !project) {
      return;
    }

    const projectId = id;
    let cancelled = false;

    async function loadFilteredTasks() {
      // Skip the API call if no filters are active — initial tasks come from GetProject
      if (!statusFilter && !assigneeFilter) {
        return;
      }

      setTasksLoading(true);

      try {
        const response = await tasksApi.list(projectId, {
          status: statusFilter,
          assignee: assigneeFilter,
        });
        if (!cancelled) {
          setTasks(response.tasks);
        }
      } catch (loadError) {
        if (!cancelled) {
          setFeedback(getApiErrorMessage(loadError).error);
        }
      } finally {
        if (!cancelled) {
          setTasksLoading(false);
        }
      }
    }

    void loadFilteredTasks();

    return () => {
      cancelled = true;
    };
  }, [id, statusFilter, assigneeFilter]);

  async function handleOptimisticStatusChange(taskId: string, status: TaskStatus) {
    const previousTasks = tasks;

    setFeedback(null);
    setTasks((currentTasks) => {
      const nextTasks = currentTasks.map((task) => (task.id === taskId ? { ...task, status } : task));
      return nextTasks.filter(matchesActiveFilters);
    });

    try {
      const updatedTask = await tasksApi.update(taskId, { status });
      setTasks((currentTasks) => {
        if (!matchesActiveFilters(updatedTask)) {
          return currentTasks.filter((task) => task.id !== updatedTask.id);
        }

        return currentTasks.map((task) => (task.id === updatedTask.id ? updatedTask : task));
      });
    } catch (updateError) {
      setTasks(previousTasks);
      setFeedback(getApiErrorMessage(updateError).error);
    }
  }

  async function handleDeleteTask(task: Task) {
    const previousTasks = tasks;
    setFeedback(null);
    setTasks((currentTasks) => currentTasks.filter((currentTask) => currentTask.id !== task.id));

    try {
      await tasksApi.delete(task.id);
    } catch (deleteError) {
      setTasks(previousTasks);
      setFeedback(getApiErrorMessage(deleteError).error);
    }
  }

  function handleTaskSaved(savedTask: Task) {
    setTasks((currentTasks) => {
      if (!matchesActiveFilters(savedTask)) {
        return currentTasks.filter((task) => task.id !== savedTask.id);
      }

      const existingIndex = currentTasks.findIndex((task) => task.id === savedTask.id);

      if (existingIndex >= 0) {
        return currentTasks.map((task) => (task.id === savedTask.id ? savedTask : task));
      }

      return [savedTask, ...currentTasks];
    });
  }

  if (loading) {
    return (
      <div className="app-layout">
        <Navbar />
        <main className="page-shell">
          <div className="placeholder-card">
            <p className="placeholder-card__eyebrow">Loading</p>
            <h2>Fetching project details...</h2>
          </div>
        </main>
      </div>
    );
  }

  if (error || !project) {
    return (
      <div className="app-layout">
        <Navbar />
        <main className="page-shell">
          <div className="placeholder-card">
            <p className="placeholder-card__eyebrow">Something went wrong</p>
            <h2>{error ?? "Project not found."}</h2>
            <Link className="button button--primary" to="/projects">
              Back to projects
            </Link>
          </div>
        </main>
      </div>
    );
  }

  return (
    <div className="app-layout">
      <Navbar />

      <main className="page-shell">
        <section className="project-hero">
          <div>
            <Link className="back-link" to="/projects">
              ← Back to projects
            </Link>
            <p className="placeholder-card__eyebrow">Project detail</p>
            <h2>{project.name}</h2>
            <p>{project.description?.trim() ? project.description : "No description added yet."}</p>
          </div>

          <div className="project-hero__meta">
            <span>Created {new Date(project.created_at).toLocaleDateString()}</span>
            <button className="button button--primary" onClick={() => setTaskModal({ open: true })} type="button">
              New task
            </button>
          </div>
        </section>

        <section className="filters-card">
          <div className="filters-card__header">
            <div>
              <p className="placeholder-card__eyebrow">Filters</p>
              <h3>Refine the task list</h3>
            </div>
            {tasksLoading ? <span className="inline-hint">Refreshing tasks...</span> : null}
          </div>

          <div className="filters-grid">
            <label className="field">
              <span>Status</span>
              <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
                <option value="">All statuses</option>
                <option value="todo">Todo</option>
                <option value="in_progress">In Progress</option>
                <option value="done">Done</option>
              </select>
            </label>

            <label className="field field--inline">
              <input
                type="checkbox"
                checked={!!assigneeFilter}
                onChange={(event) =>
                  setAssigneeFilter(event.target.checked ? currentUserId : "")
                }
              />
              <span>Show only my tasks</span>
            </label>
          </div>
        </section>

        {feedback ? <div className="form-message form-message--error">{feedback}</div> : null}

        <section className="task-list">
          {tasks.length === 0 ? (
            <div className="placeholder-card">
              <p className="placeholder-card__eyebrow">No tasks</p>
              <h2>No tasks match this filter.</h2>
              <p>Try clearing the filters or create a new task for this project.</p>
            </div>
          ) : (
            tasks.map((task) => (
              <TaskCard
                key={task.id}
                currentUserId={user?.id ?? ""}
                onDelete={handleDeleteTask}
                onEdit={(selectedTask) => setTaskModal({ open: true, task: selectedTask })}
                onStatusChange={handleOptimisticStatusChange}
                projectOwnerId={project.owner_id}
                task={task}
              />
            ))
          )}
        </section>
      </main>

      <TaskModal
        open={taskModal.open}
        onClose={() => setTaskModal({ open: false, task: undefined })}
        onSaved={handleTaskSaved}
        projectId={project.id}
        task={taskModal.task}
      />
    </div>
  );
}
