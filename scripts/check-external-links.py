#!/usr/bin/env python3
"""     (http·https)    .

  `check-doc-links.py`     ·   URL 
 (  flaky     ). 
  " " . README · ·detections README 
  "  "   (   boho.or.kr·
privacy.go.kr·pipc.go.kr·fsec.or.kr, CISA KEV, OWASP·SigmaHQ·Suricata·MITRE·MISP,
  artex-demo.vercel.app, GitHub  ) ··  
 ,  · .

  URL   
-   `.md`  , fenced code block(```  ~~~)   
  (`` `...` ``)  URL  .   URL   · · 
  (:    `id.redhaze.top`) "   " .
-  ·(`[text](url)`·`![alt](url)`),  (`<url>`),  
     URL  http·https  .
- ·  : localhost·/ IP(127.·10.·192.168.·
  169.254.·172.16~31.·0.0.0.0·::1), RFC 2606/6761 (example.com/org/net/edu·
  `*.example.*`·`.test`·`.invalid`·`.local`·`.tld`),   FQDN   (: `target`).

    (MAINTAINING.md 8.2     )
 ·   HEAD · User-Agent    
,       .   
** User-Agent , GET ,  ** .   
.
- OK:   2xx·3xx.  .
- RESTRICTED: 401·403·405·429.        ( ·
   · )      .    .
- DOWN: 404·410·5xx( )·DNS///SSL ( ). 
    .

 ·5xx·429          .
  .    **   **  URL 
GET     ,   · .

 (allowlist)
`scripts/external-links-allowlist.txt`   URL  DOWN   "ALLOWED"  
 strict    .      ,   
(:  CHANGELOG.zh.md)     ,  strict 
       **    **   .


- (·  0): `python3 -I scripts/check-external-links.py`
-    / (    ): `--strict` (allowlist 
   DOWN      1). RESTRICTED·ALLOWED  strict    .
-      : `--list` (    URL   ).
"""
import argparse
import http.client
import os
import re
import socket
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections import defaultdict
from urllib.parse import urlsplit

# [text](url)  ( ) ![alt](url) .  ·  .
MD_LINK = re.compile(r"(?<!\!)\[[^\]]*\]\(([^)\s]+)")
MD_IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
AUTOLINK = re.compile(r"<(https?://[^>\s]+)>")
BARE_URL = re.compile(r"https?://[^\s)>\]\"'`]+")
INLINE_CODE = re.compile(r"`[^`]*`")
TRAILING_PUNCT = ".,;:!?\"'»)]}>"

PRIVATE_IPV4 = re.compile(r"^(127\.|10\.|192\.168\.|169\.254\.|0\.0\.0\.0$)")
PRIVATE_IPV4_172 = re.compile(r"^172\.(1[6-9]|2\d|3[01])\.")

BROWSER_UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36"
)

RESTRICTED_CODES = {401, 403, 405, 429}

ALLOWLIST_PATH = os.path.join("scripts", "external-links-allowlist.txt")


def load_allowlist(root: str) -> set:
    """     DOWN URL  (  ).

      URL . `#`  ,  ·   .
    """
    path = os.path.join(root, ALLOWLIST_PATH)
    allow = set()
    if not os.path.exists(path):
        return allow
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            line = line.split("#", 1)[0].strip()
            if line:
                allow.add(line)
    return allow


def strip_code(line: str) -> str:
    """fence       (  URL   )."""
    return INLINE_CODE.sub(" ", line)


def is_checkable(url: str) -> bool:
    """·  ,      URL  ."""
    parts = urlsplit(url)
    if parts.scheme not in ("http", "https"):
        return False
    host = parts.hostname
    if not host:
        return False
    host = host.lower()
    if host == "localhost" or host.endswith(".localhost"):
        return False
    if host == "::1":
        return False
    if PRIVATE_IPV4.match(host) or PRIVATE_IPV4_172.match(host):
        return False
    if host in ("example.com", "example.org", "example.net", "example.edu"):
        return False
    if host.endswith((".example.com", ".example.org", ".example.net", ".example")):
        return False
    if host.endswith((".test", ".invalid", ".local", ".localdomain", ".tld")):
        return False
    if "." not in host:  # FQDN    (: target)
        return False
    return True


