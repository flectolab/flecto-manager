import mediumZoom from 'medium-zoom';
import type { ClientModule } from '@docusaurus/types';

const module: ClientModule = {
  onRouteDidUpdate() {
    setTimeout(() => {
      mediumZoom('.markdown img, .screenshot-item img', {
        margin: 24,
        background: 'rgba(0, 0, 0, 0.8)',
      });
    }, 100);
  },
};

export default module;
