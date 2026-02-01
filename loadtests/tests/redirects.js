import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { CONFIG, SUMMARY_TREND_STATS } from '../config/config.js';
import { getRandomProject, buildProjectApiPath, buildAuthHeaders } from '../common/utils.js';
import { login } from '../common/auth.js';

const redirectsRequests = new Counter('redirects_requests');
const redirectsDuration = new Trend('redirects_duration', true);
const redirectsTotalDuration = new Trend('redirects_total_duration', true);
const redirectsItemsTotal = new Counter('redirects_items_total_fetched');

/**
 * Generate k6 options for the redirects scenario
 */
export function generateOptions() {
    const cfg = CONFIG.scenarios.redirects;
    if (!cfg || !cfg.enabled) {
        return { scenarios: {}, thresholds: {} };
    }

    return {
        scenarios: {
            redirects: {
                executor: cfg.executor,
                rate: cfg.rate,
                timeUnit: cfg.timeUnit,
                duration: cfg.duration,
                preAllocatedVUs: cfg.preAllocatedVUs,
                maxVUs: cfg.maxVUs,
                exec: 'redirectsTest',
                tags: { test: 'redirects' },
            },
        },
        thresholds: {
            'redirects_duration': [`p(95)<${cfg.thresholds.responseTime}`],
            'redirects_requests': ['count>0'],
        },
    };
}

/**
 * Get all redirects with pagination (limit=50)
 * @param {Object} data - Data from setup function containing token
 */
export function redirectsTest(data) {
    const token = data.token;
    const project = getRandomProject();
    const basePath = buildProjectApiPath(project.namespace, project.project);
    const limit = CONFIG.scenarios.redirects.limit || 50;
    let offset = 0;
    let hasMore = true;
    const startTime = Date.now();

    while (hasMore) {
        const url = `${CONFIG.baseUrl}${basePath}/redirects?limit=${limit}&offset=${offset}`;

        const response = http.get(url, {
            headers: buildAuthHeaders(token),
            tags: { name: 'get_redirects', namespace: project.namespace, project: project.project },
        });

        redirectsRequests.add(1);
        redirectsDuration.add(response.timings.duration);

        const isValid = check(response, {
            'redirects: status is 200': (r) => r.status === 200,
            'redirects: has items array': (r) => {
                try {
                    const body = JSON.parse(r.body);
                    return Array.isArray(body.Items);
                } catch {
                    return false;
                }
            },
        });

        if (!isValid || response.status !== 200) {
            break;
        }

        try {
            const body = JSON.parse(response.body);
            redirectsItemsTotal.add(body.Items.length);
            hasMore = body.Offset + body.Items.length < body.Total;
            offset += limit;
        } catch {
            hasMore = false;
        }
    }

    redirectsTotalDuration.add(Date.now() - startTime);
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
    redirectsTest(data);
}
