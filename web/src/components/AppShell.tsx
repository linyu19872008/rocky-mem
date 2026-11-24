import { useState } from "react";
import { NavLink, Outlet, useLocation } from "react-router-dom";
import { FormattedMessage, useIntl } from "react-intl";
import { LocaleSelect } from "./LocaleSelect";
import { ThemeToggle } from "./ThemeToggle";
import { Button } from "./Button";
import "./ui.css";

const APP_VERSION = "0.1.0";

export function AppShell() {
  const { pathname } = useLocation();
  const intl = useIntl();
  const [navOpen, setNavOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const wide = pathname.startsWith("/entry") || pathname.startsWith("/review");
  const narrow = pathname.startsWith("/settings");
  const innerClass = ["work-inner", wide ? "wide" : "", narrow ? "narrow" : ""].filter(Boolean).join(" ");

  const closeNav = () => setNavOpen(false);

  return (
    <div className={`app-shell ${navOpen ? "nav-open" : ""}`}>
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <header className="top-bar">
        <div className="top-bar-left">
          <Button className="ghost sm nav-toggle" onClick={() => setNavOpen((v) => !v)} aria-label="Menu">
            ☰
          </Button>
          <NavLink to="/" className="brand-compact" end onClick={closeNav}>
            <FormattedMessage id="nav.brand" />
          </NavLink>
        </div>
        <div className="top-bar-right">
          <span className="pill">v{APP_VERSION}</span>
          <div className="top-tools-wide">
            <LocaleSelect />
            <ThemeToggle />
          </div>
          <div className="top-tools-menu">
            <Button className="ghost sm" onClick={() => setMenuOpen((v) => !v)} aria-expanded={menuOpen}>
              ⋯
            </Button>
            {menuOpen ? (
              <div className="menu-pop">
                <LocaleSelect />
                <ThemeToggle />
              </div>
            ) : null}
          </div>
          <span className="pill">
            <FormattedMessage id="shell.local" />
          </span>
        </div>
      </header>

      <div className="nav-backdrop" onClick={closeNav} aria-hidden />

      <nav className="side-nav" aria-label={intl.formatMessage({ id: "shell.nav" })}>
        <div className="nav-group">
          <div className="nav-group-label">
            <FormattedMessage id="nav.group.work" />
          </div>
          <NavLink to="/" end onClick={closeNav}>
            <FormattedMessage id="nav.search" />
          </NavLink>
          <NavLink to="/library" onClick={closeNav}>
            <FormattedMessage id="nav.library" />
          </NavLink>
          <NavLink to="/review" onClick={closeNav}>
            <FormattedMessage id="nav.review" />
          </NavLink>
          <NavLink to="/entry" onClick={closeNav}>
            <FormattedMessage id="nav.entry" />
          </NavLink>
        </div>
        <div className="nav-group">
          <div className="nav-group-label">
            <FormattedMessage id="nav.group.ingest" />
          </div>
          <NavLink to="/ingest" onClick={closeNav}>
            <FormattedMessage id="nav.ingest" />
          </NavLink>
        </div>
        <div className="nav-group">
          <div className="nav-group-label">
            <FormattedMessage id="nav.group.system" />
          </div>
          <NavLink to="/overview" onClick={closeNav}>
            <FormattedMessage id="nav.overview" />
          </NavLink>
          <NavLink to="/settings" onClick={closeNav}>
            <FormattedMessage id="nav.settings" />
          </NavLink>
        </div>
      </nav>

      <div className="work-area">
        <main id="main" className={innerClass}>
          <Outlet />
        </main>
      </div>
    </div>
  );
}
