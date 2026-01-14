import argparse
import json
import re
import tarfile
import tempfile
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

# ----------------------------
# Helpers: time / parsing
# ----------------------------


def parse_iso(ts: str) -> datetime:
    # events.csv uses ISO 8601 with timezone, e.g. 2026-01-14T21:30:32+00:00
    return datetime.fromisoformat(ts.replace("Z", "+00:00"))


def safe_read_text(p: Path) -> str:
    try:
        return p.read_text(encoding="utf-8", errors="replace")
    except FileNotFoundError:
        return ""
    except Exception:
        return p.read_text(errors="replace")


def extract_if_archive(
    input_path: Path,
) -> Tuple[Path, Optional[tempfile.TemporaryDirectory]]:
    """
    Returns: root_dir_that_contains_results, tempdir_handle_if_any
    """
    if input_path.is_file() and input_path.name.endswith((".tar.gz", ".tgz")):
        td = tempfile.TemporaryDirectory(prefix="usbip_results_")
        out_dir = Path(td.name)
        with tarfile.open(input_path, "r:gz") as tf:
            tf.extractall(out_dir)
        # typically archive has "results/..."
        if (out_dir / "results").exists():
            return (out_dir / "results"), td
        # fallback: maybe extracted directly
        return out_dir, td

    # If user passes a directory:
    if input_path.is_dir():
        # accept either root containing results/ or results itself
        if (input_path / "results").exists():
            return (input_path / "results"), None
        return input_path, None

    raise FileNotFoundError(f"Input path not found: {input_path}")


# ----------------------------
# FIO extraction
# ----------------------------


def _pct_lookup(percentile_map: Dict[str, float], key: str) -> Optional[float]:
    # fio writes percentile keys as strings like "50.000000", "95.000000"
    # We match by rounding to 3 decimals where possible.
    if not percentile_map:
        return None
    # Exact
    if key in percentile_map:
        return percentile_map[key]
    # Try numeric match
    try:
        target = float(key)
        # find closest key
        best_k = None
        best_d = None
        for k in percentile_map.keys():
            try:
                v = float(k)
            except Exception:
                continue
            d = abs(v - target)
            if best_d is None or d < best_d:
                best_d = d
                best_k = k
        if best_k is not None and best_d is not None and best_d <= 1e-6:
            return percentile_map[best_k]
    except Exception:
        pass
    return None


def parse_fio_json(p: Path) -> Dict[str, Optional[float]]:
    """
    Returns normalized metrics:
      bw_kib_s, iops, io_kbytes, runtime_s,
      clat_mean_s, clat_max_s, clat_p50_s, clat_p95_s, clat_p99_s, clat_p999_s
    """
    if not p.exists():
        return {}

    fio = json.loads(p.read_text(encoding="utf-8", errors="replace"))
    jobs = fio.get("jobs", [])
    if not jobs:
        return {}

    job = jobs[0]
    w = job.get("write", {}) or {}

    io_kbytes = w.get("io_kbytes")
    bw_kib_s = w.get("bw")  # fio json "bw" is KiB/s
    iops = w.get("iops")

    runtime_ms = job.get("job_runtime")
    runtime_s = (runtime_ms / 1000.0) if isinstance(runtime_ms, (int, float)) else None

    clat_ns = w.get("clat_ns", {}) or {}
    # fio uses ns for clat
    clat_mean_s = (
        (clat_ns.get("mean") / 1e9)
        if isinstance(clat_ns.get("mean"), (int, float))
        else None
    )
    clat_max_s = (
        (clat_ns.get("max") / 1e9)
        if isinstance(clat_ns.get("max"), (int, float))
        else None
    )

    pct = clat_ns.get("percentile", {}) or {}

    def pct_s(pct_key: str) -> Optional[float]:
        v = _pct_lookup(pct, pct_key)
        if isinstance(v, (int, float)):
            return v / 1e9
        return None

    return {
        "io_kbytes": io_kbytes,
        "bw_kib_s": bw_kib_s,
        "iops": iops,
        "runtime_s": runtime_s,
        "clat_mean_s": clat_mean_s,
        "clat_max_s": clat_max_s,
        "clat_p50_s": pct_s("50.000000"),
        "clat_p95_s": pct_s("95.000000"),
        "clat_p99_s": pct_s("99.000000"),
        "clat_p999_s": pct_s("99.900000"),  # fio often has 99.900000
    }


