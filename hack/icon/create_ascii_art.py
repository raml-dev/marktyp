#!/usr/bin/env python3
"""Create the pixel-style Marktyp SVG source files used by the application."""

from pathlib import Path

PIXEL = 16
GLYPHS = {
    "M": ("10001", "11011", "10101", "10101", "10001", "10001", "10001"),
    "A": ("01110", "10001", "10001", "11111", "10001", "10001", "10001"),
    "R": ("11110", "10001", "10001", "11110", "10100", "10010", "10001"),
    "K": ("10001", "10010", "10100", "11000", "10100", "10010", "10001"),
    "T": ("11111", "00100", "00100", "00100", "00100", "00100", "00100"),
    "Y": ("10001", "10001", "01010", "00100", "00100", "00100", "00100"),
    "P": ("11110", "10001", "10001", "11110", "10000", "10000", "10000"),
}
LOGO = (
    "1111110",
    "1000010",
    "1011000",
    "1001100",
    "1000110",
    "1000010",
    "1111110",
)


def rects(rows: tuple[str, ...], x_offset: int = 0) -> list[str]:
    return [
        f'<rect x="{(x_offset + x) * PIXEL}" y="{y * PIXEL}" width="{PIXEL}" height="{PIXEL}"/>'
        for y, row in enumerate(rows)
        for x, value in enumerate(row)
        if value == "1"
    ]


def svg(width_cells: int, rows: list[str]) -> str:
    width = width_cells * PIXEL
    height = 7 * PIXEL
    return "\n".join(
        [
            '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" fill="#0ea5e9">' % (width, height),
            *rows,
            "</svg>",
            "",
        ]
    )


def main() -> None:
    root = Path(__file__).resolve().parent
    wordmark_rows: list[str] = []
    for index, letter in enumerate("MARKTYP"):
        wordmark_rows.extend(rects(GLYPHS[letter], index * 6))
    (root / "marktyp-ascii.svg").write_text(svg(41, wordmark_rows), encoding="utf-8")
    (root / "marktyp-logo.svg").write_text(svg(7, rects(LOGO)), encoding="utf-8")


if __name__ == "__main__":
    main()
