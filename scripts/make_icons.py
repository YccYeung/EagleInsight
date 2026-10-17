"""Generate the PWA icons (a green shield on a dark background) with only the standard library.

Run from the repo root:  python3 scripts/make_icons.py
"""

import struct
import zlib
from pathlib import Path

BACKGROUND = (15, 20, 25)
SHIELD = (62, 207, 142)
OUT_DIR = Path(__file__).resolve().parent.parent / "frontend" / "icons"


def in_shield(x: float, y: float) -> bool:
    """x and y are in 0..1. The shield stays inside the maskable-icon safe zone."""
    half_width = 0.24
    top, shoulder, tip = 0.27, 0.50, 0.76
    dx = abs(x - 0.5)
    if top <= y <= shoulder:
        return dx <= half_width
    if shoulder < y <= tip:
        # Sides curve in to a point at the bottom.
        t = (y - shoulder) / (tip - shoulder)
        return dx <= half_width * (1 - t * t)
    return False


def png(size: int) -> bytes:
    rows = bytearray()
    for py in range(size):
        rows.append(0)  # filter type: none
        for px in range(size):
            color = SHIELD if in_shield((px + 0.5) / size, (py + 0.5) / size) else BACKGROUND
            rows.extend(color)

    def chunk(kind: bytes, data: bytes) -> bytes:
        body = kind + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))

    header = struct.pack(">IIBBBBB", size, size, 8, 2, 0, 0, 0)  # 8-bit RGB
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", header)
        + chunk(b"IDAT", zlib.compress(bytes(rows), 9))
        + chunk(b"IEND", b"")
    )


if __name__ == "__main__":
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    for size in (192, 512):
        path = OUT_DIR / f"icon-{size}.png"
        path.write_bytes(png(size))
        print(f"wrote {path}")
