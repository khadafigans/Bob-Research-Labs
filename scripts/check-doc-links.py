#!/usr/bin/env python3
"""     ··   .

  `.md`     .

1.        (`[text](path)`) 
   (`![alt](path)`·`<img src="path">`)    .
2.   (`[text](#heading)`·`[text](other.md#heading)`)  `.md`
       .  slug  GitHub  
   ( → `\\p{Word}`··   →  ,
    slug     `-1`·`-2` )  
    .

      1  , CI  
(`.github/workflows/docs.yml`)       .

   :
-  URL(`http://`·`https://`·`mailto:`·`tel:`·`data:`):  
  flaky      .    
  .
- `.md`      `#`  (:    ): 
         .
-   `#` :     .
- fenced code block(```  ~~~   )   ·  :
   ·    .

     . : `python3 -I scripts/check-doc-links.py`
"""
import os
import re
import subprocess
import sys
import unicodedata
from collections import defaultdict

MD_LINK = re.compile(r"(?<!\!)\[[^\]]*\]\(([^)\s]+)")
MD_IMAGE = re.compile(r"!\[[^\]]*\]\(([^)\s]+)")
# <img ... src="path" ...> HTML .
HTML_IMAGE = re.compile(r"<img[^>]*\bsrc=[\"']([^\"']+)[\"']", re.IGNORECASE)

EXTERNAL_PREFIXES = ("http://", "https://", "mailto:", "tel:", "data:")

ATX_HEADING = re.compile(r"^(#{1,6})\s+(.*?)\s*#*\s*$")
MD_INLINE_LINK = re.compile(r"\[([^\]]*)\]\([^)]*\)")


def is_external(target: str) -> bool:
    return target.startswith(EXTERNAL_PREFIXES)


def _is_word_char(ch: str) -> bool:
    """GitHub   slug   `\\p{Word}`  .

    Ruby  `\\p{Word}` = Letter(L*) + Mark(M*) + Decimal_Number(Nd)
    + Connector_Punctuation(Pc) + Join_Control(U+200C·U+200D). ··
    · variation selector(Mn)   .
    """
    if ch == "_":
        return True
    if ch in ("‌", "‍"):
        return True
    category = unicodedata.category(ch)
    if category[0] in ("L", "M"):
        return True
    return category in ("Nd", "Pc")


def slugify(text: str) -> str:
    """  GitHub   slug  (   )."""
    text = text.strip().lower()
    out = []
    for ch in text:
        if ch == " ":
            out.append("-")
        elif ch == "-" or _is_word_char(ch):
            out.append(ch)
    return "".join(out)


def heading_slugs(root: str, md: str) -> set:
    """  ATX  GitHub  slug  .

     slug   GitHub    `-1`·`-2`  .
    """
    slugs = set()
    counts = defaultdict(int)
    in_fence = False
    with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
        for line in fh:
            stripped = line.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            match = ATX_HEADING.match(line.rstrip("\n"))
            if not match:
                continue
            raw = match.group(2)
            raw = MD_INLINE_LINK.sub(r"\1", raw)
            raw = raw.replace("`", "").replace("*", "").replace("_", "")
            base = slugify(raw)
            seen = counts[base]
            counts[base] += 1
            slugs.add(base if seen == 0 else f"{base}-{seen}")
    return slugs


def main() -> int:
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    tracked = subprocess.check_output(
        ["git", "ls-files"], cwd=root, text=True
    ).splitlines()
    md_files = [f for f in tracked if f.endswith(".md")]

    slug_cache: dict = {}

    def slugs_of(md_path: str) -> set:
        if md_path not in slug_cache:
            slug_cache[md_path] = heading_slugs(root, md_path)
        return slug_cache[md_path]

    checked_files = 0
    checked_anchors = 0
    broken_files = []
    broken_anchors = []
    for md in md_files:
        with open(os.path.join(root, md), encoding="utf-8", errors="ignore") as fh:
            lines = fh.readlines()
        base_dir = os.path.dirname(md)
        in_fence = False
        for lineno, line in enumerate(lines, 1):
            stripped = line.lstrip()
            if stripped.startswith("```") or stripped.startswith("~~~"):
                in_fence = not in_fence
                continue
            if in_fence:
                continue
            for pattern in (MD_LINK, MD_IMAGE, HTML_IMAGE):
                for match in pattern.finditer(line):
                    target = match.group(1).strip()
                    if is_external(target):
                        continue

                    if target.startswith("#"):
                        file_part, anchor = "", target[1:]
                    elif "#" in target:
                        file_part, anchor = target.split("#", 1)
                    else:
                        file_part, anchor = target, ""
                    file_part = file_part.split("?", 1)[0]
                    anchor = anchor.strip()

                    target_md = None
                    if file_part:
                        checked_files += 1
                        resolved = os.path.normpath(
                            os.path.join(root, base_dir, file_part)
                        )
                        if not os.path.exists(resolved):
                            broken_files.append((md, lineno, target))
                            continue
                        if file_part.endswith(".md"):
                            target_md = os.path.normpath(
                                os.path.join(base_dir, file_part)
                            )
                    else:
                        target_md = md

                    if anchor and target_md is not None and pattern is MD_LINK:
                        checked_anchors += 1
                        if anchor not in slugs_of(target_md):
                            broken_anchors.append((md, lineno, target, target_md))

    print(
        f"  {len(md_files)} ·   {checked_files} · "
        f"  {checked_anchors} "
    )
    if broken_files:
        print(f"   {len(broken_files)} —    :")
        for md, lineno, target in broken_files:
            print(f"  {md}:{lineno} -> {target}")
    if broken_anchors:
        print(f"   {len(broken_anchors)} —     :")
        for md, lineno, target, target_md in broken_anchors:
            print(f"  {md}:{lineno} -> {target}  (: {target_md})")
    if broken_files or broken_anchors:
        return 1
    print("  0 —   ··   .")
    return 0


if __name__ == "__main__":
    sys.exit(main())
