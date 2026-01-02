import { themes as prismThemes } from 'prism-react-renderer';
import type { Config } from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Flecto Manager',
  tagline: 'Centralized management platform for HTTP redirections and static pages',
  favicon: 'img/favicon.ico',

  url: 'https://flectolab.github.io',
  baseUrl: '/flecto-manager/',

  organizationName: 'flectolab',
  projectName: 'flecto-manager',

  onBrokenLinks: 'throw',
  onBrokenAnchors: 'warn',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
      onBrokenMarkdownImages: 'warn',
    },
  },

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
          editUrl: 'https://github.com/flectolab/flecto-manager/tree/main/docs/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  clientModules: [
    './src/theme/zoom.ts',
  ],

  themeConfig: {
    image: 'img/social-card.png',
    navbar: {
      title: 'Flecto Manager',
      logo: {
        alt: 'Flecto Manager Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'docSidebar',
          sidebarId: 'docsSidebar',
          position: 'left',
          label: 'Documentation',
        },
        {
          type: 'docsVersionDropdown',
          position: 'right',
          dropdownActiveClassDisabled: true,
        },
        {
          href: 'https://github.com/flectolab/flecto-manager',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            {
              label: 'Getting Started',
              to: '/docs/getting-started',
            },
            {
              label: 'Configuration',
              to: '/docs/configuration',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'GitHub Issues',
              href: 'https://github.com/flectolab/flecto-manager/issues',
            },
          ],
        },
        {
          title: 'More',
          items: [
            {
              label: 'GitHub',
              href: 'https://github.com/flectolab/flecto-manager',
            },
            {
              label: 'Releases',
              href: 'https://github.com/flectolab/flecto-manager/releases',
            },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Flectolab. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'yaml', 'go'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
