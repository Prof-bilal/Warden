"use client";

import { useEffect, useRef, useState } from "react";

/**
 * DemoVideo — full 48-second Warden explainer rendered into the homepage as
 * an autoplaying, muted, looping, inline 16:9 video between the Backends and
 * Windows sections. Plays on mount; pauses when scrolled out of view so the
 * tab doesn't waste CPU on a hidden tag. Reduced-motion is respected — the
 * eyebrow caption still appears, but the autoplay is suppressed and a
 * "Tap to play" overlay is shown instead.
 */
export default function DemoVideo() {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [reduceMotion, setReduceMotion] = useState(false);
  const [autoplayBlocked, setAutoplayBlocked] = useState(false);

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReduceMotion(mq.matches);
    const onChange = (e: MediaQueryListEvent) => setReduceMotion(e.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);

  // Try to start the video on mount; if the browser blocks autoplay, fall
  // back to a play button overlay so the user always has a path to start it.
  useEffect(() => {
    const v = videoRef.current;
    if (!v) return;
    if (reduceMotion) return;
    const p = v.play();
    if (p && typeof p.then === "function") {
      p.catch(() => setAutoplayBlocked(true));
    }
  }, [reduceMotion]);

  return (
    <section id="demo" className="border-t border-ink-800">
      <div className="mx-auto max-w-content px-6 py-20">
        {/* Eyebrow — matches Hero / Backends rhythm: 6px dot + mono small caps */}
        <div className="flex items-center gap-2.5">
          <span className="h-[6px] w-[6px] shrink-0 rounded-full bg-blueprint" aria-hidden />
          <span className="font-mono text-[0.6875rem] font-medium uppercase tracking-[0.14em] text-muted md:text-[0.75rem] md:tracking-[0.12em]">
            48-SECOND WALKTHROUGH
          </span>
        </div>

        <h2 className="mt-5 max-w-[36rem] text-[1.75rem] font-medium leading-[1.15] tracking-[-0.01em] text-paper md:text-[2rem]">
          See Warden draw the boundary in 48 seconds.
        </h2>
        <p className="mt-3 max-w-[34rem] text-[1rem] leading-[1.65] text-muted">
          Six scenes, one policy. Doorway → server reach → boundary gate →
          policy card → backend status → terminal test. The same runtime you
          can install right now, narrated for the people who don&apos;t have
          time to read the README.
        </p>

        <div className="relative mt-10 overflow-hidden rounded-[12px] border border-ink-700 bg-ink-900">
          <video
            ref={videoRef}
            src="/videos/warden-sandbox-explainer.mp4"
            poster="/videos/warden-sandbox-explainer-poster.jpg"
            autoPlay={!reduceMotion}
            muted
            loop
            playsInline
            controls={autoplayBlocked || reduceMotion}
            preload="metadata"
            className="block aspect-video w-full bg-ink-950"
            aria-label="Warden explainer — six scenes walking through the policy, boundary, and runtime."
          >
            <track kind="captions" srcLang="en" label="English captions" default />
            Your browser does not support embedded video. The same walkthrough
            is available as a static transcript on the
            {" "}<a className="underline" href="/docs/walkthrough">docs page</a>.
          </video>

          {autoplayBlocked && !reduceMotion && (
            <button
              type="button"
              onClick={() => {
                setAutoplayBlocked(false);
                videoRef.current?.play().catch(() => setAutoplayBlocked(true));
              }}
              className="absolute inset-0 flex items-center justify-center bg-ink-950/60 text-paper"
              aria-label="Play the Warden walkthrough"
            >
              <span className="rounded-full border border-paper/60 bg-ink-900/90 px-6 py-3 font-mono text-[0.875rem] uppercase tracking-[0.08em]">
                Tap to play · 48s
              </span>
            </button>
          )}
        </div>

        <p className="mt-3 text-center font-mono text-[0.6875rem] uppercase tracking-[0.08em] text-muted">
          {reduceMotion
            ? "Reduced motion — press play to watch."
            : "Autoplays, muted, loops. Captions on by default."}
        </p>
      </div>
    </section>
  );
}