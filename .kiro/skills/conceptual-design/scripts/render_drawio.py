#!/usr/bin/env python3
"""
Render a draw.io (.drawio) XML file into an editable .drawio.svg file.

The output is a normal SVG image whose root <svg> element carries a
`content` attribute holding the original mxfile XML. draw.io / diagrams.net
opens such files for further editing, while browsers/IDEs/docs render them
as plain SVG images.

Usage:
    python render_drawio.py input.drawio [-o output.drawio.svg]

Supports the common subset of draw.io shapes used in conceptual /
architecture diagrams: rectangles (rounded or sharp), ellipses, rhombus,
cylinders (databases), plain text labels, container boxes, and edges
(straight or orthogonal, with optional waypoints, dashes and labels).
Only Python stdlib is used.
"""

import argparse
import html
import math
import re
import sys
import xml.etree.ElementTree as ET
from xml.sax.saxutils import escape, quoteattr

DEFAULTS = {
    "fillColor": "#ffffff",
    "strokeColor": "#000000",
    "fontColor": "#000000",
    "fontSize": 12,
    "strokeWidth": 1,
}

CHAR_W = 0.58  # average glyph width as fraction of font size (Helvetica-ish)


def parse_style(style_str):
    """Parse 'key=value;key2;...' drawio style string into a dict."""
    out = {}
    if not style_str:
        return out
    parts = [p for p in style_str.split(";") if p]
    if parts and "=" not in parts[0]:
        out["_shape"] = parts[0]  # leading token like 'ellipse', 'rhombus'
    for p in parts:
        if "=" in p:
            k, _, v = p.partition("=")
            out[k] = v
    return out


def strip_html(value):
    """Convert drawio HTML-ish labels to plain text lines."""
    if not value:
        return []
    value = re.sub(r"<br\s*/?>", "\n", value, flags=re.I)
    value = re.sub(r"</(div|p)>", "\n", value, flags=re.I)
    value = re.sub(r"<[^>]+>", "", value)
    value = html.unescape(value)
    lines = [ln.strip() for ln in value.split("\n")]
    return [ln for ln in lines if ln != ""] or ([""] if value.strip() == "" else [value.strip()])


def wrap_lines(lines, max_w, font_size):
    """Word-wrap lines so each fits max_w pixels (approximate)."""
    max_chars = max(4, int(max_w / (font_size * CHAR_W)))
    out = []
    for line in lines:
        if len(line) <= max_chars:
            out.append(line)
            continue
        words, cur = line.split(" "), ""
        for w in words:
            if cur and len(cur) + 1 + len(w) > max_chars:
                out.append(cur)
                cur = w
            else:
                cur = f"{cur} {w}".strip()
        if cur:
            out.append(cur)
    return out


class Cell:
    def __init__(self, el):
        self.id = el.get("id")
        self.value = el.get("value", "")
        self.style = parse_style(el.get("style", ""))
        self.parent = el.get("parent")
        self.source = el.get("source")
        self.target = el.get("target")
        self.is_vertex = el.get("vertex") == "1"
        self.is_edge = el.get("edge") == "1"
        geo = el.find("mxGeometry")
        self.geo = geo
        self.x = float(geo.get("x", 0)) if geo is not None else 0.0
        self.y = float(geo.get("y", 0)) if geo is not None else 0.0
        self.w = float(geo.get("width", 0)) if geo is not None else 0.0
        self.h = float(geo.get("height", 0)) if geo is not None else 0.0
        self.waypoints = []
        if geo is not None:
            arr = geo.find("Array[@as='points']")
            if arr is not None:
                for pt in arr.findall("mxPoint"):
                    self.waypoints.append((float(pt.get("x", 0)), float(pt.get("y", 0))))
        self.abs_x = self.x
        self.abs_y = self.y


def load_cells(mxfile_root):
    model = mxfile_root.find(".//mxGraphModel")
    if model is None:
        sys.exit("error: no mxGraphModel found in input")
    cells = {}
    order = []
    for el in model.find("root"):
        if el.tag not in ("mxCell", "object", "UserObject"):
            continue
        if el.tag in ("object", "UserObject"):
            inner = el.find("mxCell")
            if inner is None:
                continue
            label = el.get("label", "")
            merged = ET.Element("mxCell", {**inner.attrib, "id": el.get("id"), "value": label})
            for child in inner:
                merged.append(child)
            el = merged
        c = Cell(el)
        cells[c.id] = c
        order.append(c)
    # absolute coordinates (children of containers are relative to parent)
    def resolve(c, seen=()):
        if c.id in seen:
            return (0.0, 0.0)
        p = cells.get(c.parent)
        if p is None or not p.is_vertex:
            return (c.x, c.y)
        px, py = resolve(p, seen + (c.id,))
        return (px + c.x, py + c.y)

    for c in order:
        if c.is_vertex:
            c.abs_x, c.abs_y = resolve(c)
    return cells, order


def anchor_point(cell, key_x, key_y, style):
    ex, ey = style.get(key_x), style.get(key_y)
    if ex is not None and ey is not None:
        return (cell.abs_x + cell.w * float(ex), cell.abs_y + cell.h * float(ey))
    return None


