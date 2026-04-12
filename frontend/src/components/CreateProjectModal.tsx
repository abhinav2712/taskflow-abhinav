import { useEffect, useState } from "react";
import type { FormEvent } from "react";

import { getApiErrorMessage, projectsApi } from "api/client";
import type { Project } from "types";

interface CreateProjectModalProps {
  open: boolean;
  onClose: () => void;
  onCreated: (project: Project) => void;
}

export default function CreateProjectModal({
  open,
  onClose,
  onCreated,
}: CreateProjectModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [fieldError, setFieldError] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) {
      setName("");
      setDescription("");
      setFieldError(null);
      setError(null);
      setSubmitting(false);
    }
  }, [open]);

  if (!open) {
    return null;
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFieldError(null);
    setError(null);
    setSubmitting(true);

    try {
      const project = await projectsApi.create({
        name: name.trim(),
        description: description.trim() || undefined,
      });
      onCreated(project);
      onClose();
    } catch (submitError) {
      const apiError = getApiErrorMessage(submitError);
      if (apiError.fields?.name) {
        setFieldError(apiError.fields.name);
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
        className="modal-card"
        onClick={(event) => event.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <div className="modal-card__header">
          <div>
            <p className="auth-card__eyebrow">New project</p>
            <h2>Create a project</h2>
          </div>
          <button className="button button--ghost" onClick={onClose} type="button">
            Close
          </button>
        </div>

        <form className="auth-form" onSubmit={handleSubmit}>
          <label className="field">
            <span>Name</span>
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Website redesign"
            />
            {fieldError ? <small>{fieldError}</small> : null}
          </label>

          <label className="field">
            <span>Description</span>
            <textarea
              className="field__textarea"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              placeholder="What is this project for?"
              rows={4}
            />
          </label>

          {error ? <div className="form-message form-message--error">{error}</div> : null}

          <div className="modal-card__actions">
            <button className="button button--ghost" onClick={onClose} type="button">
              Cancel
            </button>
            <button className="button button--primary" disabled={submitting} type="submit">
              {submitting ? "Creating..." : "Create project"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
