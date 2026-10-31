import type { IconType } from "react-icons";
import { FaApple, FaDocker, FaLinux, FaWindows } from "react-icons/fa";
import SectionMarker from "@/components/bp/SectionMarker";
import Reveal from "@/components/bp/Reveal";

const platforms: { icon: IconType; name: string; detail: string }[] = [
  { icon: FaApple, name: "macOS", detail: "macOS 13+ on Apple Silicon" },
  { icon: FaLinux, name: "Linux", detail: "Ubuntu, Debian, Fedora, and more" },
  { icon: FaWindows, name: "Windows", detail: "Windows 11 x64" },
  { icon: FaDocker, name: "Docker", detail: "Run in Docker or Docker Compose" },
];

export default function RunAnywhere() {
  return (
    <section className="border-b border-hair">
      <div className="px-6 py-16 md:py-20">
        <div className="max-w-2xl">
          <SectionMarker index="08" />
          <h2 className="mt-4 font-mono text-[1.55rem] font-semibold leading-[1.18] tracking-[-0.01em] text-offwhite md:text-[1.95rem]">
            Run anywhere.
          </h2>
          <p className="mt-4 max-w-[470px] font-sans text-[14.5px] leading-[1.75] text-offwhite/45">
            Warden uses the sandbox primitives already present on your machine — no
            VM, no container requirement, no cloud relay.
          </p>
        </div>

        <Reveal className="mt-10 grid gap-px border border-hair bg-hair sm:grid-cols-2 lg:grid-cols-4">
          {platforms.map(({ icon: Icon, name, detail }) => (
            <div key={name} className="flex items-center gap-3 bg-cell px-5 py-5">
              <Icon size={20} className="shrink-0 text-offwhite/60" aria-hidden />
              <div className="min-w-0">
                <p className="font-mono text-[12px] font-semibold leading-tight text-offwhite/85">
                  {name}
                </p>
                <p className="mt-1 truncate font-sans text-[10.5px] leading-tight text-offwhite/30">
                  {detail}
                </p>
              </div>
            </div>
          ))}
        </Reveal>
      </div>
    </section>
  );
}
