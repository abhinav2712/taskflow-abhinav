import { useEffect, useState } from "react";
import type { FormEvent } from "react";

import { getApiErrorMessage, tasksApi, usersApi } from "api/client";
import { useAuthStore } from "store/auth";
import type { Task, TaskPriority, TaskStatus, User } from "types";

interface TaskModalProps {
  open: boolean;
  onClose: () => void;
  projectId: string;
  task?: Task;
  onSaved: (task: Task) => void;
}

export default function TaskModal({ open, onClose, projectId, task, onSaved }: TaskModalProps) {
  const isEditMode = Boolean(task);
  const currentUser = useAuthStore((state) => state.user);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<TaskStatus>("todo");
  const [priority, setPriority] = useState<TaskPriority>("medium");
  const [assigneeId, setAssigneeId] = useState("");
  const [dueDate, setDueDate] = useState("");
  const [users, setUsers] = useState<User[]>([]);
  const [usersLoading, setUsersLoading] = useState(false);
  const [usersError, setUsersError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const modalTitle = isEditMode ? "Edit task" : "Create task";

  useEffect(() => {
    if (!open) {
      return;
    }

    setTitle(task?.title ?? "");
    setDescription(task?.description ?? "");
    setStatus(task?.status ?? "todo");
    setPriority(task?.priority ?? "medium");
    setAssigneeId(task?.assignee_id ?? "");
    setDueDate(task?.due_date ?? "");
    setFieldErrors({});
    setError(null);
    setSubmitting(false);
  }, [open, task]);

  useEffect(() => {
    if (!open) {
      return;
    }

    let isCancelled = false;

    async function loadUsers() {
      setUsersLoading(true);
      setUsersError(null);

      try {
        const response = await usersApi.list();
        if (!isCancelled) {
          setUsers(response.users);
        }
      } catch (loadError) {
        if (!isCancelled) {
          const apiError = getApiErrorMessage(loadError);
          setUsersError(apiError.error);
          setUsers([]);
        }
      } finally {
        if (!isCancelled) {
          setUsersLoading(false);
        }
      }
    }

    void loadUsers();

    return () => {
      isCancelled = true;
    };
  }, [open]);

  if (!open) {
    return null;
  }

  const meOption = currentUser ? { value: currentUser.id, label: `Me (${currentUser.name})` } : null;
  const otherUsers = users.filter((user) => user.id !== currentUser?.id);
  const assigneeExists =
    assigneeId === "" ||
    users.some((user) => user.id === assigneeId) ||
    currentUser?.id === assigneeId;

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFieldErrors({});
    setError(null);
    setSubmitting(true);

    try {
      const payload = {
        title: title.trim(),
        description: description.trim() || undefined,
        priority,
        assignee_id: assigneeId.trim() || undefined,
        due_date: dueDate || undefined,
        ...(isEditMode ? { status } : {}),
      };

      const savedTask = isEditMode
        ? await tasksApi.update(task!.id, payload)
        : await tasksApi.create(projectId, payload);

      onSaved(savedTask);
      onClose();
    } catch (submitError) {
      const apiError = getApiErrorMessage(submitError);
      if (apiError.fields) {
        setFieldErrors(apiError.fields);
      } else {
        setError(apiError.error);
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="modal-backdrop" onClick={onClose} role="presentation">
      <div
        className="modal-card modal-card--wide"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <div className="modal-card__header">
          <div>
            <p className="auth-card__eyebrow">{isEditMode ? "Update task" : "New task"}</p>
            <h2>{modalTitle}</h2>
          </div>
          <button className="button button--ghost" onClick={onClose} type="button">
            Close
          </button>
        </div>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="field">
            <span>Title</span>
            <input
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="Design homepage"
            />
            {fieldErrors.title ? <small>{fieldErrors.title}</small> : null}
          </label>

          <label className="field">
            <span>Description</span>
            <textarea
              className="field__textarea"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="Add more context for the team"
              rows={4}
            />
          </label>

          <div className="form-grid">
            {isEditMode ? (
              <label className="field">
                <span>Status</span>
                <select value={status} onChange={(event) => setStatus(event.target.value as TaskStatus)}>
                  <option value="todo">Todo</option>
                  <option value="in_progress">In Progress</option>
                  <option value="done">Done</option>
                </select>
                {fieldErrors.status ? <small>{fieldErrors.status}</small> : null}
              </label>
            ) : null}

            <label className="field">
              <span>Priority</span>
              <select value={priority} onChange={(event) => setPriority(event.target.value as TaskPriority)}>
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
              </select>
              {fieldErrors.priority ? <small>{fieldErrors.priority}</small> : null}
            </label>

            <label className="field">
              <span>Assignee</span>
              <select value={assigneeId} onChange={(event) => setAssigneeId(event.target.value)}>
                <option value="">Unassigned</option>
                {meOption ? <option value={meOption.value}>{meOption.label}</option> : null}
                {otherUsers.map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.name} ({user.email})
                  </option>
                ))}
                {!assigneeExists && assigneeId ? (
                  <option value={assigneeId}>Current assignee</option>
                ) : null}
              </select>
              {usersLoading ? <small>Loading assignees...</small> : null}
              {usersError ? <small>{usersError}</small> : null}
              {fieldErrors.assignee_id || fieldErrors.assignee ? (
                <small>{fieldErrors.assignee_id ?? fieldErrors.assignee}</small>
              ) : null}
            </label>

            <label className="field">
              <span>Due date</span>
              <input value={dueDate} onChange={(event) => setDueDate(event.target.value)} type="date" />
              {fieldErrors.due_date ? <small>{fieldErrors.due_date}</small> : null}
            </label>
          </div>

          {error ? <div className="form-message form-message--error">{error}</div> : null}

          <div className="modal-card__actions">
            <button className="button button--ghost" onClick={onClose} type="button">
              Cancel
            </button>
            <button className="button button--primary" disabled={submitting} type="submit">
              {submitting ? "Saving..." : isEditMode ? "Save changes" : "Create task"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
