import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { IntlProvider } from "react-intl";
import en from "../../locales/en.json";
import zhCN from "../../locales/zh-CN.json";
import {
  LOCALE_COOKIE,
  normalizeLocale,
  readBoot,
  readLocaleCookie,
  setLocaleCookie,
  type LocaleId,
} from "./boot";

const catalogs: Record<LocaleId, Record<string, string>> = {
  en,
  "zh-CN": zhCN,
};

type LocaleCtx = {
  locale: LocaleId;
  setLocale: (l: LocaleId) => void;
};

const Ctx = createContext<LocaleCtx>({ locale: "en", setLocale: () => {} });

function initialLocale(): LocaleId {
  const boot = readBoot();
  if (boot.locale) return normalizeLocale(boot.locale);
  const cookie = readLocaleCookie();
  if (cookie) return normalizeLocale(cookie);
  if (typeof navigator !== "undefined") return normalizeLocale(navigator.language);
  return "en";
}

export function LocaleProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<LocaleId>(initialLocale);

  const setLocale = useCallback((l: LocaleId) => {
    setLocaleCookie(l);
    setLocaleState(l);
    document.documentElement.lang = l;
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value = useMemo(() => ({ locale, setLocale }), [locale, setLocale]);

  return (
    <Ctx.Provider value={value}>
      <IntlProvider locale={locale} messages={catalogs[locale]} defaultLocale="en" onError={() => {}}>
        {children}
      </IntlProvider>
    </Ctx.Provider>
  );
}

export function useLocale() {
  return useContext(Ctx);
}

export { LOCALE_COOKIE };
