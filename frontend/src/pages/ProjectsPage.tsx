import Navbar from "components/Navbar";

export default function ProjectsPage() {
  return (
    <div className="app-layout">
      <Navbar />

      <main className="page-shell">
        <section className="placeholder-card">
          <p className="placeholder-card__eyebrow">App shell ready</p>
          <h2>Projects UI comes next</h2>
          <p>
            Auth, routing, persisted session state, and the protected application shell are ready.
            The Projects and Tasks screens will be added in the next frontend phase.
          </p>
        </section>
      </main>
    </div>
  );
}
