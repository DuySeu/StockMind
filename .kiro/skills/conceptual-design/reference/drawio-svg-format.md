# .drawio.svg format & rendering reference

## What an editable `.drawio.svg` actually is

It is an ordinary SVG file with one addition: the root `<svg>` element carries a
`content` attribute whose value is the **XML-escaped** draw.io diagram
(`<mxfile>…</mxfile>`). Browsers, GitHub, and IDEs ignore `content` and render the
SVG body as an image; draw.io reads `content` and lets you edit the diagram.

```xml
<svg xmlns="http://www.w3.org/2000/svg" width="240" height="120"
     content="&lt;mxfile&gt;&lt;diagram&gt;&lt;mxGraphModel&gt; … &lt;/mxGraphModel&gt;&lt;/diagram&gt;&lt;/mxfile&gt;">
  <!-- draw.io's rendered geometry: rects, paths, text … -->
</svg>
```

Escaping inside the double-quoted attribute: `&` → `&amp;`, `<` → `&lt;`,
`>` → `&gt;`, `"` → `&quot;` (apply `&` first).

**Why you should not hand-write the body:** draw.io's renderer decides fonts,
text wrapping, edge routing, arrowheads, rounded corners, cylinder curves, and
marker definitions. A hand-authored body is only an approximation. As soon as
anyone opens the file in draw.io and saves, draw.io regenerates the body from
`content`, so a hand-drawn preview is throwaway and can silently disagree with
the model it ships next to. Author the XML; let draw.io render.

## Producing the file with draw.io's renderer

### Desktop CLI (draw.io Desktop / drawio-desktop)
```bash
drawio -x -f svg -e --embed-images -o architecture.drawio.svg architecture.drawio
```
- `-x`, `--export` — export mode
- `-f svg`, `--format svg` — output format
- `-e`, `--embed-diagram` — embed an **editable copy** of the diagram (this is what
  makes the SVG reopen in draw.io). Without `-e` you get a flat, uneditable image.
- `--embed-images` — inline any referenced raster images so they render standalone
- `-o`, `--output` — output path (name it `*.drawio.svg`)
- `-r`, `--recursive` — export a whole folder

Binary locations:
- macOS: `/Applications/draw.io.app/Contents/MacOS/draw.io`
- Linux headless (no display): `xvfb-run drawio -x -f svg -e -o out.drawio.svg in.drawio`
- Install: https://github.com/jgraph/drawio-desktop/releases (or `brew install --cask drawio`)

### VS Code — "Draw.io Integration" extension (`hediet.vscode-drawio`)
Create or rename a file to end in `.drawio.svg` (also `.dio.svg`). Opening it shows
the draw.io editor; **saving writes exactly this dual image+XML format**. This is the
most convenient local, no-CLI workflow and keeps the visual and model always in sync.

### app.diagrams.net (browser)
`File > Export as > SVG…` and enable **Include a copy of my diagram** (and *Embed
Images* if the diagram references images). This produces the same `content`-bearing SVG.

## Simplified vs full XML

draw.io accepts either a full `<mxfile><diagram><mxGraphModel>…` or a bare
`<mxGraphModel>…` fragment (it wraps the bare form automatically on open). Use the
full `<mxfile>` form for the `content` attribute of a shipped `.drawio.svg`.

## Key rules for AI-generated XML

1. Always include `<mxCell id="0"/>` and `<mxCell id="1" parent="0"/>`.
2. Uncompressed, plain XML — never compressed/Base64 content.
3. **No XML comments** (`<!-- -->`) anywhere — forbidden; can break parsing.
4. All `id`s unique. Diagram elements use `parent="1"` (or a group/layer id).
5. `vertex="1"` (shapes) and `edge="1"` (connectors) are mutually exclusive.
6. Style = `key=value;` pairs, e.g. `rounded=1;whiteSpace=wrap;html=1;fillColor=#DAE8FC;`.
7. Non-rectangular shapes need a matching `perimeter=` (e.g. `ellipse` →
   `perimeter=ellipsePerimeter`); `shape=cylinder3` handles its own perimeter.
8. Coordinates: (0,0) is top-left; x→right, y→down.
9. HTML inside `value` must be XML-escaped (`&lt; &gt; &amp; &quot;`).
10. Children of a group/container use coordinates **relative to the parent**.

Official reference: https://www.drawio.com/docs/reference/diagram-generation/ and the
canonical shared rules at https://github.com/jgraph/drawio-mcp/blob/main/shared/xml-reference.md

## Conceptual architecture guidance

A conceptual architecture is more than a flow of boxes. Include:
- **Boundaries / zones** (trust boundary, VPC, external vs internal) as dashed containers.
- **Layers or C4 levels** — pick one level and stay consistent: *Context* (system + external
  actors/systems), *Container* (deployable/runnable units + data stores), or *Component*
  (internal parts of one container). Do not mix levels in one diagram.
- **Grouping** related components inside a labeled container.
- **Labeled edges** carrying the protocol/intent (`HTTPS`, `gRPC`, `reads/writes`, `publishes`).
- **A legend** explaining line styles and colors.

## Validation

Structural validation (stdlib only):
```bash
python3 scripts/validate_drawio_svg.py architecture.drawio.svg
```
It confirms: root is `<svg>`, a non-empty `content` attribute exists, the embedded
diagram parses, cells `0`/`1` are present, and reports vertex/edge counts.

For style-string validation and a full checklist, see the draw.io Style Reference:
https://github.com/jgraph/drawio-mcp/blob/main/shared/style-reference.md
