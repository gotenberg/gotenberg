#!/usr/bin/env python3
"""
Gotenberg LibreOffice conversion load test.

Continuously sends all documents in a directory through Gotenberg using
a pool of concurrent workers.  Transparently retries on 429 (busy pod)
with exponential backoff so you can observe:

  • Pod readiness state changes  (kubectl get pods -n gotenberg-local -w)
  • 429 retries in the console output
  • Even distribution across all 5 pods

Usage
-----
  # Continuous load — loop forever through all docs in ./docs/
  python3 convert_test.py --dir ./docs --concurrency 5

  # Run through the directory once, then stop
  python3 convert_test.py --dir ./docs --concurrency 5 --once

  # Single file, continuous
  python3 convert_test.py --dir ./docs --file test.docx --concurrency 5

  # Save converted PDFs to ./output/
  python3 convert_test.py --dir ./docs --concurrency 5 --save-pdfs

  # Override URL (default is Nginx Ingress on localhost:3080)
  python3 convert_test.py --dir ./docs --url http://localhost:3000

Requirements
------------
  pip install requests
"""

import argparse
import itertools
import random
import sys
import threading
import time
from dataclasses import dataclass, field
from pathlib import Path
from queue import Queue, Empty

import requests
from requests.adapters import HTTPAdapter

# ── CLI ───────────────────────────────────────────────────────────────────────

parser = argparse.ArgumentParser(description="Gotenberg conversion load test")
parser.add_argument("--url",         default="http://localhost:3080",
                    help="Gotenberg base URL via Nginx Ingress (default: http://localhost:3080)")
parser.add_argument("--dir",         default=".",
                    help="Directory containing documents to convert (default: current dir)")
parser.add_argument("--file",        default=None,
                    help="Only send this specific file (relative to --dir or absolute)")
parser.add_argument("--concurrency", type=int, default=5,
                    help="Number of concurrent workers (default: 5)")
parser.add_argument("--once",        action="store_true",
                    help="Stop after processing each document once instead of looping")
parser.add_argument("--timeout",     type=float, default=120.0,
                    help="Read timeout per request in seconds (default: 120)")
parser.add_argument("--retry-max",   type=int, default=20,
                    help="Max 429 retries before giving up on a single request (default: 20)")
parser.add_argument("--retry-delay", type=float, default=0.5,
                    help="Initial retry delay in seconds — doubles each attempt (default: 0.5)")
parser.add_argument("--save-pdfs",  action="store_true",
                    help="Write converted PDFs to ./output/")
args = parser.parse_args()

# ── File list ─────────────────────────────────────────────────────────────────

SUPPORTED = {
    ".docx", ".doc", ".odt", ".ods", ".odp", ".odg",
    ".xlsx", ".xls", ".pptx", ".ppt", ".csv", ".rtf",
    ".txt", ".html", ".htm",
}

doc_dir = Path(args.dir)
if not doc_dir.is_dir():
    sys.exit(f"Directory not found: {doc_dir}")

if args.file:
    p = Path(args.file)
    if not p.is_absolute():
        p = doc_dir / p
    if not p.exists():
        sys.exit(f"File not found: {p}")
    all_docs = [p]
else:
    all_docs = sorted(
        f for f in doc_dir.iterdir()
        if f.is_file() and f.suffix.lower() in SUPPORTED
    )
    if not all_docs:
        sys.exit(f"No supported documents found in {doc_dir}")

ENDPOINT = args.url.rstrip("/") + "/forms/libreoffice/convert"
OUT_DIR  = Path("output")
if args.save_pdfs:
    OUT_DIR.mkdir(exist_ok=True)

# ── Stats ─────────────────────────────────────────────────────────────────────

lock = threading.Lock()
stats = {
    "ok":       0,
    "failed":   0,
    "retries":  0,
    "total":    0,
    "failures": {},   # status → count
}
start_time = time.monotonic()

def bump(key, n=1):
    with lock:
        stats[key] += n

# ── Per-thread session (connection pool, one per worker) ──────────────────────

_thread_local = threading.local()

