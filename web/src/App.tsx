import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { AppShell } from "./components/AppShell";
import { LocaleProvider } from "./lib/i18n";
import { ThemeProvider } from "./lib/theme";
import { SearchPage } from "./pages/SearchPage";
import { ReviewPage } from "./pages/ReviewPage";
import { EntryPage } from "./pages/EntryPage";
import { SettingsPage } from "./pages/SettingsPage";
import { OverviewPage } from "./pages/OverviewPage";
import { IngestPage } from "./pages/IngestPage";
import { LibraryPage } from "./pages/LibraryPage";
import { RememberPage } from "./pages/RememberPage";
import { RecordPage } from "./pages/RecordPage";

export function App() {
  return (
    <ThemeProvider>
      <LocaleProvider>
        <BrowserRouter>
          <Routes>
            <Route element={<AppShell />}>
              <Route index element={<SearchPage />} />
              <Route path="library" element={<LibraryPage />} />
              <Route path="remember" element={<RememberPage />} />
              <Route path="record" element={<RecordPage />} />
              <Route path="review" element={<ReviewPage />} />
              <Route path="entry" element={<EntryPage />} />
              <Route path="settings" element={<SettingsPage />} />
              <Route path="overview" element={<OverviewPage />} />
              <Route path="ingest" element={<IngestPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </LocaleProvider>
    </ThemeProvider>
  );
}
