import { NextResponse, type NextRequest } from "next/server";

const CANONICAL_HOST = "warden.blog";

// Any request arriving on a non-canonical host (www or the legacy
// vercel.app domain) is permanently redirected to the canonical host so
// Google consolidates all signals on one origin instead of dropping them.
const REDIRECT_HOSTS = new Set([
  "www.warden.blog",
  "warden-six-rouge.vercel.app",
]);

export function middleware(request: NextRequest) {
  const host = (request.headers.get("host") ?? "").toLowerCase();
  if (!REDIRECT_HOSTS.has(host)) {
    return NextResponse.next();
  }

  const url = new URL(request.url);
  return NextResponse.redirect(
    `https://${CANONICAL_HOST}${url.pathname}${url.search}`,
    308
  );
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
