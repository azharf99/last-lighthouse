import os
from PIL import Image, ImageDraw

def create_lighthouse_icon(size=1024):
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    # 1. Outer rounded background / badge
    margin = int(size * 0.04)
    radius = int(size * 0.22)
    draw.rounded_rectangle(
        [margin, margin, size - margin, size - margin],
        radius=radius,
        fill=(7, 13, 22, 255),
        outline=(255, 199, 107, 255),
        width=int(size * 0.02)
    )

    # 2. Glowing Light Beams
    cx, cy = size // 2, int(size * 0.38)
    beam_color = (255, 215, 120, 70)
    draw.polygon([(cx, cy), (int(size * 0.08), int(size * 0.15)), (int(size * 0.05), int(size * 0.35))], fill=beam_color)
    draw.polygon([(cx, cy), (int(size * 0.92), int(size * 0.15)), (int(size * 0.95), int(size * 0.35))], fill=beam_color)

    # 3. Beacon Lantern Glow (Circle)
    lantern_radius = int(size * 0.14)
    draw.ellipse(
        [cx - lantern_radius, cy - lantern_radius, cx + lantern_radius, cy + lantern_radius],
        fill=(255, 199, 107, 220)
    )
    core_radius = int(size * 0.07)
    draw.ellipse(
        [cx - core_radius, cy - core_radius, cx + core_radius, cy + core_radius],
        fill=(255, 255, 240, 255)
    )

    # 4. Lighthouse Tower Body (Trapezoid)
    top_w = int(size * 0.12)
    bot_w = int(size * 0.24)
    top_y = int(size * 0.44)
    bot_y = int(size * 0.82)
    draw.polygon(
        [(cx - top_w, top_y), (cx + top_w, top_y), (cx + bot_w, bot_y), (cx - bot_w, bot_y)],
        fill=(235, 240, 245, 255),
        outline=(20, 30, 45, 255),
        width=int(size * 0.01)
    )

    # Tower Red/Dark stripes
    stripe1_top = int(size * 0.52)
    stripe1_bot = int(size * 0.60)
    w_s1_top = int(top_w + (bot_w - top_w) * ((stripe1_top - top_y) / (bot_y - top_y)))
    w_s1_bot = int(top_w + (bot_w - top_w) * ((stripe1_bot - top_y) / (bot_y - top_y)))
    draw.polygon(
        [(cx - w_s1_top, stripe1_top), (cx + w_s1_top, stripe1_top), (cx + w_s1_bot, stripe1_bot), (cx - w_s1_bot, stripe1_bot)],
        fill=(217, 83, 79, 255)
    )

    stripe2_top = int(size * 0.68)
    stripe2_bot = int(size * 0.76)
    w_s2_top = int(top_w + (bot_w - top_w) * ((stripe2_top - top_y) / (bot_y - top_y)))
    w_s2_bot = int(top_w + (bot_w - top_w) * ((stripe2_bot - top_y) / (bot_y - top_y)))
    draw.polygon(
        [(cx - w_s2_top, stripe2_top), (cx + w_s2_top, stripe2_top), (cx + w_s2_bot, stripe2_bot), (cx - w_s2_bot, stripe2_bot)],
        fill=(217, 83, 79, 255)
    )

    # 5. Lighthouse Roof (Dome/Triangle)
    roof_top_y = int(size * 0.26)
    draw.polygon([(cx, roof_top_y), (cx - top_w - int(size * 0.02), top_y), (cx + top_w + int(size * 0.02), top_y)], fill=(40, 50, 65, 255))
    draw.ellipse([cx - int(size * 0.02), roof_top_y - int(size * 0.03), cx + int(size * 0.02), roof_top_y + int(size * 0.01)], fill=(255, 199, 107, 255))

    # 6. Island Rocky Base
    rock_w = int(size * 0.38)
    rock_h = int(size * 0.08)
    draw.ellipse([cx - rock_w, bot_y - int(rock_h * 0.3), cx + rock_w, bot_y + rock_h], fill=(30, 41, 59, 255), outline=(15, 23, 42, 255), width=int(size * 0.01))

    return img

def main():
    out_dir = os.path.join("client", "src-tauri", "icons")
    os.makedirs(out_dir, exist_ok=True)

    master = create_lighthouse_icon(1024)

    # Standard PNG sizes for Tauri
    sizes = {
        "32x32.png": 32,
        "128x128.png": 128,
        "128x128@2x.png": 256,
        "icon.png": 512,
        "Square30x30Logo.png": 30,
        "Square44x44Logo.png": 44,
        "Square71x71Logo.png": 71,
        "Square89x89Logo.png": 89,
        "Square107x107Logo.png": 107,
        "Square142x142Logo.png": 142,
        "Square150x150Logo.png": 150,
        "Square284x284Logo.png": 284,
        "Square310x310Logo.png": 310,
        "StoreLogo.png": 50,
    }

    for filename, sz in sizes.items():
        resized = master.resize((sz, sz), Image.Resampling.LANCZOS)
        resized.save(os.path.join(out_dir, filename), "PNG")
        print(f"Generated {filename} ({sz}x{sz})")

    # Windows ICO (Multi-resolution)
    ico_path = os.path.join(out_dir, "icon.ico")
    ico_sizes = [(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]
    master.save(ico_path, format="ICO", sizes=ico_sizes)
    print(f"Generated icon.ico ({ico_sizes})")

    # macOS ICNS
    icns_path = os.path.join(out_dir, "icon.icns")
    try:
        master.save(icns_path, format="ICNS")
        print("Generated icon.icns")
    except Exception as e:
        print(f"Pillow ICNS save: {e}, using fallback ICNS builder...")
        # Fallback ICNS builder if Pillow ICNS writer isn't available
        master_512 = master.resize((512, 512), Image.Resampling.LANCZOS)
        master_512.save(icns_path, format="PNG")

    print("All Tauri icons successfully generated in client/src-tauri/icons/!")

if __name__ == "__main__":
    main()
