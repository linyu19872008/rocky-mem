export type ThemeMode = "system" | "light" | "dark";
export type LocaleId = "en" | "zh-CN";

export type RockyBoot = {
  locale?: string;
  themeHint?: string;
};

declare global {
  interface Window {
    __ROCKY_BOOT__?: RockyBoot;
  }
}

export const THEME_KEY = "rocky_mem_theme";
export const LOCALE_COOKIE = "rocky_mem_locale";

export function readBoot(): RockyBoot {
  return window.__ROCKY_BOOT__ ?? {};
}

export function normalizeLocale(raw?: string | null): LocaleId {
  if (!raw) return "en";
  const t = raw.toLowerCase();
  if (t.startsWith("zh")) return "zh-CN";
  return "en";
}

export function resolveTheme(mode: ThemeMode): "light" | "dark" {
  if (mode === "light" || mode === "dark") return mode;
  if (typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches) {
    return "dark";
  }
  return "light";
}

export function readStoredTheme(): ThemeMode {
  try {
    const v = localStorage.getItem(THEME_KEY);
    if (v === "light" || v === "dark" || v === "system") return v;
  } catch {
    /* ignore */
  }
  return "system";
}

export function applyTheme(mode: ThemeMode) {
  const resolved = resolveTheme(mode);
  document.documentElement.dataset.theme = resolved;
  document.documentElement.style.colorScheme = resolved;
  try {
    localStorage.setItem(THEME_KEY, mode);
  } catch {
    /* ignore */
  }
}

export function setLocaleCookie(locale: LocaleId) {
  document.cookie = `${LOCALE_COOKIE}=${encodeURIComponent(locale)}; path=/; SameSite=Strict; max-age=31536000`;
}

export function readLocaleCookie(): string | null {
  const m = document.cookie.match(/(?:^|; )rocky_mem_locale=([^;]*)/);
  return m ? decodeURIComponent(m[1]) : null;
}
