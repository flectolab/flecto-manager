import http from 'k6/http';
import { CONFIG, SUMMARY_TREND_STATS } from './config/config.js';
import { login } from './common/auth.js';

import { generateOptions as versionOptions, versionTest } from './tests/version.js';
import { generateOptions as pagesOptions, pagesTest } from './tests/pages.js';
import { generateOptions as redirectsOptions, redirectsTest } from './tests/redirects.js';
import { generateOptions as agentHitOptions, agentHitTest } from './tests/agent-hit.js';
import { generateOptions as agentPostOptions, agentPostTest } from './tests/agent-post.js';

// Merge all scenario options
const versionOpts = versionOptions();
const pagesOpts = pagesOptions();
const redirectsOpts = redirectsOptions();
const agentHitOpts = agentHitOptions();
const agentPostOpts = agentPostOptions();

export const options = {
    summaryTrendStats: SUMMARY_TREND_STATS,
    scenarios: {
        ...versionOpts.scenarios,
        ...pagesOpts.scenarios,
        ...redirectsOpts.scenarios,
        ...agentHitOpts.scenarios,
        ...agentPostOpts.scenarios,
    },
    thresholds: {
        'http_req_failed': ['rate<0.01'],
        ...versionOpts.thresholds,
        ...pagesOpts.thresholds,
        ...redirectsOpts.thresholds,
        ...agentHitOpts.thresholds,
        ...agentPostOpts.thresholds,
    },
};

export function setup() {
    console.log('='.repeat(60));
    console.log('Flecto Manager Load Test');
    console.log('='.repeat(60));
    console.log(`Base URL: ${CONFIG.baseUrl}`);
    console.log(`Projects: ${CONFIG.projects.length}`);
    CONFIG.projects.forEach((p, i) => {
        console.log(`  ${i + 1}. ${p.namespace}/${p.project}`);
    });
    console.log(`Agents: ${CONFIG.agents.length}`);
    CONFIG.agents.forEach((a, i) => {
        console.log(`  ${i + 1}. ${a.name} (${a.hostname})`);
    });
    console.log('='.repeat(60));

    // Check connectivity
    const healthUrl = `${CONFIG.baseUrl}/health/ping`;
    const healthResponse = http.get(healthUrl, { timeout: '5s' });
    if (healthResponse.status !== 204) {
        console.warn(`WARNING: Health check failed (status ${healthResponse.status})`);
        console.warn(`Make sure the server is running at ${CONFIG.baseUrl}`);
    } else {
        console.log('Health check: OK');
    }

    // Authenticate and return token to VUs
    const token = login();
    return { token };
}

export function teardown() {
    console.log('='.repeat(60));
    console.log('Load test completed');
    console.log('='.repeat(60));
}

// Export test functions for scenarios - they receive data from setup
export { versionTest, pagesTest, redirectsTest, agentHitTest, agentPostTest };
