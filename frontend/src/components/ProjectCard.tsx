import { Link } from "react-router-dom";

import type { Project } from "types";

interface ProjectCardProps {
  project: Project;
}

export default function ProjectCard({ project }: ProjectCardProps) {
  return (
    <Link className="project-card" to={`/projects/${project.id}`}>
      <div className="project-card__header">
        <p className="project-card__eyebrow">Project</p>
        <span className="project-card__arrow">Open</span>
      </div>
      <h3>{project.name}</h3>
      <p>{project.description?.trim() ? project.description : "No description added yet."}</p>
      <div className="project-card__footer">
        <span>Created {new Date(project.created_at).toLocaleDateString()}</span>
        <span>Updated {new Date(project.updated_at).toLocaleDateString()}</span>
      </div>
    </Link>
  );
}
