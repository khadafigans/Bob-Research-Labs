import { getRequestConfig } from "next-intl/server";

import { type Locale, resolveLocale } from "./config";

// Locale-specific message loaders. English-only for Bob Research Labs — do not
// pull ko.json/zh.json (removed from distribution / CI tree).
const loaders: Record<Locale, () => Promise<{ default: Record<string, unknown> }>> = {
  en: () => import("../../messages/en.json"),
};

export default getRequestConfig(async () => {
  const locale = resolveLocale();
  const messages = (await loaders[locale]()).default;
  return { locale, messages };
});
