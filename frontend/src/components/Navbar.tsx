import { useNavigate } from "react-router-dom";

import { useAuthStore } from "store/auth";
import { useThemeStore } from "../store/theme";

export default function Navbar() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);
  const dark = useThemeStore((state) => state.dark);
  const toggleTheme = useThemeStore((state) => state.toggle);

  function handleLogout() {
    clearAuth();
    navigate("/login", { replace: true });
  }

  return (
    <header className="navbar">
      <div className="navbar__brand">
        <div className="navbar__logo">T</div>
        <div>
          <p className="navbar__eyebrow">TaskFlow</p>
          <h1 className="navbar__title">Team tasks, without the clutter</h1>
        </div>
      </div>

      <div className="navbar__actions">
        <div className="navbar__user">
          <span className="navbar__user-label">Signed in as</span>
          <strong>{user?.name ?? "User"}</strong>
        </div>

        <button
          className="theme-toggle"
          onClick={toggleTheme}
          type="button"
          aria-label={dark ? "Switch to light mode" : "Switch to dark mode"}
          title={dark ? "Light mode" : "Dark mode"}
        >
          {dark ? (
            /* Sun icon */
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <circle cx="12" cy="12" r="4" stroke="currentColor" strokeWidth="2"/>
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" stroke="currentColor" strokeWidth="2" strokeLinecap="round"/>
            </svg>
          ) : (
            /* Moon icon */
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path d="M21 12.79A9 9 0 1 1 11.21 3a7 7 0 0 0 9.79 9.79Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          )}
        </button>

        <button className="button button--ghost" onClick={handleLogout} type="button">
          Logout
        </button>
      </div>
    </header>
  );
}
