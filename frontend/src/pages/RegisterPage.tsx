import { zodResolver } from "@hookform/resolvers/zod";
import axios from "axios";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { z } from "zod";

import { authApi, getApiErrorMessage } from "api/client";
import { useAuthStore } from "store/auth";

const registerSchema = z.object({
  name: z.string().min(1, "Name is required"),
  email: z.string().email("Enter a valid email"),
  password: z.string().min(8, "Password must be at least 8 characters"),
});

type RegisterFormValues = z.infer<typeof registerSchema>;

export default function RegisterPage() {
  const navigate = useNavigate();
  const token = useAuthStore((state) => state.token);
  const setAuth = useAuthStore((state) => state.setAuth);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({
    resolver: zodResolver(registerSchema),
  });

  if (token) {
    return <Navigate to="/projects" replace />;
  }

  async function onSubmit(values: RegisterFormValues) {
    setErrorMessage(null);

    try {
      const response = await authApi.register(values);
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
      <div className="auth-shell auth-shell--register">
        <section className="auth-card">
          <div className="auth-card__header">
            <p className="auth-card__eyebrow">Get started</p>
            <h2>Create your account</h2>
            <p>Set up a fresh TaskFlow workspace in under a minute.</p>
          </div>

          <form className="auth-form" onSubmit={handleSubmit(onSubmit)}>
            <label className="field">
              <span>Name</span>
              <input autoComplete="name" placeholder="Your full name" {...register("name")} />
              {errors.name ? <small>{errors.name.message}</small> : null}
            </label>

            <label className="field">
              <span>Email</span>
              <input autoComplete="email" placeholder="you@example.com" {...register("email")} />
              {errors.email ? <small>{errors.email.message}</small> : null}
            </label>

            <label className="field">
              <span>Password</span>
              <input
                autoComplete="new-password"
                placeholder="At least 8 characters"
                type="password"
                {...register("password")}
              />
              {errors.password ? <small>{errors.password.message}</small> : null}
            </label>

            {errorMessage ? <div className="form-message form-message--error">{errorMessage}</div> : null}

            <button className="button button--primary" disabled={isSubmitting} type="submit">
              {isSubmitting ? "Creating account..." : "Register"}
            </button>
          </form>

          <p className="auth-card__footer">
            Already have an account? <Link to="/login">Back to login</Link>
          </p>
        </section>

        <section className="hero-card hero-card--compact">
          <p className="hero-card__tag">Why TaskFlow</p>
          <h1>One clean place for projects, tasks, and ownership.</h1>
          <p>
            Designed for quick handoffs, clear priorities, and less time hunting for what matters
            next.
          </p>
        </section>
      </div>
    </div>
  );
}
