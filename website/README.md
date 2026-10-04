# Queen documentation site

The Docusaurus site lives in this directory and is configured for publication from the Queen repository at `https://dmedovich.github.io/queen/`.

```bash
npm ci --prefix website
npm run build --prefix website
```

The build generates versioned release pages from the root `CHANGELOG.md`. Before publishing a GitHub release, add a `## vX.Y.Z` heading with its notes to the changelog and include that change in the release tag. The documentation workflow builds pull requests and deploys the site after changes on `main` or a GitHub release is published. A release event fails its documentation build if the tag has no matching changelog section.

The repository's GitHub Pages source must be set to **GitHub Actions** for deployment. The `queen-docs` checkout was imported into this directory; edit the site here going forward.
