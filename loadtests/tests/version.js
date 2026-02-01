import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { CONFIG, SUMMARY_TREND_STATS } from '../config/config.js';
import { getRandomProject, buildProjectApiPath, buildAuthHeaders } from '../common/utils.js';
import { login } from '../common/auth.js';

const versionRequests = new Counter('version_requests');
const versionDuration = new Trend('version_duration', true);

/**
 * Generate k6 options for the version scenario
 */
export function generateOptions() {
    const cfg = CONFIG.scenarios.version;
    if (!cfg || !cfg.enabled) {
        return { scenarios: {}, thresholds: {} };
    }

    return {
        scenarios: {
            version: {
                executor: cfg.executor,
                rate: cfg.rate,
                timeUnit: cfg.timeUnit,
                duration: cfg.duration,
                preAllocatedVUs: cfg.preAllocatedVUs,
                maxVUs: cfg.maxVUs,
                exec: 'versionTest',
                tags: { test: 'version' },
            },
        },
        thresholds: {
            'version_duration': [`p(95)<${cfg.thresholds.responseTime}`],
            'version_requests': ['count>0'],
        },
    };
}

/**
 * Get project version test
 * @param {Object} data - Data from setup function containing token
 */
export function versionTest(data) {
    const token = data.token;
    const project = getRandomProject();
    const url = `${CONFIG.baseUrl}${buildProjectApiPath(project.namespace, project.project)}/version`;
    const response = http.get(url, {
        headers: buildAuthHeaders(token),
        tags: { name: 'get_version', namespace: project.namespace, project: project.project },
    });

    versionRequests.add(1);
    versionDuration.add(response.timings.duration);

    check(response, {
        'version: status is 200': (r) => r.status === 200,
        'version: response is number': (r) => {
            try {
                const body = JSON.parse(r.body);
                return typeof body === 'number';
            } catch {
                return false;
            }
        },
    });
}

// Options for standalone execution
const opts = generateOptions();
export const options = {
    summaryTrendStats: SUMMARY_TREND_STATS,
    scenarios: opts.scenarios,
    thresholds: {
        'http_req_failed': ['rate<0.01'],
        ...opts.thresholds,
    },
};

// Setup for standalone execution - returns data to VUs
export function setup() {
    console.log(`Base URL: ${CONFIG.baseUrl}`);
    const token = login();
    return { token };
}

// Default export for standalone execution
export default function (data) {
    versionTest(data);
}