def get_session() -> requests.Session:
    """Return a per-thread requests Session.

    Uses pool_maxsize=1 and no keepalive so each request opens and closes
    its own TCP connection.  This prevents macOS socket buffer exhaustion
    (OSError 55 / ENOBUFS) when uploading multiple large files concurrently.
    """
    if not getattr(_thread_local, "session", None):
        s = requests.Session()
        adapter = HTTPAdapter(
            pool_connections=1,
            pool_maxsize=1,
            max_retries=0,
        )
        s.mount("http://",  adapter)
        s.mount("https://", adapter)
        # Disable keepalive — connection closes after each request
        s.headers.update({"Connection": "close"})
        _thread_local.session = s
    return _thread_local.session


# ── Worker ────────────────────────────────────────────────────────────────────

def convert(doc_path: Path, req_id: int) -> dict:
    """
    POST doc_path to Gotenberg.
    - Retries on 429 (busy pod) with exponential backoff + jitter.
    - Retries on connection errors (port-forward blip) up to 5 times.
    Returns a result dict.
    """
    delay        = args.retry_delay
    attempt      = 0
    conn_retries = 0
    session      = get_session()

    while True:
        t0 = time.monotonic()
        try:
            with doc_path.open("rb") as fh:
                resp = session.post(
                    ENDPOINT,
                    files={"files": (doc_path.name, fh, "application/octet-stream")},
                    timeout=(10, args.timeout),
                )
            elapsed = time.monotonic() - t0

            if resp.status_code == 200:
                pdf_bytes = resp.content
                if args.save_pdfs:
                    out = OUT_DIR / f"{doc_path.stem}_{req_id:04d}.pdf"
                    out.write_bytes(pdf_bytes)
                return {"status": "ok", "dur": elapsed, "bytes": len(pdf_bytes),
                        "file": doc_path.name, "attempts": attempt + 1}

            if resp.status_code == 429:
                attempt += 1
                bump("retries")
                jitter = random.uniform(0, delay * 0.5)
                wait   = delay + jitter
                print(f"  [req {req_id:04d}] 429 busy — retry {attempt} in {wait:.2f}s  ({doc_path.name})",
                      flush=True)
                if attempt > args.retry_max:
                    return {"status": "give_up", "dur": elapsed, "file": doc_path.name,
                            "attempts": attempt}
                time.sleep(wait)
                delay = min(delay * 2, 10.0)
                continue

            # Strip HTML tags for readable error detail
            import re as _re
            body = resp.text
            body = _re.sub(r'<[^>]+>', ' ', body)   # remove tags
            body = ' '.join(body.split())            # collapse whitespace
            return {"status": f"http_{resp.status_code}", "dur": elapsed,
                    "file": doc_path.name, "attempts": attempt + 1,
                    "detail": body[:400]}

        except requests.exceptions.ConnectionError as exc:
            elapsed = time.monotonic() - t0
            msg = str(exc)
            # Distinguish a genuine reset from a port-forward blip
            if "ConnectionReset" in msg or "104" in msg:
                return {"status": "reset", "dur": elapsed, "file": doc_path.name,
                        "attempts": attempt + 1, "detail": msg[:80]}
            # Port-forward / transient connection failure — retry a few times
            conn_retries += 1
            if conn_retries <= 5:
                wait = conn_retries * 1.0
                print(f"  [req {req_id:04d}] conn_error (attempt {conn_retries}/5) retry in {wait:.1f}s — {msg[:60]}",
                      flush=True)
                # Drop the stale session so a fresh connection is made
                _thread_local.session = None
                session = get_session()
                time.sleep(wait)
                continue
            return {"status": "conn_error", "dur": elapsed, "file": doc_path.name,
                    "attempts": attempt + 1, "detail": msg[:80]}

        except requests.exceptions.ReadTimeout:
            elapsed = time.monotonic() - t0
            return {"status": "timeout", "dur": elapsed, "file": doc_path.name,
                    "attempts": attempt + 1}

        except Exception as exc:
            elapsed = time.monotonic() - t0
            return {"status": "error", "dur": elapsed, "file": doc_path.name,
                    "attempts": attempt + 1, "detail": str(exc)[:80]}


