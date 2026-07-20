#!/usr/bin/env python3
"""Validate that a file is a well-formed, editable .drawio.svg.

An editable .drawio.svg is a normal SVG whose root <svg> element carries a
`content` attribute holding the XML-escaped draw.io diagram (mxfile /
mxGraphModel). draw.io re-opens and edits the diagram from that attribute.

Checks performed:
  1. File parses as well-formed XML and the root element is <svg>.
  2. The root <svg> has a non-empty `content` attribute.
  3. The unescaped content parses as XML and contains an mxGraphModel/root.
  4. The mandatory structural cells id="0" and id="1" are present.
  5. Reports vertex/edge counts so you can sanity-check the model.

Usage:  python3 validate_drawio_svg.py path/to/diagram.drawio.svg
Exit code 0 = valid editable .drawio.svg, 1 = invalid.  stdlib only.
"""
import sys
import xml.etree.ElementTree as ET


def strip_ns(tag: str) -> str:
    return tag.rsplit("}", 1)[-1]


def validate(path: str) -> bool:
    ok = True

    def fail(msg):
        nonlocal ok
        ok = False
        print(f"  FAIL: {msg}")

    def ok_(msg):
        print(f"  ok  : {msg}")

    print(f"Validating {path}")

    # 1. outer SVG is well-formed
    try:
        tree = ET.parse(path)
    except ET.ParseError as e:
        fail(f"outer file is not well-formed XML: {e}")
        return False
    root = tree.getroot()
    if strip_ns(root.tag) != "svg":
        fail(f"root element is <{strip_ns(root.tag)}>, expected <svg> "
             f"(is this mxfile XML saved with a .svg name?)")
        return False
    ok_("root element is <svg> (renders as an image)")

    # 2. content attribute present
    content = root.get("content")
    if not content:
        fail("root <svg> has no `content` attribute — NOT editable in draw.io. "
             "Re-export with 'Include a copy of my diagram' / CLI flag -e.")
        return False
    ok_("root <svg> carries a `content` attribute (editable in draw.io)")

    # 3. embedded diagram parses (ET already unescaped the attribute value)
    try:
        inner = ET.fromstring(content)
    except ET.ParseError as e:
        fail(f"embedded diagram XML is not well-formed: {e}")
        return False
    inner_root = strip_ns(inner.tag)
    if inner_root not in ("mxfile", "mxGraphModel"):
        fail(f"embedded root is <{inner_root}>, expected <mxfile>/<mxGraphModel>")
        return False
    ok_(f"embedded diagram parses (root <{inner_root}>)")

    cells = [e for e in inner.iter() if strip_ns(e.tag) == "mxCell"]
    ids = {c.get("id") for c in cells}
    for req in ("0", "1"):
        if req not in ids:
            fail(f"mandatory structural cell id=\"{req}\" is missing")
    if {"0", "1"} <= ids:
        ok_('mandatory structural cells id="0" and id="1" present')

    verts = sum(1 for c in cells if c.get("vertex") == "1")
    edges = sum(1 for c in cells if c.get("edge") == "1")
    ok_(f"model contains {verts} vertices and {edges} edges")
    if verts == 0:
        fail("no vertices — the embedded model is empty")

    if "<!--" in content:
        fail("embedded XML contains an XML comment (forbidden by draw.io AI rules)")

    return ok


def main():
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    valid = validate(sys.argv[1])
    print("RESULT:", "VALID editable .drawio.svg" if valid else "INVALID")
    sys.exit(0 if valid else 1)


if __name__ == "__main__":
    main()