# ----------------------------
# Events extraction
# ----------------------------


@dataclass
class InjectionInfo:
    inject_at_s: Optional[float]
    inj_len_s: Optional[float]
    inj_mode: Optional[str]
    injection_start_ts: Optional[datetime]
    injection_end_ts: Optional[datetime]


def parse_events_csv(
    p: Path,
) -> Tuple[Dict[str, Optional[float]], InjectionInfo, pd.DataFrame]:
    """
    Returns:
      - basic run timestamps: run_start_ts, fio_start_ts, fio_end_ts, etc.
      - InjectionInfo
      - full events dataframe (normalized)
    """
    if not p.exists():
        empty_info = InjectionInfo(None, None, None, None, None)
        return ({}, empty_info, pd.DataFrame(columns=["ts", "event", "details"]))

    df = pd.read_csv(p)
    if "ts" not in df.columns or "event" not in df.columns:
        empty_info = InjectionInfo(None, None, None, None, None)
        return ({}, empty_info, df)

    df["ts_dt"] = df["ts"].apply(parse_iso)

    def first_ts(ev_name: str) -> Optional[datetime]:
        sub = df[df["event"] == ev_name]
        if sub.empty:
            return None
        return sub.iloc[0]["ts_dt"]

    def last_ts(ev_name: str) -> Optional[datetime]:
        sub = df[df["event"] == ev_name]
        if sub.empty:
            return None
        return sub.iloc[-1]["ts_dt"]

    run_start = first_ts("run_start")
    fio_start = first_ts("fio_start")
    fio_end = first_ts("fio_end") or last_ts("fio_end")

    # injection parsing
    inj_start_row = df[df["event"] == "injection_start"]
    inj_mode = None
    inj_len_s = None
    inj_start_ts = None
    if not inj_start_row.empty:
        inj_start_ts = inj_start_row.iloc[0]["ts_dt"]
        details = str(inj_start_row.iloc[0].get("details", ""))
        # e.g. "mode=route_blackhole;len=30s"
        m = re.search(r"mode=([a-zA-Z0-9_\-]+)", details)
        if m:
            inj_mode = m.group(1)
        m = re.search(r"len=(\d+)s", details)
        if m:
            inj_len_s = float(m.group(1))

    inj_end_row = df[df["event"] == "injection_reverted"]
    inj_end_ts = None
    if not inj_end_row.empty:
        inj_end_ts = inj_end_row.iloc[0]["ts_dt"]

    inject_wait = df[df["event"] == "inject_wait_done"]
    inject_at_s = None
    if not inject_wait.empty:
        details = str(inject_wait.iloc[0].get("details", ""))
        m = re.search(r"at=(\d+)s", details)
        if m:
            inject_at_s = float(m.group(1))

    info = InjectionInfo(
        inject_at_s=inject_at_s,
        inj_len_s=inj_len_s,
        inj_mode=inj_mode,
        injection_start_ts=inj_start_ts,
        injection_end_ts=inj_end_ts,
    )

    # derived durations
    basics = {
        "run_start_ts": run_start.isoformat() if run_start else None,
        "fio_start_ts": fio_start.isoformat() if fio_start else None,
        "fio_end_ts": fio_end.isoformat() if fio_end else None,
        "fio_wall_s": (fio_end - fio_start).total_seconds()
        if (fio_start and fio_end)
        else None,
        "inject_at_s": inject_at_s,
        "inj_len_s": inj_len_s,
    }

    return basics, info, df


# ----------------------------
# Log signatures / anomaly counters
# ----------------------------

