import type { Task, TaskStatus } from "types";

interface TaskCardProps {
  task: Task;
  assigneeLabel?: string | null;
  projectOwnerId: string;
  currentUserId: string;
  onStatusChange: (id: string, status: TaskStatus) => void;
  onEdit: (task: Task) => void;
  onDelete: (task: Task) => void;
}

const priorityLabelMap = {
  low: "Low priority",
  medium: "Medium priority",
  high: "High priority",
} as const;

export default function TaskCard({
  task,
  assigneeLabel,
  projectOwnerId,
  currentUserId,
  onStatusChange,
  onEdit,
  onDelete,
}: TaskCardProps) {
  const canDelete = task.creator_id === currentUserId || projectOwnerId === currentUserId;

  return (
    <article className="task-card">
      <div className="task-card__top">
        <div>
          <h3>{task.title}</h3>
          <p>{task.description?.trim() ? task.description : "No description provided."}</p>
        </div>
        <span className={`status-badge ${statusClassName(task.status)}`}>{statusLabel(task.status)}</span>
      </div>

      <div className="task-card__meta">
        <span className={`pill pill--priority-${task.priority}`}>{priorityLabelMap[task.priority]}</span>

        <span className="task-chip">
          {/* person icon */}
          <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true">
            <path d="M8 8a3 3 0 1 0 0-6 3 3 0 0 0 0 6Zm-5 6a5 5 0 0 1 10 0H3Z" fill="currentColor"/>
          </svg>
          {assigneeLabel ?? "Unassigned"}
        </span>

        {task.due_date ? (
          <span className="task-chip task-chip--date">
            {/* calendar icon */}
            <svg width="13" height="13" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <rect x="1" y="3" width="14" height="12" rx="2" stroke="currentColor" strokeWidth="1.5"/>
              <path d="M1 7h14M5 1v4M11 1v4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/>
            </svg>
            {task.due_date}
          </span>
        ) : (
          <span className="task-chip task-chip--muted">No due date</span>
        )}
      </div>

      <div className="task-card__actions">
        <label className="inline-select">
          <span>Status</span>
          <select
            value={task.status}
            onChange={(event) => onStatusChange(task.id, event.target.value as TaskStatus)}
          >
            <option value="todo">Todo</option>
            <option value="in_progress">In Progress</option>
            <option value="done">Done</option>
          </select>
        </label>

        <div className="task-card__buttons">
          <button className="button button--ghost" onClick={() => onEdit(task)} type="button">
            Edit
          </button>
          {canDelete ? (
            <button className="button button--danger" onClick={() => onDelete(task)} type="button">
              Delete
            </button>
          ) : null}
        </div>
      </div>
    </article>
  );
}

function statusLabel(status: TaskStatus) {
  if (status === "in_progress") {
    return "In Progress";
  }

  if (status === "done") {
    return "Done";
  }

  return "Todo";
}

function statusClassName(status: TaskStatus) {
  if (status === "in_progress") {
    return "status-badge--progress";
  }

  if (status === "done") {
    return "status-badge--done";
  }

  return "status-badge--todo";
}
