---
name: conceptual-design
description: Draw conceptual and system architecture diagrams as editable draw.io SVG files (*.drawio.svg). Use this skill whenever the user asks to draw, sketch, or design an architecture, system design, conceptual design, component diagram, deployment diagram, or infrastructure diagram — or mentions draw.io, diagrams.net, .drawio files, or editable SVGs in any way. Also use it when asked to create, update, or fix draw.io editable SVGs, embed diagram data inside an SVG, or keep architecture diagrams editable instead of flat images. The output is a single *.drawio.svg file that renders as a normal SVG image everywhere (GitHub, VS Code, browsers, docs) AND opens in draw.io for further editing.
---

# draw.io Architecture Diagrams (*.drawio.svg)

Produce architecture / conceptual-design diagrams as `*.drawio.svg` files: normal SVG images with the draw.io XML embedded, so the user can keep editing them in draw.io later.

**Core principle: the draw.io XML is the source of truth; the SVG body is a rendered artifact.** A `.drawio.svg` is a normal SVG whose root `<svg>` element carries a `content` attribute holding the XML-escaped mxfile — that attribute is what draw.io reopens and edits. Never strip it.

## Workflow

1. **Design the diagram first, on paper (mentally).** Decide the layers/tiers, which components exist, and which arrows connect them. A good conceptual diagram has 5–15 nodes; if the user's system is bigger, group components into labeled containers. Model responsibilities as nodes and flows/protocols as labeled edges — give it real structure (zones/trust boundaries, layers, a title), not a flat flowchart.

   **Conceptual designs are high-level — stay at the level of responsibilities/stages, never individual functions, classes, or implementation detail.** Unless the user explicitly asks for a detailed/component-level view, structure every conceptual design as three left→right zones, each a labeled dashed container:
   - **1 · USER INPUT** — the things the user provides (arguments, files, requests, events).
   - **2 · PIPELINE / WORKFLOW** — 2–5 high-level stages describing *what happens* to the input (each a responsibility, e.g. "Analyze preview vs safezone", not `parse_verdict()`). Show key external resources (datastores, APIs, models) as supporting dependencies feeding the relevant stage with dashed edges below the pipeline.
   - **3 · OUTPUT** — what the workflow produces (result, response, side effects) plus the error/failure path if relevant.

   Collapse implementation-level steps into stages: e.g. `fetch_image()` + `build_prompt()` + `invoke_model()` become one "Analyze …" stage. If you catch yourself naming functions or listing more than ~5 pipeline boxes, you are too low-level — zoom out.
2. **Plan the layout with real coordinates** (see Layout rules below) — this is the step that decides whether the result looks professional or like spaghetti.
3. **Write the draw.io XML** to `<name>.drawio` (format below).
4. **Render to .drawio.svg** — pick the best available renderer, in this order:

   **a. draw.io's own renderer, if available** (best practice — pixel-faithful, supports every shape):
   - Desktop CLI: `drawio -x -f svg -e --embed-images -o <name>.drawio.svg <name>.drawio`
     (macOS binary: `/Applications/draw.io.app/Contents/MacOS/draw.io`; Linux headless: prefix with `xvfb-run`)
   - VS Code extension `hediet.vscode-drawio`: name the file `*.drawio.svg` and it saves in this dual format natively.
   - app.diagrams.net: File > Export as > SVG with **Include a copy of my diagram** checked.

   Check for the CLI first (`which drawio` or the macOS path). Sandboxed/headless environments usually don't have it — that is what the fallback is for.

   **b. Fallback — bundled renderer (no draw.io needed, stdlib Python only):**
   ```
   python <skill-path>/scripts/render_drawio.py <name>.drawio -o <output>.drawio.svg
   ```
   This draws an approximation of the diagram and embeds the XML correctly. It only supports the shapes listed below — stick to them when you know you'll use the fallback. When delivering a fallback render, tell the user the visual is an approximation and draw.io will re-render it faithfully on first edit (the embedded model is exact either way).

   Save the final `.drawio.svg` in the user's working folder (next to their files) unless they say otherwise. **Always delete the intermediate `.drawio` file after a successful render + validate — the `.drawio.svg` is the only deliverable. Do not ask; remove it automatically.** (The XML source of truth already lives inside the `.drawio.svg` `content` attribute, so nothing is lost — to edit later, extract the XML from `content`, modify, and re-render.)
5. **Validate**: `python <skill-path>/scripts/validate_drawio_svg.py <output>.drawio.svg` — confirms the file is a well-formed SVG with a valid editable mxfile in `content`. Then read the SVG back (it is text) and sanity-check that every node label appears and the viewBox is reasonable. If something is off, fix the XML and re-render — never hand-patch the SVG body.

## draw.io XML format

Minimal valid skeleton — every diagram must have the two bootstrap cells `0` and `1`:

