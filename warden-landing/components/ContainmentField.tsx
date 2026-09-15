"use client";

import { useEffect, useRef } from "react";

type P = {
  x: number;
  y: number;
  vx: number;
  vy: number;
  r: number;
  kind: 0 | 1 | 2 | 3; // 0 muted, 1 grant, 2 deny, 3 blueprint
  seed: number;
};

type Flash = { x: number; y: number; t: number; life: number; kind: 1 | 2 | 3 };

const COLORS = {
  muted: "141,149,165",
  grant: "63,178,126",
  deny: "226,96,79",
  blueprint: "110,147,232",
};

/**
 * ContainmentFieldWarden-original hero background.
 * Distinct from CodeAtlas flow-field streamlines: instead of long curvy
 * lines, this renders discrete syscall-like particles trapped inside an
 * elliptical sandbox boundary. Grant particles drift inside, deny particles
 * strike the boundary and flash red before bouncing back (fail-closed).
 */
export default function ContainmentField() {
  const ref = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvasEl = ref.current;
    if (!canvasEl) return;
    const context = canvasEl.getContext("2d");
    if (!context) return;
    const canvas: HTMLCanvasElement = canvasEl;
    const ctx: CanvasRenderingContext2D = context;

    let w = 0;
    let h = 0;
    let raf = 0;
    let running = true;
    let particles: P[] = [];
    let flashes: Flash[] = [];
    let mx = -9999;
    let my = -9999;
    let smx = -9999; // smoothed cursor (spotlight + links)
    let smy = -9999;
    let hovering = false;
    let flashTimer = 0;

    const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const dpr = Math.min(window.devicePixelRatio || 1, 1.5);

    function boundary() {
      const cx = w / 2;
      const cy = h * 0.42;
      const rx = Math.min(w * 0.44, 620);
      const ry = Math.min(h * 0.44, 420);
      return { cx, cy, rx, ry };
    }

    function inside(x: number, y: number) {
      const { cx, cy, rx, ry } = boundary();
      const dx = (x - cx) / rx;
      const dy = (y - cy) / ry;
      return dx * dx + dy * dy;
    }

    function spawn(i: number): P {
      const { cx, cy, rx, ry } = boundary();
      // random point inside ellipse (rejection-free via sqrt distribution)
      const a = Math.random() * Math.PI * 2;
      const rr = Math.sqrt(Math.random());
      const x = cx + Math.cos(a) * rx * rr * 0.95;
      const y = cy + Math.sin(a) * ry * rr * 0.95;
      const speed = 0.15 + Math.random() * 0.45;
      const va = Math.random() * Math.PI * 2;
      const roll = Math.random();
      const kind: P["kind"] = roll < 0.62 ? 0 : roll < 0.76 ? 3 : roll < 0.9 ? 1 : 2;
      return {
        x,
        y,
        vx: Math.cos(va) * speed,
        vy: Math.sin(va) * speed,
        r: kind === 0 ? 1 + Math.random() * 1.2 : 1.4 + Math.random() * 1.4,
        kind,
        seed: i,
      };
    }

    function resize() {
      const rect = canvas.getBoundingClientRect();
      w = Math.max(1, Math.floor(rect.width));
      h = Math.max(1, Math.floor(rect.height));
      canvas.width = Math.floor(w * dpr);
      canvas.height = Math.floor(h * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      const count = Math.min(110, Math.max(36, Math.floor((w * h) / 22000)));
      particles = Array.from({ length: count }, (_, i) => spawn(i));
    }

    function boundaryPoint(angle: number) {
      const { cx, cy, rx, ry } = boundary();
      return { x: cx + Math.cos(angle) * rx, y: cy + Math.sin(angle) * ry };
    }

    function step() {
      ctx.clearRect(0, 0, w, h);
      const { cx, cy, rx, ry } = boundary();

      // smooth-follow cursor for spotlight / links
      if (!reduced) {
        if (hovering) {
          if (smx < -1000) {
            smx = mx;
            smy = my;
          } else {
            smx += (mx - smx) * 0.18;
            smy += (my - smy) * 0.18;
          }
        }
      }

      // ── sandbox boundary: dashed grant ellipse + faint blueprint outer ──
      ctx.save();
      ctx.setLineDash([6, 8]);
      ctx.strokeStyle = "rgba(63,178,126,0.22)";
      ctx.lineWidth = 1.2;
      ctx.beginPath();
      ctx.ellipse(cx, cy, rx, ry, 0, 0, Math.PI * 2);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.strokeStyle = "rgba(110,147,232,0.10)";
      ctx.lineWidth = 1;
      ctx.beginPath();
      ctx.ellipse(cx, cy, rx + 18, ry + 18, 0, 0, Math.PI * 2);
      ctx.stroke();
      // gate tick at bottom (allowed path)
      const gateA = Math.PI * 0.42;
      ctx.strokeStyle = "rgba(63,178,126,0.55)";
      ctx.lineWidth = 2;
      ctx.beginPath();
      ctx.ellipse(cx, cy, rx, ry, 0, gateA, gateA + 0.28);
      ctx.stroke();
      ctx.restore();

      // ── cursor spotlight (visible hover feedback) ──
      if (hovering && !reduced && smx > -1000) {
        const spot = ctx.createRadialGradient(smx, smy, 0, smx, smy, 230);
        spot.addColorStop(0, "rgba(63,178,126,0.10)");
        spot.addColorStop(0.5, "rgba(110,147,232,0.06)");
        spot.addColorStop(1, "rgba(0,0,0,0)");
        ctx.fillStyle = spot;
        ctx.beginPath();
        ctx.arc(smx, smy, 230, 0, Math.PI * 2);
        ctx.fill();
      }

      // ── deny flashes on the boundary ──
      flashTimer -= 1;
      if (flashTimer <= 0 && !reduced) {
        flashTimer = 40 + Math.random() * 70;
        const a = Math.random() * Math.PI * 2;
        const p = boundaryPoint(a);
        flashes.push({ x: p.x, y: p.y, t: 0, life: 46, kind: Math.random() < 0.7 ? 2 : 1 });
        if (flashes.length > 14) flashes.shift();
      }
      flashes = flashes.filter((f) => f.t < f.life);
      for (const f of flashes) {
        const k = f.t / f.life;
        const isShock = f.kind === 3;
        const rad = isShock ? 6 + k * 90 : 4 + k * 26;
        const alpha = (1 - k) * (isShock ? 0.45 : 0.5);
        const col = f.kind === 2 ? COLORS.deny : f.kind === 3 ? COLORS.blueprint : COLORS.grant;
        ctx.beginPath();
        ctx.arc(f.x, f.y, rad, 0, Math.PI * 2);
        ctx.strokeStyle = `rgba(${col},${alpha.toFixed(3)})`;
        ctx.lineWidth = 1.4;
        ctx.stroke();
        ctx.beginPath();
        ctx.arc(f.x, f.y, 2.2, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${col},${(alpha + 0.25).toFixed(3)})`;
        ctx.fill();
        if (!reduced) f.t += 1;
      }

      // ── faint constellation links (short segments, not streamlines) ──
      ctx.lineWidth = 1;
      for (let i = 0; i < particles.length; i++) {
        const a = particles[i];
        for (let j = i + 1; j < particles.length; j++) {
          const b = particles[j];
          const dx = a.x - b.x;
          const dy = a.y - b.y;
          const d2 = dx * dx + dy * dy;
          if (d2 < 110 * 110) {
            const alpha = (1 - Math.sqrt(d2) / 110) * 0.07;
            ctx.strokeStyle = `rgba(${COLORS.blueprint},${alpha.toFixed(3)})`;
            ctx.beginPath();
            ctx.moveTo(a.x, a.y);
            ctx.lineTo(b.x, b.y);
            ctx.stroke();
          }
        }
      }

      // ── particles ──
      const HOVER_R = 170;
      for (const p of particles) {
        if (!reduced) {
          // strong visible mouse repel + slight swirl
          const mdx = p.x - mx;
          const mdy = p.y - my;
          const md2 = mdx * mdx + mdy * mdy;
          if (hovering && md2 < HOVER_R * HOVER_R && md2 > 1) {
            const md = Math.sqrt(md2);
            const fall = 1 - md / HOVER_R; // 0..1, stronger when close
            const force = 0.02 + fall * 0.09;
            p.vx += (mdx / md) * force + (-mdy / md) * fall * 0.03;
            p.vy += (mdy / md) * force + (mdx / md) * fall * 0.03;
          }
          // damping + drift
          p.vx *= 0.995;
          p.vy *= 0.995;
          const base = 0.18 + ((p.seed % 5) * 0.06);
          const sp = Math.hypot(p.vx, p.vy) || 1;
          p.vx += ((p.vx / sp) * base - p.vx) * 0.02;
          p.vy += ((p.vy / sp) * base - p.vy) * 0.02;

          let nx = p.x + p.vx;
          let ny = p.y + p.vy;
          // bounce off ellipse boundary (fail-closed)
          if (inside(nx, ny) > 1) {
            // reflect velocity around normal
            const { cx: bcx, cy: bcy, rx: brx, ry: bry } = boundary();
            const nxn = (p.x - bcx) / (brx * brx);
            const nyn = (p.y - bcy) / (bry * bry);
            const nl = Math.hypot(nxn, nyn) || 1;
            const nxu = nxn / nl;
            const nyu = nyn / nl;
            const dot = p.vx * nxu + p.vy * nyu;
            p.vx -= 2 * dot * nxu;
            p.vy -= 2 * dot * nyu;
            nx = p.x + p.vx;
            ny = p.y + p.vy;
            if (p.kind === 2 && Math.random() < 0.12) {
              flashes.push({ x: p.x, y: p.y, t: 0, life: 34, kind: 2 });
            }
          }
          p.x = nx;
          p.y = ny;
        }

        const col =
          p.kind === 1
            ? COLORS.grant
            : p.kind === 2
              ? COLORS.deny
              : p.kind === 3
                ? COLORS.blueprint
                : COLORS.muted;
        const alpha = p.kind === 0 ? 0.5 : 0.85;
        ctx.beginPath();
        ctx.arc(p.x, p.y, p.r, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(${col},${alpha})`;
        ctx.fill();
        if (p.kind === 1 || p.kind === 2) {
          ctx.beginPath();
          ctx.arc(p.x, p.y, p.r * 2.6, 0, Math.PI * 2);
          ctx.fillStyle = `rgba(${col},0.10)`;
          ctx.fill();
        }
      }

      // ── cursor links + hub (network-to-pointer, clearly visible on hover) ──
      if (hovering && !reduced && smx > -1000) {
        const LINK_R = 170;
        ctx.lineWidth = 1;
        for (const p of particles) {
          const dx = p.x - smx;
          const dy = p.y - smy;
          const d = Math.hypot(dx, dy);
          if (d < LINK_R && d > 1) {
            const a = (1 - d / LINK_R) * 0.4;
            const col =
              p.kind === 1
                ? COLORS.grant
                : p.kind === 2
                  ? COLORS.deny
                  : COLORS.blueprint;
            ctx.strokeStyle = `rgba(${col},${a.toFixed(3)})`;
            ctx.beginPath();
            ctx.moveTo(smx, smy);
            ctx.lineTo(p.x, p.y);
            ctx.stroke();
          }
        }
        // hub dot + expanding ring
        ctx.beginPath();
        ctx.arc(smx, smy, 3, 0, Math.PI * 2);
        ctx.fillStyle = "rgba(63,178,126,0.9)";
        ctx.fill();
        ctx.beginPath();
        ctx.arc(smx, smy, 10, 0, Math.PI * 2);
        ctx.strokeStyle = "rgba(63,178,126,0.35)";
        ctx.lineWidth = 1.2;
        ctx.stroke();
      }

      if (running && !reduced) raf = requestAnimationFrame(step);
    }

    function onMove(e: PointerEvent) {
      const rect = canvas.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      // ignore when pointer is outside the hero canvas
      if (x < 0 || y < 0 || x > rect.width || y > rect.height) {
        hovering = false;
        mx = -9999;
        my = -9999;
        return;
      }
      hovering = true;
      mx = x;
      my = y;
    }
    function onLeave() {
      hovering = false;
      mx = -9999;
      my = -9999;
      smx = -9999;
      smy = -9999;
    }
    function onDown(e: PointerEvent) {
      if (reduced) return;
      const rect = canvas.getBoundingClientRect();
      const x = e.clientX - rect.left;
      const y = e.clientY - rect.top;
      if (x < 0 || y < 0 || x > rect.width || y > rect.height) return;
      // click burst: shockwave ring + radial push (policy probe pulse)
      flashes.push({ x, y, t: 0, life: 42, kind: 3 });
      if (flashes.length > 18) flashes.shift();
      for (const p of particles) {
        const dx = p.x - x;
        const dy = p.y - y;
        const d = Math.hypot(dx, dy);
        if (d < 220 && d > 1) {
          const fall = 1 - d / 220;
          p.vx += (dx / d) * fall * 3.2;
          p.vy += (dy / d) * fall * 3.2;
        }
      }
    }

    const io = new IntersectionObserver(
      (entries) => {
        const vis = entries[0]?.isIntersecting ?? true;
        if (vis && !reduced) {
          if (!running) {
            running = true;
            raf = requestAnimationFrame(step);
          }
        } else {
          running = vis;
          if (!vis) cancelAnimationFrame(raf);
          if (vis && reduced) step(); // single static frame
        }
      },
      { threshold: 0 }
    );

    resize();
    step();
    window.addEventListener("resize", resize);
    window.addEventListener("pointermove", onMove, { passive: true });
    window.addEventListener("pointerdown", onDown, { passive: true });
    canvas.parentElement?.addEventListener("pointermove", onMove);
    window.addEventListener("pointerleave", onLeave);
    document.documentElement.addEventListener("pointerleave", onLeave);
    io.observe(canvas);

    return () => {
      running = false;
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerdown", onDown);
      window.removeEventListener("pointerleave", onLeave);
      document.documentElement.removeEventListener("pointerleave", onLeave);
      io.disconnect();
    };
  }, []);

  return (
    <canvas
      ref={ref}
      className="pointer-events-none absolute inset-0 h-full w-full opacity-90"
      aria-hidden
    />
  );
}
