import { getRequestConfig } from "next-intl/server";

import { type Locale, resolveLocale } from "./config";

// Locale-specific message loaders. Use an explicit import map to prevent bundlers
// from pulling in the entire messages/ directory as a context module (excludes zh.sources.json which only exists locally).
const loaders: Record<Locale, () => Promise<{ default: Record<string, unknown> }>> = {
  en: () => import("../../messages/en.json"),
  ko: () => import("../../messages/ko.json"),
  zh: () => import("../../messages/zh.json"),
};

// locale    . requestLocale     
//  ,   messages/zh.json  ,  messages/ko.json  .
export default getRequestConfig(async () => {
  const locale = resolveLocale();
  const messages = (await loaders[locale]()).default;
  return { locale, messages };
});
