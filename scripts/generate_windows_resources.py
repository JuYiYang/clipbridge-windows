from pathlib import Path

from PIL import Image


ROOT = Path(__file__).resolve().parents[1]
MAC_ASSET = (
    ROOT.parent
    / "clipbridge-macos"
    / "Maccy"
    / "Assets.xcassets"
    / "StatusBarMenuImage.imageset"
    / "DarkMenuBar-32w.png"
)


def make_icon() -> None:
    output = ROOT / "assets" / "clipbridge.ico"
    output.parent.mkdir(parents=True, exist_ok=True)

    source = Image.open(MAC_ASSET).convert("RGBA")
    sizes = [16, 24, 32, 48, 64, 128, 256]
    images = []
    for size in sizes:
      canvas = Image.new("RGBA", (size, size), (0, 0, 0, 0))
      glyph_size = max(1, round(size * 0.78))
      glyph = source.resize((glyph_size, glyph_size), Image.LANCZOS)
      offset = ((size - glyph_size) // 2, (size - glyph_size) // 2)
      canvas.alpha_composite(glyph, offset)
      images.append(canvas)

    images[0].save(output, sizes=[image.size for image in images], append_images=images[1:])


if __name__ == "__main__":
    make_icon()
