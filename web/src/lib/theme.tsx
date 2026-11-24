import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { applyTheme, readStoredTheme, type ThemeMode } from "./boot";

type ThemeCtx = {
  mode: ThemeMode;
  setMode: (m: ThemeMode) => void;
  cycle: () => void;
};

const Ctx = createContext<ThemeCtx>({
  mode: "system",
  setMode: () => {},
  cycle: () => {},
});

const order: ThemeMode[] = ["system", "light", "dark"];

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>(() => readStoredTheme());

  useEffect(() => {
    applyTheme(mode);
    if (mode !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => applyTheme("system");
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [mode]);

  const setMode = useCallback((m: ThemeMode) => {
    setModeState(m);
    applyTheme(m);
  }, []);

  const cycle = useCallback(() => {
    setModeState((cur) => {
      const next = order[(order.indexOf(cur) + 1) % order.length];
      applyTheme(next);
      return next;
    });
  }, []);

  const value = useMemo(() => ({ mode, setMode, cycle }), [mode, setMode, cycle]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useTheme() {
  return useContext(Ctx);
}
