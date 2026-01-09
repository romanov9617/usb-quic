#!/usr/bin/env python3
"""
Plot usbip experiment metrics from the summary produced by aggregate_usbip.py

Creates PNGs in --out/plots.

Usage:
  python plot_usbip.py --summary ./out/summary.csv --out ./out
"""
from __future__ import annotations

import argparse
from pathlib import Path

import numpy as np
import pandas as pd
import matplotlib.pyplot as plt


def _ensure_dir(p: Path) -> None:
    p.mkdir(parents=True, exist_ok=True)


def _log_safe(v):
    # avoid log(0)
    return np.where(np.asarray(v) <= 0, np.nan, v)


def plot_iops_vs_delay(df: pd.DataFrame, outdir: Path) -> None:
    # One line per job+rw
    gcols = ["jobname", "rw"]
    fig, ax = plt.subplots(figsize=(9, 5))
    for (job, rw), part in df.groupby(gcols):
        part = part.sort_values("delay_ms")
        ax.plot(part["delay_ms"], part["iops"], marker="o", label=f"{job}:{rw}")
    ax.set_xlabel("Injected one-way delay (ms) from profile.env (as recorded)")
    ax.set_ylabel("IOPS")
    ax.set_yscale("log")
    ax.grid(True, which="both")
    ax.legend(fontsize=8)
    fig.tight_layout()
    fig.savefig(outdir / "iops_vs_delay_log.png", dpi=160)
    plt.close(fig)


def plot_bw_vs_delay(df: pd.DataFrame, outdir: Path) -> None:
    fig, ax = plt.subplots(figsize=(9, 5))
    for (job, rw), part in df.groupby(["jobname", "rw"]):
        part = part.sort_values("delay_ms")
        # fio bw is KiB/s; convert to MiB/s for readability
        bw_mib_s = part["bw_kib_s"] / 1024.0
        ax.plot(part["delay_ms"], bw_mib_s, marker="o", label=f"{job}:{rw}")
    ax.set_xlabel("Injected delay (ms)")
    ax.set_ylabel("Bandwidth (MiB/s)")
    ax.set_yscale("log")
    ax.grid(True, which="both")
    ax.legend(fontsize=8)
    fig.tight_layout()
    fig.savefig(outdir / "bw_vs_delay_log.png", dpi=160)
    plt.close(fig)


def plot_tail_latency(df: pd.DataFrame, outdir: Path, job_filter: str | None = None, rw_filter: str | None = None) -> None:
    part = df.copy()
    if job_filter:
        part = part[part["jobname"] == job_filter]
    if rw_filter:
        part = part[part["rw"] == rw_filter]
    if part.empty:
        return

    # For each job+rw: plot p50/p95/p99/p99.9
    for (job, rw), g in part.groupby(["jobname", "rw"]):
        g = g.sort_values("delay_ms")
        fig, ax = plt.subplots(figsize=(9, 5))
        for col, label in [
            ("clat_p50_ms", "p50"),
            ("clat_p95_ms", "p95"),
            ("clat_p99_ms", "p99"),
            ("clat_p99_9_ms", "p99.9"),
        ]:
            if col in g.columns:
                ax.plot(g["delay_ms"], g[col], marker="o", label=label)
        ax.set_xlabel("Injected delay (ms)")
        ax.set_ylabel("Completion latency (ms)")
        ax.set_yscale("log")
        ax.grid(True, which="both")
        ax.legend()
        ax.set_title(f"Tail latency vs delay — {job}:{rw}")
        fig.tight_layout()
        fig.savefig(outdir / f"tail_latency_{job}_{rw}.png", dpi=160)
        plt.close(fig)


def plot_slat_clat_breakdown(df: pd.DataFrame, outdir: Path) -> None:
    # Compare slat_mean and clat_mean as two lines per job+rw
    for (job, rw), g in df.groupby(["jobname", "rw"]):
        g = g.sort_values("delay_ms")
        fig, ax = plt.subplots(figsize=(9, 5))
        ax.plot(g["delay_ms"], g["slat_mean_ms"], marker="o", label="slat_mean_ms")
        ax.plot(g["delay_ms"], g["clat_mean_ms"], marker="o", label="clat_mean_ms")
        ax.set_xlabel("Injected delay (ms)")
        ax.set_ylabel("Mean latency component (ms)")
        ax.set_yscale("log")
        ax.grid(True, which="both")
        ax.legend()
        ax.set_title(f"slat vs clat mean — {job}:{rw}")
        fig.tight_layout()
        fig.savefig(outdir / f"slat_clat_mean_{job}_{rw}.png", dpi=160)
        plt.close(fig)


def plot_progress(df: pd.DataFrame, outdir: Path) -> None:
    # total_ios per run vs delay (shows near-deadlock conditions)
    fig, ax = plt.subplots(figsize=(9, 5))
    for (job, rw), g in df.groupby(["jobname", "rw"]):
        g = g.sort_values("delay_ms")
        ax.plot(g["delay_ms"], g["total_ios"], marker="o", label=f"{job}:{rw}")
    ax.set_xlabel("Injected delay (ms)")
    ax.set_ylabel("Total IOs completed (15s run)")
    ax.set_yscale("log")
    ax.grid(True, which="both")
    ax.legend(fontsize=8)
    fig.tight_layout()
    fig.savefig(outdir / "total_ios_vs_delay_log.png", dpi=160)
    plt.close(fig)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--summary", type=str, required=True, help="Path to summary.csv")
    ap.add_argument("--out", type=str, required=True, help="Output directory (will create plots/)")
    args = ap.parse_args()

    summary = Path(args.summary).resolve()
    out = Path(args.out).resolve()
    plotdir = out / "plots"
    _ensure_dir(plotdir)

    df = pd.read_csv(summary)

    # Ensure numeric
    for c in ["delay_ms", "iops", "bw_kib_s", "slat_mean_ms", "clat_mean_ms", "total_ios",
              "clat_p50_ms", "clat_p95_ms", "clat_p99_ms", "clat_p99_9_ms"]:
        if c in df.columns:
            df[c] = pd.to_numeric(df[c], errors="coerce")

    # Filter out rows without delay (if any)
    df = df.dropna(subset=["delay_ms"])
    df = df.sort_values(["delay_ms", "jobname", "rw"])

    plot_iops_vs_delay(df, plotdir)
    plot_bw_vs_delay(df, plotdir)
    plot_tail_latency(df, plotdir)
    plot_slat_clat_breakdown(df, plotdir)
    plot_progress(df, plotdir)

    print(f"Wrote plots to: {plotdir}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
