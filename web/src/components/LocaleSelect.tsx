import { useIntl } from "react-intl";
import { useLocale } from "../lib/i18n";
import type { LocaleId } from "../lib/boot";
import { SegmentedControl } from "./SegmentedControl";

export function LocaleSelect() {
  const { locale, setLocale } = useLocale();
  const intl = useIntl();
  return (
    <SegmentedControl<LocaleId>
      label={intl.formatMessage({ id: "locale.label" })}
      value={locale}
      onChange={setLocale}
      options={[
        { value: "en", label: "EN" },
        { value: "zh-CN", label: intl.formatMessage({ id: "locale.zhCNShort" }) },
      ]}
    />
  );
}
