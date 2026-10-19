import createNextIntlPlugin from "next-intl/plugin";

import { fileURLToPath } from "node:url";

// ：`NEXT_EXPORT=1 next build`  web/out，
// nginx web 。(next dev)， /api 。
const isExport = process.env.NEXT_EXPORT === "1";
// Vercel demo： mock，， /api 。
const isMock = process.env.NEXT_PUBLIC_MOCK === "1";

/** @type {import('next').NextConfig} */
const nextConfig = {
  turbopack: { root: fileURLToPath(new URL(".", import.meta.url)) },
  reactCompiler: true,
  // dev  IPv4  /_next/*  HMR（ IP ）。
  allowedDevOrigins: ["*.*.*.*"],
  compiler: {
    removeConsole: process.env.NODE_ENV === "production",
  },
  ...(isExport
    ? {
        // ： Node ；； <route>/index.html。
        output: "export",
        images: { unoptimized: true },
        trailingSlash: true,
      }
    : isMock
      ? {
          // Vercel mock demo：， /api 。
          images: { unoptimized: true },
        }
      : {
          // ： /api/*  Go （ :8787， AUTOPENTEST_API ）。
          async rewrites() {
            const backend = process.env.AUTOPENTEST_API ?? "http://localhost:8787";
            return [{ source: "/api/:path*", destination: `${backend}/api/:path*` }];
          },
        }),
};

// next-intl .   src/i18n/request.ts  . i18n  
//     (  output: "export"  ).
const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

export default withNextIntl(nextConfig);
