import { create } from "zustand";
import { persist } from "zustand/middleware";

interface ThemeState {
  dark: boolean;
  toggle: () => void;
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      dark: false,
      toggle: () => set((state) => ({ dark: !state.dark })),
    }),
    { name: "theme-preference" },
  ),
);

/**
 * Apply the current theme to <html> on startup, before React renders.
 * Called once from main.tsx to avoid a flash of wrong theme on load.
 */
export function applyStoredTheme() {
  try {
    const raw = localStorage.getItem("theme-preference");
    const stored = raw ? (JSON.parse(raw) as { state?: { dark?: boolean } }) : null;
    if (stored?.state?.dark) {
      document.documentElement.classList.add("dark");
    }
  } catch {
    // silently ignore parse errors
  }
}
