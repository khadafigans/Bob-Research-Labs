import { resolveLocale } from "@/i18n/config";

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

//   ·  . layout.tsx   metadata   
//  <title>·<meta name="description">  ,  locale  
const META_BY_LOCALE = {
  en: {
    title: "Bob Research Labs: We're coming to root your server!",
    description: "Autonomous penetration testing system powered by LLM - We are coming to root your server",
  },
  ko: {
    title: "Bob Research Labs: We're coming to root your server!",
    description: "LLM      - We are coming to root your server",
  },
  zh: {
    title: "Bob Research Labs: We're coming to root your server!",
    description: "LLM  - We are coming to root your server",
  },
} as const;

export const APP_CONFIG = {
  name: "Bob Research Labs",
  version: packageJson.version,
  copyright: `© ${currentYear} Powered by Bob Research Labs`,
  meta: META_BY_LOCALE[resolveLocale()] || META_BY_LOCALE.en,
};
