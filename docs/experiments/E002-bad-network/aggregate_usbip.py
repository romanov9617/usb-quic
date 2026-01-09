#!/usr/bin/env python3
"""
Aggregate usbip fio experiment artifacts into tidy tables.

Expected repo layout (example):
  results/
    20260109T191003Z_00_baseline/
      profile.env
      sys_net.txt
      tc_qdisc_after.txt
      tc_qdisc_cleared.txt
      netem_apply.txt
      netem_clear.txt
      preflight.txt
      fio/
        seq_write.json
        seq_read.json
        rand_read_4k.json
        rand_write_4k.json

This script walks --root, finds profile.env files, and aggregates:
  - profile_table.csv  (one row per profile run)
  - fio_table.csv      (one row per profile x job x rwdir)
  - net_table.csv      (one row per profile with selected TCP stats)
  - summary.csv        (joined view: fio_table + profile + net)

Usage:
  python aggregate_usbip.py --root ./results --out ./out
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any, Dict, List, Optional

import pandas as pd


def parse_env(path: Path) -> Dict[str, str]:
    env: Dict[str, str] = {}
    for line in path.read_text(encoding="utf-8", errors="ignore").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip()
    return env


def _to_ms(s: str) -> Optional[float]:
    s = s.strip()
    if not s:
        return None
    # accepts "20ms", "0.5s", "200us"
    m = re.fullmatch(r"(\d+(?:\.\d+)?)(ms|s|us)", s)
    if not m:
        return None
    val = float(m.group(1))
    unit = m.group(2)
    if unit == "us":
        return val / 1000.0
    if unit == "ms":
        return val
    if unit == "s":
        return val * 1000.0
    return None


def parse_ss_rtt(sys_net_txt: str) -> Dict[str, Any]:
    """
    Extract a small set of stats from the 'ss' line in sys_net.txt.
    We look for patterns like:
      rtt:1.547/2.559 rto:202 cwnd:6 bytes_retrans:5897 ...
    Returns numeric fields where possible.
    """
    out: Dict[str, Any] = {}
    # Find the line that contains "ESTAB" and "rtt:"
    m = re.search(r"\bESTAB\b.*", sys_net_txt)
    if not m:
        return out
    line = m.group(0)

    # helper to fetch token like "rtt:1.547/2.559"
    def get_tok(key: str) -> Optional[str]:
        mm = re.search(rf"\b{re.escape(key)}:([^\s]+)", line)
        return mm.group(1) if mm else None

    # rtt is "x/y"
    rtt = get_tok("rtt")
    if rtt and "/" in rtt:
        a, b = rtt.split("/", 1)
        try:
            out["tcp_rtt_ms"] = float(a)
            out["tcp_rtt_var_ms"] = float(b)
        except ValueError:
            pass

    # rto is in ms (integer)
    rto = get_tok("rto")
    if rto:
        try:
            out["tcp_rto_ms"] = float(rto)
        except ValueError:
            pass

    cwnd = get_tok("cwnd")
    if cwnd:
        try:
            out["tcp_cwnd"] = int(float(cwnd))
        except ValueError:
            pass

    bytes_retrans = get_tok("bytes_retrans")
    if bytes_retrans:
        try:
            out["tcp_bytes_retrans"] = int(float(bytes_retrans))
        except ValueError:
            pass

    segs_out = get_tok("segs_out")
    if segs_out:
        try:
            out["tcp_segs_out"] = int(float(segs_out))
        except ValueError:
            pass

    segs_in = get_tok("segs_in")
    if segs_in:
        try:
            out["tcp_segs_in"] = int(float(segs_in))
        except ValueError:
            pass

    retrans = get_tok("retrans")
    if retrans and "/" in retrans:
        a, b = retrans.split("/", 1)
        try:
            out["tcp_retrans_inflight"] = int(float(a))
            out["tcp_retrans_total"] = int(float(b))
        except ValueError:
            pass

    return out


def parse_nstat(sys_net_txt: str) -> Dict[str, Any]:
    """
    Parse 'nstat' block lines like:
      TcpRetransSegs                  230                0.0
    Returns a dictionary of selected counters.
    """
    wanted = {
        "TcpRetransSegs",
        "TcpTimeouts",
        "TcpAttemptFails",
        "TcpEstabResets",
        "TcpInErrs",
        "IpOutDiscards",
        "TcpOutSegs",
        "TcpInSegs",
        # TcpExt interesting
        "TcpExtTCPLostRetransmit",
        "TcpExtTCPLossUndo",
        "TcpExtTCPLossProbes",
        "TcpExtTCPLossProbeRecovery",
        "TcpExtDelayedACKs",
        "TcpExtDelayedACKLost",
        "TcpExtTCPOFOQueue",
    }
    out: Dict[str, Any] = {}
    for line in sys_net_txt.splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        parts = re.split(r"\s+", line)
        if len(parts) >= 2 and parts[0] in wanted:
            try:
                out[parts[0]] = int(float(parts[1]))
            except ValueError:
                pass
    return out


def parse_qdisc_stats(text: str) -> Dict[str, Any]:
    """
    Parse qdisc header and Sent/dropped line:
      qdisc netem 1: root refcnt 2 limit 1000
      Sent 103678816 bytes 119425 pkt (dropped 0, overlimits 0 requeues 0)
    """
    out: Dict[str, Any] = {}
    m = re.search(r"qdisc\s+(\S+)\s+.*\blimit\s+(\d+)", text)
    if m:
        out["qdisc_type"] = m.group(1)
        out["qdisc_limit_pkts"] = int(m.group(2))
    m2 = re.search(
        r"Sent\s+(\d+)\s+bytes\s+(\d+)\s+pkt\s+\(dropped\s+(\d+),\s+overlimits\s+(\d+)\s+requeues\s+(\d+)\)",
        text,
    )
    if m2:
        out["qdisc_sent_bytes"] = int(m2.group(1))
        out["qdisc_sent_pkts"] = int(m2.group(2))
        out["qdisc_dropped_pkts"] = int(m2.group(3))
        out["qdisc_overlimits"] = int(m2.group(4))
        out["qdisc_requeues"] = int(m2.group(5))
    return out


def fio_percentile_ns_to_ms(
    percentile_ns: Dict[str, Any], pct: float
) -> Optional[float]:
    """
    fio stores percentiles with keys like "99.000000": 18481152 (ns).
    We pick the closest key to pct.
    """
    if not percentile_ns:
        return None
    keys = []
    for k in percentile_ns.keys():
        try:
            keys.append(float(k))
        except ValueError:
            continue
    if not keys:
        return None
    keys.sort()
    closest = min(keys, key=lambda x: abs(x - pct))
    ns = percentile_ns.get(f"{closest:.6f}")
    if ns is None:
        # some fio dumps may have different formatting; try string match
        for kk, vv in percentile_ns.items():
            try:
                if abs(float(kk) - pct) < 1e-6:
                    ns = vv
                    break
            except ValueError:
                continue
    if ns is None:
        return None
    try:
        return float(ns) / 1e6
    except Exception:
        return None


def parse_fio_json(path: Path) -> List[Dict[str, Any]]:
    """
    Return list of rows. Usually one job per file.
    We output two rows max: read and/or write, depending on which has ios.
    """
    data = json.loads(path.read_text(encoding="utf-8", errors="ignore"))
    rows: List[Dict[str, Any]] = []
    fio_ver = data.get("fio version")
    jobs = data.get("jobs") or []
    for job in jobs:
        jobname = job.get("jobname")
        job_opts = job.get("job options") or {}
        bs = job_opts.get("bs")
        rw = job_opts.get("rw")
        for direction in ("read", "write"):
            d = job.get(direction) or {}
            total_ios = d.get("total_ios", 0) or 0
            io_bytes = d.get("io_bytes", 0) or 0
            if total_ios == 0 and io_bytes == 0:
                continue
            slat = d.get("slat_ns") or {}
            clat = d.get("clat_ns") or {}
            lat = d.get("lat_ns") or {}
            p = (clat.get("percentile") or {}) if isinstance(clat, dict) else {}
            row = {
                "jobname": jobname,
                "rw": direction,
                "fio_version": fio_ver,
                "fio_rw": rw,
                "bs": bs,
                "runtime_ms": d.get("runtime"),
                "io_bytes": io_bytes,
                "bw_kib_s": d.get("bw"),  # fio 'bw' is KiB/s
                "bw_bytes_s": d.get("bw_bytes"),
                "iops": d.get("iops"),
                "total_ios": total_ios,
                "slat_mean_ms": (slat.get("mean", 0.0) / 1e6) if slat else None,
                "slat_max_ms": (slat.get("max", 0.0) / 1e6) if slat else None,
                "clat_mean_ms": (clat.get("mean", 0.0) / 1e6) if clat else None,
                "clat_max_ms": (clat.get("max", 0.0) / 1e6) if clat else None,
                "lat_mean_ms": (lat.get("mean", 0.0) / 1e6) if lat else None,
                "lat_max_ms": (lat.get("max", 0.0) / 1e6) if lat else None,
                "clat_p50_ms": fio_percentile_ns_to_ms(p, 50.0),
                "clat_p95_ms": fio_percentile_ns_to_ms(p, 95.0),
                "clat_p99_ms": fio_percentile_ns_to_ms(p, 99.0),
                "clat_p99_9_ms": fio_percentile_ns_to_ms(p, 99.9),
                "clat_p99_99_ms": fio_percentile_ns_to_ms(p, 99.99),
                "clat_p99_999_ms": fio_percentile_ns_to_ms(p, 99.999),
                "source_file": str(path),
            }
            rows.append(row)
    return rows


def find_profiles(root: Path) -> List[Path]:
    return sorted(root.rglob("profile.env"))


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", type=str, required=True, help="Path to results/ root")
    ap.add_argument("--out", type=str, required=True, help="Output directory")
    args = ap.parse_args()

    root = Path(args.root).resolve()
    out = Path(args.out).resolve()
    out.mkdir(parents=True, exist_ok=True)

    profile_rows: List[Dict[str, Any]] = []
    fio_rows: List[Dict[str, Any]] = []
    net_rows: List[Dict[str, Any]] = []

    for env_path in find_profiles(root):
        run_dir = env_path.parent
        profile_id = run_dir.name

        env = parse_env(env_path)

        delay_ms = _to_ms(env.get("DELAY", ""))
        jitter_ms = _to_ms(env.get("JITTER", ""))
        loss = env.get("LOSS", "").strip().rstrip("%")
        loss_pct = float(loss) if loss else None
        limit = env.get("LIMIT", "")
        limit_pkts = int(limit) if limit.isdigit() else None

        iface = None
        usbip_target = None
        kernel = None
        fio_version = None
        fs_mount = None

        preflight = run_dir / "preflight.txt"
        if preflight.exists():
            text = preflight.read_text(encoding="utf-8", errors="ignore")
            m = re.search(r"^###\s*iface\s*$\s*([^\n]+)", text, re.M)
            if m:
                iface = m.group(1).strip()
            m = re.search(r"^###\s*usbip_target\s*$\s*([^\n]+)", text, re.M)
            if m:
                usbip_target = m.group(1).strip()
            m = re.search(r"^###\s*uname\s*$\s*([^\n]+)", text, re.M)
            if m:
                kernel = m.group(1).strip()
            m = re.search(r"^###\s*mount_line\s*$\s*([^\n]+)", text, re.M)
            if m:
                fs_mount = m.group(1).strip()
            m = re.search(
                r"^###\s*fio_version\s*$\s*([^\n]+)\s*$\s*([^\n]+)", text, re.M
            )
            if m:
                # second line contains fio-3.xx
                fio_version = m.group(2).strip()

        # qdisc stats
        qdisc_after = run_dir / "tc_qdisc_after.txt"
        qstats = {}
        if qdisc_after.exists():
            qstats = parse_qdisc_stats(
                qdisc_after.read_text(encoding="utf-8", errors="ignore")
            )

        # TCP stats
        sys_net = run_dir / "sys_net.txt"
        net = {"profile_id": profile_id}
        if sys_net.exists():
            stext = sys_net.read_text(encoding="utf-8", errors="ignore")
            net.update(parse_ss_rtt(stext))
            net.update(parse_nstat(stext))

        # collect profile row
        profile_rows.append(
            {
                "profile_id": profile_id,
                "run_dir": str(run_dir),
                "iface": iface,
                "usbip_target": usbip_target,
                "kernel": kernel,
                "fio_version": fio_version or env.get("fio_version"),
                "mount_line": fs_mount,
                "delay_ms": delay_ms,
                "jitter_ms": jitter_ms,
                "loss_pct": loss_pct,
                "netem_limit_pkts": limit_pkts,
                **{f"qdisc_{k}": v for k, v in qstats.items()},
            }
        )

        # collect fio rows
        fio_dir = run_dir / "fio"
        if fio_dir.exists():
            for json_path in sorted(fio_dir.glob("*.json")):
                for r in parse_fio_json(json_path):
                    r["profile_id"] = profile_id
                    r["delay_ms"] = delay_ms
                    r["jitter_ms"] = jitter_ms
                    r["loss_pct"] = loss_pct
                    fio_rows.append(r)

        # collect net row if any data exists
        if len(net) > 1:
            net_rows.append(net)

    df_profiles = pd.DataFrame(profile_rows).sort_values(
        ["delay_ms", "profile_id"], na_position="last"
    )
    df_fio = pd.DataFrame(fio_rows)
    df_net = pd.DataFrame(net_rows)

    # save base tables
    df_profiles.to_csv(out / "profile_table.csv", index=False)
    df_fio.to_csv(out / "fio_table.csv", index=False)
    df_net.to_csv(out / "net_table.csv", index=False)

    # joined summary
    df = df_fio.merge(
        df_profiles,
        on=["profile_id", "delay_ms", "jitter_ms", "loss_pct"],
        how="left",
        suffixes=("", "_profile"),
    )
    if not df_net.empty:
        df = df.merge(df_net, on="profile_id", how="left", suffixes=("", "_net"))
    df.to_csv(out / "summary.csv", index=False)

    print(f"Wrote: {out / 'profile_table.csv'}")
    print(f"Wrote: {out / 'fio_table.csv'}")
    print(f"Wrote: {out / 'net_table.csv'}")
    print(f"Wrote: {out / 'summary.csv'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
