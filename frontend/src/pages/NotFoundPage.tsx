import { Link } from "react-router-dom";

export default function NotFoundPage() {
  return (
    <div className="not-found-layout">
      <div className="not-found-card">
        <p className="auth-card__eyebrow">404</p>
        <h1>Page not found</h1>
        <p>The page you were looking for doesn&apos;t exist or has moved.</p>
        <Link className="button button--primary" to="/projects">
          Go to app
        </Link>
      </div>
    </div>
  );
}
