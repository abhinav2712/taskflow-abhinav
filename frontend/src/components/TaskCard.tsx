import type { Task, TaskStatus } from "types";

interface TaskCardProps {
  task: Task;
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
        <span>{task.assignee_id ? `Assignee ${task.assignee_id}` : "Unassigned"}</span>
        <span>{task.due_date ? `Due ${task.due_date}` : "No due date"}</span>
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
