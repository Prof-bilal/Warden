import Image from "next/image";

export default function Diagram({
  src,
  alt,
  priority = false,
}: {
  src: string;
  alt: string;
  priority?: boolean;
}) {
  return (
    <Image
      src={src}
      alt={alt}
      width={1376}
      height={768}
      priority={priority}
      sizes="(max-width: 1024px) 90vw, 55vw"
      className="h-auto w-full rounded-[6px] object-contain"
    />
  );
}
