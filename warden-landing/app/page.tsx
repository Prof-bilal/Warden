import Nav from "@/components/Nav";
import Hero from "@/components/Hero";
import Problem from "@/components/Problem";
import HowItWorks from "@/components/HowItWorks";
import Backends from "@/components/Backends";
import DemoVideo from "@/components/DemoVideo";
import Windows from "@/components/Windows";
import Compatibility from "@/components/Compatibility";
import Cta from "@/components/Cta";
import Footer from "@/components/Footer";

export default function Home() {
  return (
    <main className="min-h-screen bg-ink-950">
      <Nav />
      <Hero />
      <Problem />
      <HowItWorks />
      <Backends />
      <DemoVideo />
      <Windows />
      <Compatibility />
      <Cta />
      <Footer />
    </main>
  );
}
