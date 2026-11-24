import { useIntl } from "react-intl";
import { useTheme } from "../lib/theme";
import type { ThemeMode } from "../lib/boot";
import { SegmentedControl } from "./SegmentedControl";

export function ThemeToggle() {
  const { mode, setMode } = useTheme();
  const intl = useIntl();
  return (
    <SegmentedControl<ThemeMode>
      label={intl.formatMessage({ id: "theme.label" })}
      value={mode}
      onChange={setMode}
      options={[
        { value: "system", label: intl.formatMessage({ id: "theme.systemShort" }) },
        { value: "light", label: intl.formatMessage({ id: "theme.lightShort" }) },
        { value: "dark", label: intl.formatMessage({ id: "theme.darkShort" }) },
      ]}
    />
  );
}
