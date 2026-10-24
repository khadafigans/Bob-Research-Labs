// Supported locales and default. English is the only UI language for Bob Research Labs.
export const LOCALES = ["en"] as const;
export type Locale = (typeof LOCALES)[number];
export const DEFAULT_LOCALE: Locale = "en";

// Resolve the active locale at build time. Must be compatible with static export
// (output: "export" in next.config), so we only read environment variables without
// using dynamic APIs like cookies() or headers(). Falls back to "en" if NEXT_PUBLIC_LOCALE
// is empty or not in the supported list.
export function resolveLocale(): Locale {
  const raw = process.env.NEXT_PUBLIC_LOCALE;
  return LOCALES.includes(raw as Locale) ? (raw as Locale) : DEFAULT_LOCALE;
}
