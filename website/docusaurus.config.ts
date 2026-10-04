import type {Config} from '@docusaurus/types';
import type {Options as PresetOptions, ThemeConfig} from '@docusaurus/preset-classic';

const siteUrl = process.env.DOCUSAURUS_URL ?? 'https://dmedovich.github.io';
const baseUrl = process.env.DOCUSAURUS_BASE_URL ?? '/queen/';

const config: Config = {
  title: 'Queen',
  tagline: 'Go migrations embedded in your app',
  favicon: 'favicon.ico',

  url: siteUrl,
  baseUrl,
  organizationName: 'dmedovich',
  projectName: 'queen',

  onBrokenLinks: 'throw',
  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'throw',
    },
  },
  trailingSlash: false,

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          routeBasePath: 'docs',
          editUrl: 'https://github.com/dmedovich/queen/edit/main/website/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies PresetOptions,
    ],
  ],

  themeConfig: {
    image: 'img/queen_logo.png',
    navbar: {
      title: 'Queen',
      logo: {
        alt: 'Queen logo',
        src: 'img/queen_logo.png',
      },
      items: [
        {to: '/docs/quick-start', label: 'Quick Start', position: 'left'},
        {to: '/docs/library', label: 'Library', position: 'left'},
        {to: '/docs/examples', label: 'Examples', position: 'left'},
        {to: '/docs/support-matrix', label: 'Support Matrix', position: 'left'},
        {to: '/docs/releases', label: 'Releases', position: 'left'},
        {
          href: 'https://github.com/dmedovich/queen',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Learn',
          items: [
            {label: 'Quick Start', to: '/docs/quick-start'},
            {label: 'Library API', to: '/docs/library'},
            {label: 'Compatibility', to: '/docs/compatibility'},
            {label: 'CLI Reference', to: '/docs/cli-reference'},
            {label: 'Migration Format', to: '/docs/migrations'},
          ],
        },
        {
          title: 'Operations',
          items: [
            {label: 'Support Matrix', to: '/docs/support-matrix'},
            {label: 'Database Drivers', to: '/docs/drivers'},
            {label: 'Known Limitations', to: '/docs/known-limitations'},
            {label: 'Troubleshooting', to: '/docs/troubleshooting'},
          ],
        },
        {
          title: 'Project',
          items: [
            {label: 'GitHub', href: 'https://github.com/dmedovich/queen'},
            {label: 'Go Reference', href: 'https://pkg.go.dev/github.com/dmedovich/queen'},
            {label: 'Release Notes', to: '/docs/releases'},
            {label: 'Release Checklist', to: '/docs/release-checklist'},
          ],
        },
      ],
      copyright: `Copyright ${new Date().getFullYear()} Queen contributors. Apache-2.0.`,
    },
    prism: {
      additionalLanguages: ['go', 'bash', 'sql', 'yaml'],
    },
  } satisfies ThemeConfig,
};

export default config;
