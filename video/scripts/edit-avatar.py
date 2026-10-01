#!/usr/bin/env python3
"""
Professional avatar edit for the Warden presenter photo.

1. rembg   — cut the subject out of the busy background
2. grade   — even lighting on the face, tasteful contrast/saturation,
             subtle warmth to match the video's dark-premium palette
3. compose — place on a clean dark studio backdrop (#09090b gradient with a
             soft green key-light glow, matching Warden branding)
4. crop    — 4:5 portrait (LinkedIn/X avatar friendly) + a square headshot

Output: video/out/avatar-portrait.png, avatar-square.png
"""
import os
from io import BytesIO

from PIL import Image, ImageDraw, ImageEnhance, ImageFilter, ImageOps
from rembg import remove, new_session

SRC = os.path.expanduser("~/Downloads/Untitled.jpeg")
OUT_DIR = os.path.join(os.path.dirname(__file__), "..", "out")
os.makedirs(OUT_DIR, exist_ok=True)

# ── 1. subject cutout ────────────────────────────────────────────────
print("• Removing background (rembg / u2net)…")
session = new_session("u2net")
src = Image.open(SRC).convert("RGB")
cut = remove(src, session=session, alpha_matting=True,
             alpha_matting_foreground_threshold=240,
             alpha_matting_background_threshold=15,
             alpha_matting_erode_size=8)

# trim transparent margins
bbox = cut.getbbox()
cut = cut.crop(bbox)
print(f"  subject: {cut.size}")

# ── 2. color grade ───────────────────────────────────────────────────
rgb = cut.convert("RGB")
rgb = ImageEnhance.Brightness(rgb).enhance(1.06)   # lift exposure a touch
rgb = ImageEnhance.Contrast(rgb).enhance(1.12)     # crisp
rgb = ImageEnhance.Color(rgb).enhance(1.05)        # gentle saturation

# soft warmth: +R, -B
r, g, b = rgb.split()
r = r.point(lambda v: min(255, int(v * 1.03)))
b = b.point(lambda v: int(v * 0.97))
rgb = Image.merge("RGB", (r, g, b))

# unsharp mask for detail
rgb = rgb.filter(ImageFilter.UnsharpMask(radius=2, percent=90, threshold=3))

# faint vignette on the subject edges for depth
w, h = rgb.size
vign = Image.new("L", (w, h), 0)
d = ImageDraw.Draw(vign)
d.ellipse([-w * 0.25, -h * 0.2, w * 1.25, h * 1.2], fill=70)
vign = vign.filter(ImageFilter.GaussianBlur(120))
dark = ImageEnhance.Brightness(rgb).enhance(0.82)
rgb = Image.composite(dark, rgb, vign)

graded = Image.merge("RGBA", (*rgb.split(), cut.split()[3]))

# ── 3. studio backdrop (Warden dark + green key light) ───────────────
def make_backdrop(W, H):
    bg = Image.new("RGB", (W, H))
    px = bg.load()
    for y in range(H):
        t = y / H
        # vertical gradient #16161a -> #09090b
        rr = int(0x16 + (0x09 - 0x16) * t)
        gg = int(0x16 + (0x09 - 0x16) * t)
        bb = int(0x1a + (0x0b - 0x1a) * t)
        for x in range(W):
            px[x, y] = (rr, gg, bb)
    # soft green key-light glow, upper-left (matches #16a34a branding)
    glow = Image.new("RGB", (W, H), (0, 0, 0))
    gd = ImageDraw.Draw(glow)
    gd.ellipse([-W * 0.35, -H * 0.25, W * 0.55, H * 0.45],
               fill=(22, 70, 40))
    glow = glow.filter(ImageFilter.GaussianBlur(int(W * 0.28)))
    bg = Image.blend(bg, Image.blend(bg, glow, 0.55), 0.85)
    # subtle grid texture like the video scenes
    grid = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    gdr = ImageDraw.Draw(grid)
    step = max(W, H) // 28
    for x in range(0, W, step):
        gdr.line([(x, 0), (x, H)], fill=(255, 255, 255, 5))
    for y in range(0, H, step):
        gdr.line([(0, y), (W, y)], fill=(255, 255, 255, 5))
    bg = bg.convert("RGBA")
    bg.alpha_composite(grid)
    return bg

def compose(W, H, scale, y_off=0.0):
    bg = make_backdrop(W, H)
    s = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    tw = int(W * scale)
    th = int(tw * graded.height / graded.width)
    sub = graded.resize((tw, th), Image.LANCZOS)
    # contact shadow
    sh = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    alpha = sub.split()[3].point(lambda v: int(v * 0.45))
    shadow = Image.merge("RGBA", ((0,), (0,), (0,), (alpha,))) if False else None
    shm = Image.new("RGBA", sub.size, (0, 0, 0, 0))
    shm.putalpha(alpha)
    sh.alpha_composite(shm, (int(W * 0.52), int(H * (0.9 + y_off))))
    sh = sh.filter(ImageFilter.GaussianBlur(30))
    bg.alpha_composite(sh)
    x = (W - tw) // 2
    y = int(H * y_off)
    bg.alpha_composite(sub, (x, y))
    return bg.convert("RGB")

print("• Composing portraits…")
portrait = compose(1200, 1500, scale=0.96, y_off=0.06)   # 4:5
square = compose(1400, 1400, scale=0.9, y_off=0.10)      # square

# final gentle grade on the whole frame
for img, name in [(portrait, "avatar-portrait.png"), (square, "avatar-square.png")]:
    img = ImageEnhance.Contrast(img).enhance(1.04)
    p = os.path.join(OUT_DIR, name)
    img.save(p, "PNG")
    print(f"✓ {p}  {img.size}")
