import { Navigate, Outlet } from "react-router-dom";

import { useAuthStore } from "store/auth";

export default function ProtectedRoute() {
  const token = useAuthStore((state) => state.token);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
