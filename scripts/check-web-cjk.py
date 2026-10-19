#!/usr/bin/env python3
"""   UI(`web/out`) HTML  ()   .

     "    " 
( B4   `out` HTML  0 ).    , 
         
     .       .

 
---------
`web/out`   `.html`  ** **(   
·React Server Components  )    
  1  .     ,   
 (JSON  )  HTML    . 
     .

   ( )
-----------------------
     .  **MOCK (`NEXT_PUBLIC_MOCK=1`,
   )      **  .
`web/src/lib/mock/data.ts`·`handler.ts`      
 `useEffect`  `mockHandle()`   React  , `out` HTML 
  JS  .        
(   `/system/logs`       ,
 G114   ).

    .   (`web/src`  `.tsx`· )
      :  , `t.has(...)`   
  i18n ,    (   ·`[]`  
), locale  `zh`     ,     
  ( [A])    .    
 "   "  ( G115   
 ).

· 
-------------
- **(`-` ) **.       ,
       .
- **(CJK ) **:    A(``–``),  
  (``–``),     B (`𠀀`–). · 
  ""     (     
            ).

    
-----------------------------
 `out` HTML         .   
 ()     (:   `` )
,    `ALLOWED_HAN`      . 
        .

     .
: `python3 -I scripts/check-web-cjk.py [out ]`
(   `web/out`. CI  `web`  
`python3 -I ../scripts/check-web-cjk.py`  .)
"""
import os
import re
import subprocess
import sys

# :  A +  (U+3400–U+9FFF = –),  (U+F900–U+FAFF = –),
HAN = re.compile(r"[--\U00020000-\U0002ffff]")

ALLOWED_HAN: set = set()


def find_out_dir() -> str:
    """ `out`  .

       ,  git   `web/out`  .
    """
    if len(sys.argv) > 1:
        return os.path.abspath(sys.argv[1])
    root = subprocess.check_output(
        ["git", "rev-parse", "--show-toplevel"], text=True
    ).strip()
    return os.path.join(root, "web", "out")


def main() -> int:
    out_dir = find_out_dir()
    if not os.path.isdir(out_dir):
        print(
            f":    — {out_dir}\n"
            " `npm run build:static`      ."
        )
        return 2

    checked = 0
    offenders = []  # (,  , )
    for dirpath, _dirnames, filenames in os.walk(out_dir):
        for name in filenames:
            if not name.endswith(".html"):
                continue
            checked += 1
            path = os.path.join(dirpath, name)
            with open(path, encoding="utf-8", errors="ignore") as fh:
                text = fh.read()
            found = [ch for ch in HAN.findall(text) if ch not in ALLOWED_HAN]
            if found:
                rel = os.path.relpath(path, out_dir)
                kinds = "".join(sorted(set(found)))
                offenders.append((rel, len(found), kinds))

    print(f" HTML {checked} (: {out_dir})")
    if offenders:
        print(f"()  {len(offenders)}  —   HTML   :")
        for rel, count, kinds in offenders:
            preview = kinds if len(kinds) <= 30 else kinds[:30] + "…"
            print(f"  out/{rel}:  {count} ·  {preview}")
        print(
            "\n ,       "
            "scripts/check-web-cjk.py  ALLOWED_HAN    ."
        )
        return 1
    if checked == 0:
        print(":  HTML   —    .")
        return 2
    print("()  0 —   HTML   /.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
