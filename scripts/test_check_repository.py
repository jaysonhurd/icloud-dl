"""Exercise repository checks against isolated Git repositories."""

import subprocess
import tempfile
import unittest
from pathlib import Path

from check_repository import check


class RepositoryChecks(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name)
        subprocess.run(["git", "init", "--quiet", str(self.root)], check=True)

    def write(self, name, text):
        path = self.root / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)

    def test_valid_repository_and_fenced_examples(self):
        self.write("skills/example/SKILL.md", "---\nname: example\ndescription: Example workflow.\n---\n")
        self.write("guide.md", '# Guide\n\n## Usage\n\n<a id="custom"></a>\n')
        self.write("README.md", "[Guide](guide.md#usage)\n[Custom](guide.md#custom)\n"
                   "[External](https://example.com/not-checked)\n"
                   "```markdown\n[Example](missing.md)\n```\n")
        self.write("install.sh", "#!/usr/bin/env bash\nexit 0\n")
        self.assertEqual(check(self.root), [])

    def test_missing_link_and_anchor(self):
        self.write("guide.md", "# Guide\n")
        self.write("README.md", "[Missing](missing.md)\n[Anchor](guide.md#missing)\n")
        errors = check(self.root)
        self.assertEqual(len(errors), 2)
        self.assertTrue(any("missing local link" in error for error in errors))
        self.assertTrue(any("missing Markdown anchor" in error for error in errors))

    def test_duplicate_heading_anchors(self):
        self.write("README.md", "# Title\n\n## Usage\n\n## Usage\n\n[Second](#usage-1)\n")
        self.assertEqual(check(self.root), [])

    def test_invalid_skill_metadata(self):
        self.write("skills/example/SKILL.md", "---\nname: Wrong_Name\ndescription: ''\n---\n")
        errors = check(self.root)
        self.assertTrue(any("invalid skill name" in error for error in errors))
        self.assertTrue(any("missing skill description" in error for error in errors))

    def test_missing_skill_frontmatter(self):
        self.write("skills/example/SKILL.md", "# Missing metadata\n")
        self.assertTrue(any("missing YAML frontmatter" in error for error in check(self.root)))

    def test_quoted_skill_metadata(self):
        for quote in ('"', "'"):
            with self.subTest(quote=quote):
                self.write("skills/example/SKILL.md",
                           f"---\nname: {quote}example{quote}\ndescription: {quote}Example workflow.{quote}\n---\n")
                self.assertEqual(check(self.root), [])

    def test_quoted_empty_description_and_mismatched_name(self):
        self.write("skills/example/SKILL.md", '---\nname: "other"\ndescription: "   "\n---\n')
        errors = check(self.root)
        self.assertTrue(any("must match its directory" in error for error in errors))
        self.assertTrue(any("missing skill description" in error for error in errors))

    def test_template_placeholder_links_only_skip_generated_templates(self):
        self.write(".specify/templates/plan-template.md", "[Feature](../[FEATURE]/spec.md)\n")
        self.assertEqual(check(self.root), [])
        self.write("README.md", "[Feature](../[FEATURE]/spec.md)\n")
        self.assertTrue(any("README.md: missing local link" in error for error in check(self.root)))

    def test_concrete_template_links_still_checked(self):
        self.write(".specify/templates/plan-template.md", "[Missing](missing.md)\n")
        self.assertTrue(any("missing local link" in error for error in check(self.root)))

    def test_broken_shell_syntax(self):
        self.write("install.sh", "#!/usr/bin/env bash\nif true; then\n")
        self.assertTrue(any("install.sh" in error for error in check(self.root)))

    def test_ignored_files_are_not_checked(self):
        self.write(".gitignore", ".idea/\n")
        self.write(".idea/notes.md", "[Local notes](missing.md)\n")
        self.assertEqual(check(self.root), [])


    def test_deleted_tracked_files_are_skipped_but_references_fail(self):
        self.write("guide.md", "# Guide\n")
        subprocess.run(["git", "add", "guide.md"], cwd=self.root, check=True)
        (self.root / "guide.md").unlink()
        self.assertEqual(check(self.root), [])
        self.write("README.md", "[Guide](guide.md)\n")
        self.assertTrue(any("missing local link" in e for e in check(self.root)))

    def test_symlinked_document_uses_source_relative_links(self):
        self.write("source/guide.md", "# Guide\n[Details](details.md)\n")
        self.write("source/details.md", "# Details\n")
        (self.root / "alias.md").symlink_to("source/guide.md")
        self.assertEqual(check(self.root), [])

    def test_dangling_symlink_is_reported(self):
        (self.root / "broken.md").symlink_to("absent.md")
        self.assertTrue(any("dangling symlink" in e for e in check(self.root)))


if __name__ == "__main__":
    unittest.main()
