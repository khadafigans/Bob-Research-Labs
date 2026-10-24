import { resolveLocale } from "@/i18n/config";

import packageJson from "../../package.json";

const currentYear = new Date().getFullYear();

// 브라우저 탭 제목·검색엔진 메타 설명. layout.tsx 의 정적 metadata 로 들어가 모든
// 페이지의 <title>·<meta name="description"> 에 렌더되므로, 활성 locale 에 맞춰
// 한국어(ko)·중국어(zh)를 분기한다. zh 는 상류 대조를 위해 원문을 그대로 보존한다.
// ko 제목은 엠대시 대신 콜론을 써서 "제품명: 설명" 형태로 둔다.
const META_BY_LOCALE = {
  en: {
    title: "Bob Research Labs: We're coming to root your server!",
    description: "Autonomous penetration testing system powered by LLM - We are coming to root your server",
  },
  ko: {
    title: "Bob Research Labs: We're coming to root your server!",
    description: "LLM 기반 자율 침투 테스트 시스템 - We are coming to root your server",
  },
  zh: {
    title: "Bob Research Labs: We're coming to root your server!",
    description: "LLM 驱动的自主渗透测试系统 - We are coming to root your server",
  },
} as const;

export const APP_CONFIG = {
  name: "Bob Research Labs",
  version: packageJson.version,
  copyright: `© ${currentYear} Powered by Bob Research Labs`,
  meta: META_BY_LOCALE[resolveLocale()] || META_BY_LOCALE.en,
};
