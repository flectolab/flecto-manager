import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { CONFIG, SUMMARY_TREND_STATS } from '../config/config.js';
import { getRandomProject, buildProjectApiPath, buildAuthHeaders } from '../common/utils.js';
import { login } from '../common/auth.js';

const pagesRequests = new Counter('pages_requests');
const pagesDuration = new Trend('pages_duration', true);
const pagesTotalDuration = new Trend('pages_total_duration', true);
const pagesItemsTotal = new Counter('pages_items_total_fetched');

/**
 * Generate k6 options for the pages scenario
 */
export function generateOptions() {
    const cfg = CONFIG.scenarios.pages;
    if (!cfg || !cfg.enabled) {
        return { scenarios: {}, thresholds: {} };
    }

    return {
        scenarios: {
            pages: {
                executor: cfg.executor,
                rate: cfg.rate,
                timeUnit: cfg.timeUnit,
                duration: cfg.duration,
                preAllocatedVUs: cfg.preAllocatedVUs,
                maxVUs: cfg.maxVUs,
                exec: 'pagesTest',
                tags: { test: 'pages' },
            },
        },
        thresholds: {
            'pages_duration': [`p(95)<${cfg.thresholds.responseTime}`],
            'pages_requests': ['count>0'],
        },
    };
}

/**
 * Get all pages with pagination (limit=50)
 * @param {Object} data - Data from setup function containing token
 */
export function pagesTest(data) {
    const token = data.token;
    const project = getRandomProject();
    const basePath = buildProjectApiPath(project.namespace, project.project);
    const limit = CONFIG.scenarios.pages.limit || 50;
    let offset = 0;
    let hasMore = true;
    const startTime = Date.now();

    while (hasMore) {

        const url = `${CONFIG.baseUrl}${basePath}/pages?limit=${limit}&offset=${offset}`;

        const response = http.get(url, {
            headers: buildAuthHeaders(token),
            tags: { name: 'get_pages', namespace: project.namespace, project: project.project },
        });

        pagesRequests.add(1);
        pagesDuration.add(response.timings.duration);

        const isValid = check(response, {
            'pages: status is 200': (r) => r.status === 200,
            'pages: has items array': (r) => {
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
            pagesItemsTotal.add(body.Items.length);
            hasMore = body.Offset + body.Items.length < body.Total;
            offset += limit;
        } catch {
            hasMore = false;
        }
    }

    pagesTotalDuration.add(Date.now() - startTime);
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
    pagesTest(data);
}
