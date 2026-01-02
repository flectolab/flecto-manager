import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import useBaseUrl from '@docusaurus/useBaseUrl';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function HomepageHeader() {
  const { siteConfig } = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/getting-started">
            Get Started
          </Link>
          <Link
            className="button button--outline button--lg"
            style={{ color: '#fff', borderColor: '#fff' }}
            href="https://github.com/flectolab/flecto-manager">
            View on GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

type FeatureItem = {
  title: string;
  description: JSX.Element;
  icon: string;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'HTTP Redirections',
    icon: '🔀',
    description: (
      <>
        Manage all your HTTP redirections from a single place. Support for
        301/302/307/308 redirects with regex patterns.
      </>
    ),
  },
  {
    title: 'Static Pages',
    icon: '📄',
    description: (
      <>
        Serve robots.txt, sitemap.xml, and other static files. Version control
        your content with draft support before publishing.
      </>
    ),
  },
  {
    title: 'Multi-Project',
    icon: '📁',
    description: (
      <>
        Organize your configurations by namespaces and projects. Perfect for
        managing multiple domains or environments.
      </>
    ),
  },
  {
    title: 'Distributed Agents',
    icon: '🌐',
    description: (
      <>
        Deploy lightweight agents close to your users. Agents sync
        configurations automatically and serve requests with minimal latency.
      </>
    ),
  },
  {
    title: 'Role-Based Access',
    icon: '🔐',
    description: (
      <>
        Fine-grained permissions with role-based access control. Support for
        local authentication and OpenID Connect.
      </>
    ),
  },
];

function Feature({ title, icon, description }: FeatureItem) {
  return (
    <div className="feature">
      <div style={{ fontSize: '4rem', marginBottom: '1rem' }}>{icon}</div>
      <Heading as="h3">{title}</Heading>
      <p>{description}</p>
    </div>
  );
}

function HomepageFeatures() {
  return (
    <section className="features">
      <div className="features-container">
        {FeatureList.map((props, idx) => (
          <Feature key={idx} {...props} />
        ))}
      </div>
    </section>
  );
}

type Screenshot = {
  src: string;
  caption: string;
};

function ScreenshotCard({ screenshot }: { screenshot: Screenshot }) {
  const imgSrc = useBaseUrl(screenshot.src);

  return (
    <figure className="screenshot-item">
      <div className="screenshot-image-container">
        <img src={imgSrc} alt={screenshot.caption} />
      </div>
      <figcaption>{screenshot.caption}</figcaption>
    </figure>
  );
}

function HomepageScreenshots() {
  const screenshots: Screenshot[] = [
    { src: 'img/screenshots/project/dashboard.png', caption: 'Dashboard Overview' },
    { src: 'img/screenshots/project/redirects.png', caption: 'Redirect Management' },
    { src: 'img/screenshots/project/pages.png', caption: 'Static Pages Editor' },
    { src: 'img/screenshots/project/agents.png', caption: 'Agent Status' },
  ];

  return (
    <section className="screenshots">
      <div className="container">
        <Heading as="h2">Screenshots</Heading>
        <div className="screenshots-grid">
          {screenshots.map((screenshot, idx) => (
            <ScreenshotCard key={idx} screenshot={screenshot} />
          ))}
        </div>
      </div>
    </section>
  );
}

export default function Home(): JSX.Element {
  const { siteConfig } = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} - HTTP Redirection Manager`}
      description="Centralized management platform for HTTP redirections and static pages">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
        <HomepageScreenshots />
      </main>
    </Layout>
  );
}
