#!/usr/bin/env python3
"""
Add a formal dark suit + tie onto the presenter cutout (torso region).

The subject is cropped roughly at the chest in the source photo, so we draw
a tailored suit-shoulders shape that follows the torso silhouette: dark navy
jacket, white shirt collar wedge, dark tie, subtle lapel highlights.
Everything is drawn *under* the subject alpha for the collar area but *over*
the lower torso for the jacket body — the simplest robust trick: draw the
suit a few pixels larger than the subject's lower-third silhouette, then
re-composite the subject's head/torso (top 55%) back on top.
"""
import os
from PIL import Image, ImageDraw, ImageFilter, ImageEnhance

SRC = os.path.join(os.path.dirname(__file__), "..", "out", "avatar-square.png")
OUT = os.path.join(os.path.dirname(__file__), "..", "out", "avatar-suit.png")

NAVY_JACKET = (28, 34, 48)
NAVY_DARK = (18, 22, 32)
SHIRT = (235, 238, 242)
TIE = (110, 26, 30)
LAPEL_LIGHT = (52, 62, 84)

img = Image.open(SRC).convert("RGBA")
W, H = img.size
alpha = img.split()[3]

# subject silhouette bbox
bbox = alpha.getbbox()
sx0, sy0, sx1, sy1 = bbox
sw, sh = sx1 - sx0, sy1 - sy0
cx = (sx0 + sx1) // 2

# torso region: below ~42% height of the subject
torso_top = sy0 + int(sh * 0.42)

# Build suit layer: a broad shoulders shape spanning wider than the subject,
# tapering down and out of frame (bottom).
suit = Image.new("RGBA", (W, H), (0, 0, 0, 0))
d = ImageDraw.Draw(suit)

shoulder_w = int(sw * 0.78)          # half-width of shoulders
shoulder_y = torso_top + int(sh * 0.02)
d.polygon([
    (cx - shoulder_w, H),                       # bottom-left (out of frame)
    (cx - shoulder_w, shoulder_y + int(sh*0.10)),  # left shoulder top
    (cx - int(shoulder_w*0.55), shoulder_y),       # left shoulder slope start
    (cx, shoulder_y - int(sh*0.015)),              # collar center top
    (cx + int(shoulder_w*0.55), shoulder_y),
    (cx + shoulder_w, shoulder_y + int(sh*0.10)),
    (cx + shoulder_w, H),
], fill=NAVY_JACKET + (255,))

# jacket shading: darker at edges
shade = Image.new("RGBA", (W, H), (0, 0, 0, 0))
sd = ImageDraw.Draw(shade)
sd.polygon([
    (cx - shoulder_w, H),
    (cx - shoulder_w, shoulder_y + int(sh*0.10)),
    (cx - int(shoulder_w*0.55), shoulder_y),
    (cx - int(shoulder_w*0.30), shoulder_y + int(sh*0.06)),
    (cx - int(shoulder_w*0.38), H),
], fill=NAVY_DARK + (200,))
sd.polygon([
    (cx + shoulder_w, H),
    (cx + shoulder_w, shoulder_y + int(sh*0.10)),
    (cx + int(shoulder_w*0.55), shoulder_y),
    (cx + int(shoulder_w*0.30), shoulder_y + int(sh*0.06)),
    (cx + int(shoulder_w*0.38), H),
], fill=NAVY_DARK + (200,))
shade = shade.filter(ImageFilter.GaussianBlur(18))
suit.alpha_composite(shade)

# white shirt V under the chin
shirt_v_top = shoulder_y - int(sh * 0.005)
d.polygon([
    (cx - int(sw*0.10), shirt_v_top),
    (cx + int(sw*0.10), shirt_v_top),
    (cx + int(sw*0.035), shirt_v_top + int(sh*0.13)),
    (cx - int(sw*0.035), shirt_v_top + int(sh*0.13)),
], fill=SHIRT + (255,))

# tie: from collar down the V then straight
tie_w = int(sw * 0.045)
tie_knot_y = shirt_v_top + int(sh*0.012)
d.polygon([  # knot
    (cx - tie_w, tie_knot_y),
    (cx + tie_w, tie_knot_y),
    (cx + int(tie_w*0.8), tie_knot_y + int(sh*0.028)),
    (cx - int(tie_w*0.8), tie_knot_y + int(sh*0.028)),
], fill=TIE + (255,))
d.polygon([  # blade
    (cx - int(tie_w*0.8), tie_knot_y + int(sh*0.028)),
    (cx + int(tie_w*0.8), tie_knot_y + int(sh*0.028)),
    (cx + int(tie_w*1.1), shirt_v_top + int(sh*0.16)),
    (cx - int(tie_w*1.1), shirt_v_top + int(sh*0.16)),
], fill=TIE + (255,))

# lapel lines
for sign in (-1, 1):
    d.line([
        (cx + sign*int(sw*0.10), shirt_v_top),
        (cx + sign*int(sw*0.20), shirt_v_top + int(sh*0.16)),
    ], fill=LAPEL_LIGHT + (230,), width=max(3, int(sw*0.012)))

suit = suit.filter(ImageFilter.GaussianBlur(1.2))

# Composite order: backdrop(original) -> suit -> subject(top portion re-on-top)
out = img.copy()
out.alpha_composite(suit)

# Re-composite the subject's head/upper-chest back on top so neck/face
# stay in front of the collar.
head = img.crop((0, 0, W, torso_top + int(sh * 0.06)))
out.alpha_composite(head, (0, 0))

# soften the seam
seam_y = torso_top + int(sh * 0.055)
seam = Image.new("RGBA", (W, 40), (0, 0, 0, 0))
sd2 = ImageDraw.Draw(seam)
for i in range(40):
    a = int(90 * (1 - abs(i - 20) / 20))
    sd2.line([(0, i), (W, i)], fill=(20, 24, 34, a))
out.alpha_composite(seam, (0, seam_y - 20))

out = ImageEnhance.Contrast(out).enhance(1.03)
out.save(OUT, "PNG")
print("✓", os.path.abspath(OUT))
