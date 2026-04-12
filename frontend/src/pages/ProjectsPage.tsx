import { useEffect, useState } from "react";

import { getApiErrorMessage, projectsApi } from "api/client";
import CreateProjectModal from "components/CreateProjectModal";
import Navbar from "components/Navbar";
import ProjectCard from "components/ProjectCard";
import type { Project } from "types";

export default function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function loadProjects() {
      setLoading(true);
      setError(null);

      try {
        const response = await projectsApi.list();
        if (!cancelled) {
          setProjects(response.projects);
        }
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

    void loadProjects();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="app-layout">
      <Navbar />

      <main className="page-shell">
        <section className="projects-header">
          <div>
            <p className="placeholder-card__eyebrow">Projects</p>
            <h2>Your active workspace</h2>
            <p>Open a project, review the latest tasks, or create a new one in seconds.</p>
          </div>
          <button className="button button--primary" onClick={() => setShowModal(true)} type="button">
            Create project
          </button>
        </section>

        {loading ? (
          <section className="projects-grid">
            {Array.from({ length: 3 }).map((_, index) => (
              <div className="project-card project-card--skeleton" key={index}>
                <div className="skeleton-line skeleton-line--short" />
                <div className="skeleton-line" />
                <div className="skeleton-line" />
              </div>
            ))}
          </section>
        ) : error ? (
          <section className="placeholder-card">
            <p className="placeholder-card__eyebrow">Couldn&apos;t load projects</p>
            <h2>{error}</h2>
            <p>Check that the backend is running and your token is still valid.</p>
          </section>
        ) : projects.length === 0 ? (
          <section className="placeholder-card">
            <p className="placeholder-card__eyebrow">No projects yet</p>
            <h2>Create one to get started.</h2>
            <p>Your projects will appear here once you add the first one.</p>
          </section>
        ) : (
          <section className="projects-grid">
            {projects.map((project) => (
              <ProjectCard key={project.id} project={project} />
            ))}
          </section>
        )}
      </main>

      <CreateProjectModal
        open={showModal}
        onClose={() => setShowModal(false)}
        onCreated={(project) => setProjects((currentProjects) => [project, ...currentProjects])}
      />
    </div>
  );
}