SIGNATURES = {
    # key: regex
    "ecnnreset_-104": r"\b-104\b",
    "vhci_reset": r"reset (high-speed|full-speed|super-speed) USB device",
    "setaddress": r"SetAddress Request",
    "fat_not_unmounted": r"Volume was not properly unmounted",
    "usb_disconnect": r"USB disconnect",
    "usb_connect_newdev": r"new (high-speed|full-speed|super-speed) USB device",
    "usbip_error": r"\b(usbip|vhci_hcd).*(error|ERR|failed)\b",
}


def count_signatures(text: str) -> Dict[str, int]:
    out = {}
    for k, rx in SIGNATURES.items():
        out[k] = len(re.findall(rx, text, flags=re.IGNORECASE))
    return out


# ----------------------------
# Main scan
# ----------------------------


def find_cases(results_root: Path) -> List[Path]:
    """
    Expected layout:
      results/<run_id>/<case_id>/{fio.json, events.csv, ...}
    """
    cases = []
    for run_dir in sorted(results_root.glob("*")):
        if not run_dir.is_dir():
            continue
        for case_dir in sorted(run_dir.glob("*")):
            if case_dir.is_dir():
                cases.append(case_dir)
    return cases


def make_plots(df: pd.DataFrame, out_dir: Path) -> None:
    # Sort for consistent visuals
    df = df.copy()
    if "inj_len_s" in df.columns:
        df["inj_len_s_num"] = pd.to_numeric(df["inj_len_s"], errors="coerce")
    else:
        df["inj_len_s_num"] = np.nan

    df = df.sort_values(["inj_len_s_num", "case_id"], na_position="last")

    # 1) Throughput
    plt.figure()
    x = np.arange(len(df))
    plt.bar(x, df["bw_kib_s"].fillna(0))
    plt.xticks(x, df["case_id"], rotation=30, ha="right")
    plt.ylabel("bw (KiB/s)")
    plt.title("FIO write bandwidth by case")
    plt.tight_layout()
    plt.savefig(out_dir / "bw_kib_s_by_case.png", dpi=160)
    plt.close()

    # 2) Max clat
    plt.figure()
    plt.bar(x, df["clat_max_s"].fillna(0))
    plt.xticks(x, df["case_id"], rotation=30, ha="right")
    plt.ylabel("clat max (s)")
    plt.title("FIO max completion latency by case")
    plt.tight_layout()
    plt.savefig(out_dir / "clat_max_s_by_case.png", dpi=160)
    plt.close()

    # 3) Percentiles line plot
    plt.figure()
    for col, label in [
        ("clat_p50_s", "p50"),
        ("clat_p95_s", "p95"),
        ("clat_p99_s", "p99"),
        ("clat_p999_s", "p99.9"),
    ]:
        if col in df.columns:
            plt.plot(x, df[col], marker="o", label=label)
    plt.xticks(x, df["case_id"], rotation=30, ha="right")
    plt.ylabel("clat percentile (s)")
    plt.title("FIO clat percentiles by case")
    plt.legend()
    plt.tight_layout()
    plt.savefig(out_dir / "clat_percentiles_by_case.png", dpi=160)
    plt.close()

    # 4) Error signatures
    sig_cols = [c for c in df.columns if c in SIGNATURES.keys()]
    if sig_cols:
        plt.figure()
        # stacked bars
        bottom = np.zeros(len(df))
        for c in sig_cols:
            vals = df[c].fillna(0).to_numpy(dtype=float)
            plt.bar(x, vals, bottom=bottom, label=c)
            bottom += vals
        plt.xticks(x, df["case_id"], rotation=30, ha="right")
        plt.ylabel("count")
        plt.title("Kernel/USBIP log signature counts")
        plt.legend(fontsize=8)
        plt.tight_layout()
        plt.savefig(out_dir / "log_signatures_stacked.png", dpi=160)
        plt.close()

    # 5) Recovery overhead: (fio wall) - runtime setting vs injection
    if "fio_wall_s" in df.columns and "inj_len_s" in df.columns:
        plt.figure()
        wall = pd.to_numeric(df["fio_wall_s"], errors="coerce")
        inj = pd.to_numeric(df["inj_len_s"], errors="coerce")
        plt.scatter(inj, wall)
        for _, r in df.iterrows():
            try:
                plt.annotate(
                    r["case_id"],
                    (float(r["inj_len_s"]), float(r["fio_wall_s"])),
                    fontsize=8,
                )
            except Exception:
                pass
        plt.xlabel("injection length (s)")
        plt.ylabel("fio wall time (s)")
        plt.title("fio wall time vs injection length")
        plt.tight_layout()
        plt.savefig(out_dir / "fio_wall_vs_injection.png", dpi=160)
        plt.close()