def extract(root: str, md_files: list) -> dict:
    """ .md    URL → [(, ), ...]  ."""
    found = defaultdict(list)
    for md in md_files:
        with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
            lines = fh.readlines()
        in_fence = False
        for lineno, raw in enumerate(lines, 1):
            stripped = raw.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            line = strip_code(raw)
            candidates = []
            for pattern in (MD_LINK, MD_IMAGE, AUTOLINK):
                candidates.extend(m.group(1) for m in pattern.finditer(line))
            for m in BARE_URL.finditer(line):
                candidates.append(m.group(0).rstrip(TRAILING_PUNCT))
            for url in candidates:
                url = url.strip().rstrip(TRAILING_PUNCT)
                if is_checkable(url):
                    where = (md, lineno)
                    if where not in found[url]:
                        found[url].append(where)
    return found


def probe(url: str, timeout: float, retries: int, delay: float):
    """URL   UA  GET   (, )  .

     "OK" · "RESTRICTED" · "DOWN".  ·5xx·429  .
    """
    req = urllib.request.Request(
        url,
        method="GET",
        headers={
            "User-Agent": BROWSER_UA,
            "Accept": "*/*",
            "Accept-Language": "ko,en;q=0.8",
        },
    )
    last = ""
    for attempt in range(retries + 1):
        try:
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                code = resp.getcode()
                if code in RESTRICTED_CODES:
                    return "RESTRICTED", f"HTTP {code}"
                return "OK", f"HTTP {code}"
        except urllib.error.HTTPError as exc:
            code = exc.code
            if code in RESTRICTED_CODES:
                return "RESTRICTED", f"HTTP {code}"
            if code >= 500 or code == 408:
                last = f"HTTP {code}"  #    
            else:
                return "DOWN", f"HTTP {code}"  # 404·410  ,  
        except (
            urllib.error.URLError,
            http.client.HTTPException,
            ssl.SSLError,
            socket.timeout,
            ConnectionError,
            TimeoutError,
            OSError,
        ) as exc:
            reason = getattr(exc, "reason", exc)
            last = f"{type(exc).__name__}: {reason}"
        if attempt < retries:
            time.sleep(delay * (attempt + 1))
    return "DOWN", last


def main() -> int:
    ap = argparse.ArgumentParser(description="     ")
    ap.add_argument("--strict", action="store_true",
                    help="DOWN      1 (/ )")
    ap.add_argument("--list", action="store_true",
                    help="     URL   ")
    ap.add_argument("--timeout", type=float, default=15.0, help=" ()")
    ap.add_argument("--retries", type=int, default=2, help=" ·5xx·429  ")
    ap.add_argument("--delay", type=float, default=0.5, help=" ·   ()")
    args = ap.parse_args()

    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    tracked = subprocess.check_output(["git", "ls-files"], cwd=root, text=True).splitlines()
    md_files = [f for f in tracked if f.endswith(".md")]

    allow = load_allowlist(root)
    found = extract(root, md_files)
    urls = sorted(found)
    print(f"  {len(md_files)}    URL {len(urls)} ")

    if args.list:
        for url in urls:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            tag = " [allowlist]" if url in allow else ""
            print(f"  {url}{tag}  ({where})")
        return 0

    results = {"OK": [], "RESTRICTED": [], "ALLOWED": [], "DOWN": []}
    for i, url in enumerate(urls):
        if i:
            time.sleep(args.delay)  #    (  )
        verdict, detail = probe(url, args.timeout, args.retries, args.delay)
        if verdict == "DOWN" and url in allow:
            verdict = "ALLOWED"  #       DOWN
        results[verdict].append((url, detail))
        print(f"  [{verdict:10}] {url} — {detail}")

    print(
        f"\n: OK {len(results['OK'])} · RESTRICTED {len(results['RESTRICTED'])} · "
        f"ALLOWED {len(results['ALLOWED'])} · DOWN {len(results['DOWN'])}"
    )
    if results["RESTRICTED"]:
        print("RESTRICTED( ·  ,   ):")
        for url, detail in results["RESTRICTED"]:
            print(f"  {url} — {detail}")
    if results["ALLOWED"]:
        print("ALLOWED(allowlist    DOWN —     strict ):")
        for url, detail in results["ALLOWED"]:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            print(f"  {url} — {detail}  (: {where})")
    if results["DOWN"]:
        print("DOWN(   —  ):")
        for url, detail in results["DOWN"]:
            where = ", ".join(f"{f}:{ln}" for f, ln in found[url])
            print(f"  {url} — {detail}  (: {where})")

    if args.strict and results["DOWN"]:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
