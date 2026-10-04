import {mkdirSync, readFileSync, writeFileSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {fileURLToPath} from 'node:url';

const siteDir = dirname(dirname(fileURLToPath(import.meta.url)));
const repoDir = dirname(siteDir);
const changelog = readFileSync(join(repoDir, 'CHANGELOG.md'), 'utf8');
const headings = [...changelog.matchAll(/^## (v\d+\.\d+\.\d+)(?:[^\n]*)$/gm)];
if (headings.length === 0) {
  throw new Error('CHANGELOG.md has no version headings');
}

const releaseTag = process.env.RELEASE_TAG;
if (releaseTag && !headings.some((heading) => heading[1] === releaseTag)) {
  throw new Error(`CHANGELOG.md has no notes for ${releaseTag}`);
}

const releaseDir = join(siteDir, 'docs', 'releases');
mkdirSync(releaseDir, {recursive: true});
writeFileSync(join(releaseDir, 'index.md'),
  `---\ntitle: Releases\nslug: /releases\nsidebar_position: 1\n---\n\n# Release Notes\n\n${headings.map((heading) => `- [${heading[1]}](./${heading[1]}.md)`).join('\n')}\n`);

for (let i = 0; i < headings.length; i++) {
  const heading = headings[i];
  const version = heading[1];
  const end = i + 1 < headings.length ? headings[i + 1].index : changelog.length;
  const notes = changelog.slice(heading.index + heading[0].length, end).trim();
  if (!notes) {
    throw new Error(`CHANGELOG.md has no notes for ${version}`);
  }
  const linkedNotes = notes.replace(/\]\((?!https?:|\/|#)([^)]+)\)/g,
    (_, path) => `](https://github.com/dmedovich/queen/blob/${version}/${path})`);
  writeFileSync(join(releaseDir, `${version}.md`),
    `---\ntitle: ${version}\nslug: /releases/${version}\n---\n\n# ${version}\n\n${linkedNotes}\n`);
}

console.log(`Generated ${headings.length} release note page(s) from CHANGELOG.md`);