def main() -> int:
    ap = argparse.ArgumentParser(description="USB/IP migration experiment visualizer")
    ap.add_argument("input", help="Path to results/ directory OR .tar.gz archive")
    ap.add_argument(
        "--out", default="viz_out", help="Output directory (default: viz_out)"
    )
    args = ap.parse_args()

    input_path = Path(args.input).expanduser().resolve()
    results_root, td = extract_if_archive(input_path)

    out_dir = Path(args.out).expanduser().resolve()
    out_dir.mkdir(parents=True, exist_ok=True)

    rows = []
    events_rows = []

    for case_dir in find_cases(results_root):
        run_id = case_dir.parent.name
        case_id = case_dir.name

        fio_metrics = parse_fio_json(case_dir / "fio.json")
        basics, inj_info, ev_df = parse_events_csv(case_dir / "events.csv")

        dmesg = safe_read_text(case_dir / "dmesg_tail.log")
        usbip = safe_read_text(case_dir / "usbip_port.log")
        sig_counts = count_signatures(dmesg + "\n" + usbip)

        row = {
            "run_id": run_id,
            "case_id": case_id,
            "inj_mode": inj_info.inj_mode,
            "inject_at_s": inj_info.inject_at_s,
            "inj_len_s": inj_info.inj_len_s,
            "injection_start_ts": inj_info.injection_start_ts.isoformat()
            if inj_info.injection_start_ts
            else None,
            "injection_end_ts": inj_info.injection_end_ts.isoformat()
            if inj_info.injection_end_ts
            else None,
        }
        row.update(basics)
        row.update(fio_metrics)
        row.update(sig_counts)

        # Extra: a simple "stall-ish" metric: max clat minus inj_len (if both exist)
        try:
            if row.get("clat_max_s") is not None and row.get("inj_len_s") is not None:
                row["clat_max_minus_inj_s"] = float(row["clat_max_s"]) - float(
                    row["inj_len_s"]
                )
        except Exception:
            row["clat_max_minus_inj_s"] = None

        rows.append(row)

        # events export (normalized)
        if not ev_df.empty:
            tmp = ev_df.copy()
            tmp["run_id"] = run_id
            tmp["case_id"] = case_id
            # keep original ISO + parsed dt
            tmp["ts_iso"] = tmp["ts"]
            tmp["ts"] = tmp["ts_dt"].astype(str)
            events_rows.append(tmp[["run_id", "case_id", "ts_iso", "event", "details"]])

    df = pd.DataFrame(rows)
    df.to_csv(out_dir / "summary_runs.csv", index=False)

    if events_rows:
        ev_all = pd.concat(events_rows, ignore_index=True)
        ev_all.to_csv(out_dir / "summary_events.csv", index=False)

    # plots
    if not df.empty:
        make_plots(df, out_dir)

    print(f"OK: wrote {out_dir / 'summary_runs.csv'}")
    if events_rows:
        print(f"OK: wrote {out_dir / 'summary_events.csv'}")
    print(f"OK: plots in {out_dir}")

    # cleanup tempdir if archive
    if td is not None:
        td.cleanup()

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
