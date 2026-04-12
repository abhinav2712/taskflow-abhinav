import { useNavigate } from "react-router-dom";

import { useAuthStore } from "store/auth";

export default function Navbar() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const clearAuth = useAuthStore((state) => state.clearAuth);

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
        <button className="button button--ghost" onClick={handleLogout} type="button">
          Logout
        </button>
      </div>
    </header>
  );
}