```xml
<mxfile host="app.diagrams.net">
  <diagram id="d1" name="Page-1">
    <mxGraphModel dx="800" dy="600" grid="0" page="0">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>

        <!-- node -->
        <mxCell id="web" value="Web App" style="rounded=1;whiteSpace=wrap;html=1;fillColor=#dae8fc;strokeColor=#6c8ebf;fontSize=13;" vertex="1" parent="1">
          <mxGeometry x="40" y="120" width="160" height="60" as="geometry"/>
        </mxCell>

        <!-- edge -->
        <mxCell id="e1" value="HTTPS" style="edgeStyle=orthogonalEdgeStyle;strokeColor=#6c8ebf;" edge="1" parent="1" source="web" target="api">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
```

Rules: uncompressed XML only · no XML comments in the final file · unique ids · `vertex="1"` xor `edge="1"` · escape special characters in `value` (`&` → `&amp;`, `<` → `&lt;`) · (0,0) is top-left. Full format detail and validation checklist: `reference/drawio-svg-format.md`. A worked example: `examples/architecture.drawio`.

Shapes supported by the fallback renderer (safe everywhere; with the draw.io CLI you may use any draw.io shape or icon library on top of these):

| Purpose | style |
|---|---|
| Component / service | `rounded=1;whiteSpace=wrap;html=1;fillColor=#dae8fc;strokeColor=#6c8ebf;` |
| Database | `shape=cylinder3;whiteSpace=wrap;html=1;fillColor=#d5e8d4;strokeColor=#82b366;` |
| Actor / external system | `rounded=0;whiteSpace=wrap;html=1;fillColor=#ffe6cc;strokeColor=#d79b00;` |
| Decision | `rhombus;whiteSpace=wrap;html=1;fillColor=#fff2cc;strokeColor=#d6b656;` |
| Queue / topic | `rounded=1;whiteSpace=wrap;html=1;fillColor=#e1d5e7;strokeColor=#9673a6;` |
| Start/end, cloud service | `ellipse;whiteSpace=wrap;html=1;fillColor=#f8cecc;strokeColor=#b85450;` |
| Group container / trust boundary | `rounded=1;fillColor=none;strokeColor=#999999;dashed=1;verticalAlign=top;fontStyle=1;` |
| Title / free text | `text;fillColor=none;strokeColor=none;fontSize=18;fontStyle=1;` |

Edge options: `edgeStyle=orthogonalEdgeStyle` for right-angle routing, `dashed=1` for async/optional flows, `endArrow=none` for plain lines, `exitX/exitY/entryX/entryY` (0–1 fractions) to pin which side an arrow leaves/enters. Put short labels (protocol, action) in the edge `value`.

Containers: children of a container cell use `parent="<container-id>"` and coordinates **relative to the container**.

Prefer **plain-text labels** (use `&#10;` for line breaks): draw.io wraps rich HTML labels in `<foreignObject>`, which some viewers (and older GitHub) render poorly, and the fallback renderer strips tags anyway.

## Layout rules (what makes it look good)

- Flow in one direction: left→right (request flow) or top→bottom (layered architecture). Don't mix.
- Standard node size 160×60; databases 120×80. Keep sizes consistent within a tier.
- Gaps: ≥ 80px horizontally, ≥ 60px vertically between nodes. Containers get 30px inner padding + 30px top for the title.
- Align nodes in the same tier on the same x (or y) coordinate — misalignment of a few px looks sloppy.
- Avoid crossing edges: order nodes within a tier so arrows go to neighbors. Use `exitX/entryX` pins when two edges would overlap.
- Keep edge labels short (1–3 words) — long labels overlap neighboring nodes.
- One color per role (use the palette above), not one color per node. A diagram with 8 colors reads worse than one with 3.
- Set `fontColor=#000000` on every shape block (nodes) so labels stay black and legible in all viewers — draw.io otherwise applies a lighter default that is hard to read. Leave the title, subtitle, container/zone headings, and edge labels at their default color.
- Add a bold title text node at the top.
- Coordinates that overlap: before writing XML, list each node with its (x, y, w, h) and check no two rectangles in the same tier intersect.

## Common mistakes / red flags

- **Hand-writing dozens of `<rect>`/`<path>`/`<text>` SVG elements** — you are re-implementing a renderer. Author the XML, then render via the CLI or the bundled script.
- Exporting a plain SVG (no `content` attribute) and calling it "editable" — it opens in draw.io as a flat image. Re-render with `-e` / "Include a copy of my diagram" / the bundled script.
- Saving raw mxfile XML with a `.svg` extension — browsers can't render it.
- Claiming the file is a valid editable `.drawio.svg` **without running the validator**.
- Inventing style keys or using exotic shapes with the fallback renderer — anything outside the table above falls back to a plain rectangle. (With the real draw.io CLI, exotic shapes are fine.)
- Forgetting `parent="1"` on top-level cells or the `vertex="1"`/`edge="1"` flags — cells silently disappear.
- A single flow of boxes labeled "conceptual architecture" — add zones, layers, grouping.
- **Too low-level for a conceptual design** — naming individual functions/classes, or more than ~5 pipeline boxes. Collapse into high-level stages and use the three zones (USER INPUT · PIPELINE/WORKFLOW · OUTPUT) unless the user explicitly asked for a detailed component view.
- If asked to *edit* an existing `.drawio.svg`: extract the XML from the `content` attribute (HTML-unescape it), modify it, and re-render. Never edit the SVG body directly.