def worker(work_queue: Queue, req_counter: itertools.count):
    """Pull docs from the queue and convert them."""
    while True:
        try:
            doc_path = work_queue.get(timeout=2)
        except Empty:
            return

        req_id = next(req_counter)
        bump("total")
        result = convert(doc_path, req_id)
        if result["status"] == "ok":
            bump("ok")
        else:
            bump("failed")
            with lock:
                stats["failures"][result["status"]] = stats["failures"].get(result["status"], 0) + 1

        elapsed = time.monotonic() - start_time
        with lock:
            ok = stats["ok"]; total = stats["total"]; retries = stats["retries"]

            icon  = "✓" if result["status"] == "ok" else "✗"
            if result["status"] == "ok":
                extra = f"{result['bytes']:,} bytes  attempts={result['attempts']}"
            elif result["status"] == "give_up":
                extra = f"status=give_up  all {result['attempts']} retries exhausted (all pods busy)"
            else:
                extra = f"status={result['status']}"

            # Print result — hold lock so multi-line output isn't interleaved
            print(
                f"  {icon} [{elapsed:7.1f}s] req={req_id:04d}  {result['file']:<40}"
                f"  {result['dur']:6.1f}s  {extra}",
                flush=True,
            )
            if result["status"] != "ok":
                detail   = result.get("detail", "")
                attempts = result.get("attempts", 1)
                print(
                    f"       └─ attempts={attempts}  "
                    f"detail: {detail if detail else '(none)'}",
                    flush=True,
                )

        # Running totals every 10 requests
        if total % 10 == 0:
            rate = ok / elapsed if elapsed > 0 else 0
            print(
                f"\n  ── {total} sent  {ok} ok  {total-ok} failed  "
                f"{retries} retries  {rate:.2f} req/s ──\n",
                flush=True,
            )

        work_queue.task_done()


# ── Main ──────────────────────────────────────────────────────────────────────

def fill_queue(q: Queue):
    """Fill q — once if --once, otherwise loop forever."""
    if args.once:
        for doc in all_docs:
            q.put(doc)
    else:
        try:
            for doc in itertools.cycle(all_docs):
                # Throttle: don't build up more than 2× concurrency of backlog
                while q.qsize() >= args.concurrency * 2:
                    time.sleep(0.1)
                q.put(doc)
        except KeyboardInterrupt:
            pass


def main():
    print()
    print("=" * 70)
    print(f"  Gotenberg continuous load test")
    print(f"  URL         : {ENDPOINT}")
    print(f"  Docs dir    : {doc_dir}  ({len(all_docs)} files)")
    print(f"  Files       : {', '.join(d.name for d in all_docs[:5])}"
          + (" …" if len(all_docs) > 5 else ""))
    print(f"  Concurrency : {args.concurrency} workers")
    print(f"  Mode        : {'once' if args.once else 'continuous loop'}")
    print(f"  Retry 429   : up to {args.retry_max}x  initial delay {args.retry_delay}s")
    print(f"  Timeout     : {args.timeout}s read")
    print("=" * 70)
    print("  Ctrl-C to stop\n")
    print("  Watch pods:   kubectl get pods -n gotenberg-local -w")
    print()

    work_queue   = Queue()
    req_counter  = itertools.count(1)

    # Start the queue filler in a background thread
    filler = threading.Thread(target=fill_queue, args=(work_queue,), daemon=True)
    filler.start()

    # Give the queue a head start
    time.sleep(0.3)

    # Start workers — staggered so they don't all open large upload connections
    # simultaneously and exhaust macOS TCP socket buffers (ENOBUFS / OSError 55)
    threads = []
    for i in range(args.concurrency):
        t = threading.Thread(target=worker, args=(work_queue, req_counter), daemon=True)
        t.start()
        threads.append(t)
        if i < args.concurrency - 1:
            time.sleep(0.5)   # 0.5s stagger between worker starts

    try:
        if args.once:
            work_queue.join()
        else:
            # Run until Ctrl-C
            while True:
                time.sleep(1)
    except KeyboardInterrupt:
        print("\n\n  Stopping…", flush=True)

    elapsed = time.monotonic() - start_time
    ok      = stats["ok"]
    total   = stats["total"]
    rate    = ok / elapsed if elapsed > 0 else 0

    print()
    print("=" * 70)
    print(f"  FINAL SUMMARY  ({elapsed:.1f}s)")
    print(f"    Requests sent : {total}")
    print(f"    Success       : {ok}  ({100*ok//max(total,1)}%)")
    print(f"    Failed        : {total - ok}")
    print(f"    429 retries   : {stats['retries']}")
    print(f"    Throughput    : {rate:.2f} conversions/s")
    # Failure breakdown
    if stats["failures"]:
        print(f"\n  Failure breakdown:")
        for reason, count in sorted(stats["failures"].items(), key=lambda x: -x[1]):
            print(f"    {reason:<30} {count}x")
    print("=" * 70)


if __name__ == "__main__":
    main()
