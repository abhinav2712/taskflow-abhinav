import { Navigate, Route, Routes } from "react-router-dom";

import ProtectedRoute from "components/ProtectedRoute";
import LoginPage from "pages/LoginPage";
import NotFoundPage from "pages/NotFoundPage";
import ProjectsPage from "pages/ProjectsPage";
import RegisterPage from "pages/RegisterPage";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/projects" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/register" element={<RegisterPage />} />
      <Route element={<ProtectedRoute />}>
        <Route path="/projects" element={<ProjectsPage />} />
      </Route>
      <Route path="*" element={<NotFoundPage />} />
    </Routes>
  );
}
