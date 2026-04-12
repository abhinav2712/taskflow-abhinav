import { zodResolver } from "@hookform/resolvers/zod";
import axios from "axios";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { z } from "zod";

import { authApi, getApiErrorMessage } from "api/client";
import { useAuthStore } from "store/auth";

const loginSchema = z.object({
  email: z.string().email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
});

type LoginFormValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const navigate = useNavigate();
  const token = useAuthStore((state) => state.token);
  const setAuth = useAuthStore((state) => state.setAuth);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "test@example.com",
      password: "password123",
    },
  });

  if (token) {
    return <Navigate to="/projects" replace />;
  }

  async function onSubmit(values: LoginFormValues) {
    setErrorMessage(null);

    try {
      const response = await authApi.login(values);
      setAuth(response.token, response.user);
      navigate("/projects", { replace: true });
    } catch (error) {
      const apiError = getApiErrorMessage(error);

      if (axios.isAxiosError(error) && apiError.fields) {
        setErrorMessage(Object.values(apiError.fields)[0]);
        return;
      }

      setErrorMessage(apiError.error);
    }
  }

  return (
    <div className="auth-layout">
      <div className="auth-shell">
        <section className="hero-card">
          <p className="hero-card__tag">TaskFlow</p>
          <h1>Stay on top of team work with a calmer, faster flow.</h1>
          <p>
            Clean project tracking, thoughtful task ownership, and a dashboard that feels light
            instead of noisy.
          </p>
          <div className="badge-row">
            <span className="status-badge status-badge--todo">Todo</span>
            <span className="status-badge status-badge--progress">In Progress</span>
            <span className="status-badge status-badge--done">Done</span>
          </div>
        </section>

        <section className="auth-card">
          <div className="auth-card__header">
            <p className="auth-card__eyebrow">Welcome back</p>
            <h2>Login to your workspace</h2>
            <p>Use the seeded reviewer credentials or your registered account.</p>
          </div>

          <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
            <label className="field">
              <span>Email</span>
              <input autoComplete="email" placeholder="you@example.com" {...register("email")} />
              {errors.email ? <small>{errors.email.message}</small> : null}
            </label>

            <label className="field">
              <span>Password</span>
              <input
                autoComplete="current-password"
                placeholder="Enter your password"
                type="password"
                {...register("password")}
              />
              {errors.password ? <small>{errors.password.message}</small> : null}
            </label>

            {errorMessage ? <div className="form-message form-message--error">{errorMessage}</div> : null}

            <button className="button button--primary" disabled={isSubmitting} type="submit">
              {isSubmitting ? "Logging in..." : "Login"}
            </button>
          </form>

          <p className="auth-card__footer">
            New here? <Link to="/register">Create an account</Link>
          </p>
        </section>
      </div>
    </div>
  );
}