def boundary_point(cell, toward):
    """Intersection of the line center->toward with the cell rectangle."""
    cx, cy = cell.abs_x + cell.w / 2, cell.abs_y + cell.h / 2
    dx, dy = toward[0] - cx, toward[1] - cy
    if dx == 0 and dy == 0:
        return (cx, cy)
    scale = math.inf
    if dx:
        scale = min(scale, (cell.w / 2) / abs(dx))
    if dy:
        scale = min(scale, (cell.h / 2) / abs(dy))
    return (cx + dx * scale, cy + dy * scale)


def route_edge(edge, cells):
    src, tgt = cells.get(edge.source), cells.get(edge.target)
    if src is None or tgt is None:
        return None
    pts = list(edge.waypoints)
    src_fixed = anchor_point(src, "exitX", "exitY", edge.style)
    tgt_fixed = anchor_point(tgt, "entryX", "entryY", edge.style)
    src_toward = pts[0] if pts else (tgt_fixed or (tgt.abs_x + tgt.w / 2, tgt.abs_y + tgt.h / 2))
    tgt_toward = pts[-1] if pts else (src_fixed or (src.abs_x + src.w / 2, src.abs_y + src.h / 2))
    p0 = src_fixed or boundary_point(src, src_toward)
    pn = tgt_fixed or boundary_point(tgt, tgt_toward)

    if pts:
        return [p0] + pts + [pn]

    if "orthogonal" in edge.style.get("edgeStyle", ""):
        x0, y0 = p0
        x1, y1 = pn
        if abs(x0 - x1) < 1 or abs(y0 - y1) < 1:
            return [p0, pn]
        # bend based on which axis dominates
        if abs(x1 - x0) > abs(y1 - y0):
            mx = (x0 + x1) / 2
            return [p0, (mx, y0), (mx, y1), pn]
        my = (y0 + y1) / 2
        return [p0, (x0, my), (x1, my), pn]
    return [p0, pn]


def text_block(lines, cx, top_y, font_size, color, bold, align="middle", anchor="middle"):
    if not lines:
        return ""
    lh = font_size * 1.25
    weight = ' font-weight="bold"' if bold else ""
    out = []
    for i, ln in enumerate(lines):
        y = top_y + lh * i + font_size
        out.append(
            f'<text x="{cx:.1f}" y="{y:.1f}" font-family="Helvetica, Arial, sans-serif" '
            f'font-size="{font_size}px" fill="{color}" text-anchor="{anchor}"{weight}>{escape(ln)}</text>'
        )
    return "\n".join(out)


def render_vertex(c):
    s = c.style
    shape = s.get("_shape", "") or s.get("shape", "")
    fill = s.get("fillColor", DEFAULTS["fillColor"])
    if fill == "none":
        fill = "none"
    stroke = s.get("strokeColor", DEFAULTS["strokeColor"])
    stroke_w = s.get("strokeWidth", DEFAULTS["strokeWidth"])
    font_color = s.get("fontColor", DEFAULTS["fontColor"])
    font_size = float(s.get("fontSize", DEFAULTS["fontSize"]))
    bold = s.get("fontStyle", "0") in ("1", "3", "5", "7")
    dashed = ' stroke-dasharray="6,4"' if s.get("dashed") == "1" else ""
    opacity = f' fill-opacity="{float(s.get("opacity"))/100:.2f}"' if s.get("opacity") else ""
    x, y, w, h = c.abs_x, c.abs_y, c.w, c.h
    body = ""

    is_text_only = shape == "text" or (fill == "none" and stroke == "none")
    if not is_text_only:
        common = f'fill="{fill}" stroke="{stroke}" stroke-width="{stroke_w}"{dashed}{opacity}'
        if shape == "ellipse":
            body = f'<ellipse cx="{x+w/2}" cy="{y+h/2}" rx="{w/2}" ry="{h/2}" {common}/>'
        elif shape == "rhombus":
            pts = f"{x+w/2},{y} {x+w},{y+h/2} {x+w/2},{y+h} {x},{y+h/2}"
            body = f'<polygon points="{pts}" {common}/>'
        elif "cylinder" in shape:
            ry = min(h * 0.15, 15)
            body = (
                f'<path d="M {x} {y+ry} A {w/2} {ry} 0 0 1 {x+w} {y+ry} '
                f'L {x+w} {y+h-ry} A {w/2} {ry} 0 0 1 {x} {y+h-ry} Z" {common}/>'
                f'<path d="M {x} {y+ry} A {w/2} {ry} 0 0 0 {x+w} {y+ry}" fill="none" '
                f'stroke="{stroke}" stroke-width="{stroke_w}"/>'
            )
        else:
            rx = ' rx="8" ry="8"' if s.get("rounded") == "1" else ""
            body = f'<rect x="{x}" y="{y}" width="{w}" height="{h}"{rx} {common}/>'

    lines = wrap_lines(strip_html(c.value), max(w - 8, 40), font_size)
    lh = font_size * 1.25
    block_h = lh * len(lines)
    valign = s.get("verticalAlign", "middle")
    extra_top = min(h * 0.15, 15) if "cylinder" in shape else 0  # push below cylinder cap
    if valign == "top":
        top = y + 6 + extra_top
    elif valign == "bottom":
        top = y + h - block_h - 4
    else:
        top = y + (h - block_h) / 2 + extra_top / 2
    body += "\n" + text_block(lines, x + w / 2, top, font_size, font_color, bold)
    return body


