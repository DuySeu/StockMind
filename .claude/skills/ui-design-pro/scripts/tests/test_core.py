#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Stdlib-only regression tests for core.py / design_system.py (unittest, not
pytest -- this project ships with zero external dependencies and the tests
shouldn't add one).

Run with:
    python -m unittest discover -s scripts/tests -v
or directly:
    python scripts/tests/test_core.py
"""

import os
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(SCRIPTS_DIR))

from core import BM25, detect_domain, search, search_stack, CSV_CONFIG, AVAILABLE_STACKS
from design_system import generate_design_system, DesignSystemGenerator


class TestTokenizer(unittest.TestCase):
    def test_short_domain_terms_are_kept(self):
        bm25 = BM25()
        tokens = bm25.tokenize("UI and UX design with 3D and AI")
        self.assertIn("ui", tokens)
        self.assertIn("3d", tokens)
        self.assertIn("ai", tokens)

    def test_stopwords_removed(self):
        bm25 = BM25()
        tokens = bm25.tokenize("this is for the team to do")
        for stopword in ("is", "for", "the", "to", "do"):
            self.assertNotIn(stopword, tokens)

    def test_synonym_normalization(self):
        bm25 = BM25()
        self.assertEqual(bm25.tokenize("e-commerce store"), bm25.tokenize("ecommerce store"))
        self.assertEqual(bm25.tokenize("dark-mode toggle"), bm25.tokenize("dark toggle"))


class TestSearchDomains(unittest.TestCase):
    """Known query -> expected top-domain sanity checks (not exact-row pinning,
    since data can grow; these assert the engine still finds *something*
    relevant for each domain's core vocabulary)."""

    def test_ui_is_searchable_in_style_domain(self):
        result = search("ui minimalism", domain="style", max_results=1)
        self.assertGreater(result["count"], 0, "literal 'ui' token must be searchable, not filtered by tokenizer")

    def test_accessibility_query_hits_ux(self):
        result = search("accessibility contrast wcag keyboard", domain="ux", max_results=3)
        self.assertGreater(result["count"], 0)

    def test_zero_result_query_reports_suggestions_not_error(self):
        result = search("zzqqxx totally made up gibberish", domain="ux", max_results=2)
        self.assertEqual(result["count"], 0)
        self.assertIn("suggestions", result)
        self.assertNotIn("error", result)

    def test_every_configured_domain_file_exists_and_is_searchable(self):
        for domain, config in CSV_CONFIG.items():
            with self.subTest(domain=domain):
                result = search("design", domain=domain, max_results=1)
                self.assertNotIn("error", result, f"domain '{domain}' failed: {result.get('error')}")

    def test_every_stack_file_exists_and_is_searchable(self):
        for stack in AVAILABLE_STACKS:
            with self.subTest(stack=stack):
                result = search_stack("performance", stack, max_results=1)
                self.assertNotIn("error", result, f"stack '{stack}' failed: {result.get('error')}")


class TestDomainDetection(unittest.TestCase):
    def test_style_keywords_route_to_style(self):
        self.assertEqual(detect_domain("glassmorphism dark ui"), "style")

    def test_accessibility_keywords_route_to_ux(self):
        self.assertEqual(detect_domain("accessibility contrast wcag"), "ux")

    def test_ambiguous_query_returns_runner_up(self):
        domain, runner_up = detect_domain("font pairing elegant crypto", return_scores=True)
        self.assertIsNotNone(domain)
        # runner_up may be None if the winning domain has no close second --
        # this just verifies the call shape works without raising.

    def test_empty_query_falls_back_to_style(self):
        self.assertEqual(detect_domain("...!!!???"), "style")


class TestGeneratorWritesNothing(unittest.TestCase):
    """The generator used to persist a design-system/<slug>/MASTER.md. That was
    removed -- the token file is the source of truth, and a generated copy of it
    only ever drifted. These tests keep it from coming back."""

    def test_returns_text_and_dict_only(self):
        result = generate_design_system("saas dashboard", "Test Project")
        self.assertEqual(set(result), {"text", "design_system"})
        self.assertTrue(result["text"].strip())

    def test_writes_no_files(self):
        cwd = os.getcwd()
        with tempfile.TemporaryDirectory() as tmp:
            try:
                os.chdir(tmp)
                generate_design_system("saas dashboard", "Test Project")
                self.assertEqual(list(Path(tmp).iterdir()), [], "generator must not write to disk")
            finally:
                os.chdir(cwd)

    def test_defaults_to_markdown_not_ansi_box(self):
        text = generate_design_system("saas dashboard", "Test Project")["text"]
        self.assertNotIn("\033[", text, "default output must not carry ANSI escapes")
        self.assertNotIn("│", text, "default output must not draw a box")
        self.assertLess(
            len(text),
            len(generate_design_system("saas dashboard", "Test Project", "ascii")["text"]),
            "markdown is the cheap default; ascii is the expensive terminal form",
        )


class TestReasoningMatch(unittest.TestCase):
    def test_known_category_matches_exactly(self):
        gen = DesignSystemGenerator()
        rule = gen._find_reasoning_rule("SaaS (General)")
        self.assertTrue(rule, "exact-match category lookup should not fall through to fuzzy matching")

    def test_unknown_category_falls_back_gracefully(self):
        gen = DesignSystemGenerator()
        rule = gen._find_reasoning_rule("Totally Unknown Category XYZ")
        # Should not raise; may return {} which _apply_reasoning handles with defaults.
        self.assertIsInstance(rule, dict)


if __name__ == "__main__":
    unittest.main()