def render_edge(edge, cells):
    path_pts = route_edge(edge, cells)
    if not path_pts:
        return ""
    s = edge.style
    stroke = s.get("strokeColor", DEFAULTS["strokeColor"])
    stroke_w = s.get("strokeWidth", "1.5")
    dashed = ' stroke-dasharray="6,4"' if s.get("dashed") == "1" else ""
    no_arrow = s.get("endArrow") == "none"
    marker = "" if no_arrow else f' marker-end="url(#arrow-{stroke.lstrip("#")})"'
    d = "M " + " L ".join(f"{p[0]:.1f} {p[1]:.1f}" for p in path_pts)
    out = f'<path d="{d}" fill="none" stroke="{stroke}" stroke-width="{stroke_w}"{dashed}{marker}/>'

    lines = strip_html(edge.value)
    if lines:
        # midpoint of the polyline
        total = sum(math.dist(path_pts[i], path_pts[i + 1]) for i in range(len(path_pts) - 1))
        acc, mid = 0.0, path_pts[0]
        for i in range(len(path_pts) - 1):
            seg = math.dist(path_pts[i], path_pts[i + 1])
            if acc + seg >= total / 2 and seg > 0:
                t = (total / 2 - acc) / seg
                mid = (
                    path_pts[i][0] + (path_pts[i + 1][0] - path_pts[i][0]) * t,
                    path_pts[i][1] + (path_pts[i + 1][1] - path_pts[i][1]) * t,
                )
                break
            acc += seg
        font_size = float(s.get("fontSize", 11))
        wmax = max(len(ln) for ln in lines) * font_size * CHAR_W + 8
        hbox = len(lines) * font_size * 1.25 + 4
        out += (
            f'\n<rect x="{mid[0]-wmax/2:.1f}" y="{mid[1]-hbox/2:.1f}" width="{wmax:.1f}" '
            f'height="{hbox:.1f}" fill="#ffffff" fill-opacity="0.85" stroke="none"/>'
        )
        out += "\n" + text_block(
            lines, mid[0], mid[1] - hbox / 2, font_size, s.get("fontColor", "#333333"), False
        )
    return out


def render(mxfile_text):
    root = ET.fromstring(mxfile_text)
    cells, order = load_cells(root)

    vertices = [c for c in order if c.is_vertex and c.w > 0 and c.h > 0]
    edges = [c for c in order if c.is_edge]
    if not vertices:
        sys.exit("error: diagram has no vertices")

    # containers (bigger boxes) behind regular nodes
    child_ids = {c.parent for c in order}
    vertices.sort(key=lambda c: (0 if c.id in child_ids else 1, c.w * c.h * -1))

    parts = []
    for v in vertices:
        parts.append(render_vertex(v))
    for e in edges:
        parts.append(render_edge(e, cells))

    edge_colors = {e.style.get("strokeColor", DEFAULTS["strokeColor"]) for e in edges}
    markers = "\n".join(
        f'<marker id="arrow-{c.lstrip("#")}" viewBox="0 0 10 10" refX="9" refY="5" '
        f'markerWidth="7" markerHeight="7" orient="auto-start-reverse">'
        f'<path d="M 0 0 L 10 5 L 0 10 z" fill="{c}"/></marker>'
        for c in edge_colors
    )

    min_x = min(v.abs_x for v in vertices) - 20
    min_y = min(v.abs_y for v in vertices) - 20
    max_x = max(v.abs_x + v.w for v in vertices) + 20
    max_y = max(v.abs_y + v.h for v in vertices) + 20
    W, H = max_x - min_x, max_y - min_y

    content_attr = quoteattr(mxfile_text.strip())
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"
width="{W:.0f}" height="{H:.0f}" viewBox="{min_x:.1f} {min_y:.1f} {W:.1f} {H:.1f}"
content={content_attr}>
<defs>
{markers}
</defs>
<rect x="{min_x:.1f}" y="{min_y:.1f}" width="{W:.1f}" height="{H:.1f}" fill="#ffffff"/>
{chr(10).join(parts)}
</svg>
'''
    return svg


def main():
    ap = argparse.ArgumentParser(description="Render .drawio XML to editable .drawio.svg")
    ap.add_argument("input", help="path to .drawio file")
    ap.add_argument("-o", "--output", help="output path (default: <input>.drawio.svg)")
    args = ap.parse_args()

    with open(args.input, encoding="utf-8") as f:
        text = f.read()

    out_path = args.output
    if not out_path:
        base = re.sub(r"\.drawio$", "", args.input)
        out_path = base + ".drawio.svg"

    svg = render(text)
    with open(out_path, "w", encoding="utf-8") as f:
        f.write(svg)
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
