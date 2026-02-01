import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { CONFIG, SUMMARY_TREND_STATS } from '../config/config.js';
import { getRandomProject, getRandomAgent, buildProjectApiPath, buildAuthHeaders } from '../common/utils.js';
import { login } from '../common/auth.js';

const agentPostRequests = new Counter('agent_post_requests');
const agentPostDuration = new Trend('agent_post_duration', true);

/**
 * Generate k6 options for the agent post scenario
 */
export function generateOptions() {
    const cfg = CONFIG.scenarios.agentPost;
    if (!cfg || !cfg.enabled) {
        return { scenarios: {}, thresholds: {} };
    }

    return {
        scenarios: {
            agentPost: {
                executor: cfg.executor,
                rate: cfg.rate,
                timeUnit: cfg.timeUnit,
                duration: cfg.duration,
                preAllocatedVUs: cfg.preAllocatedVUs,
                maxVUs: cfg.maxVUs,
                exec: 'agentPostTest',
                tags: { test: 'agentPost' },
            },
        },
        thresholds: {
            'agent_post_duration': [`p(95)<${cfg.thresholds.responseTime}`],
            'agent_post_requests': ['count>0'],
        },
    };
}

/**
 * POST agent endpoint test
 * @param {Object} data - Data from setup function containing token
 */
export function agentPostTest(data) {
    const token = data.token;
    const project = getRandomProject();
    const agent = getRandomAgent();
    const basePath = buildProjectApiPath(project.namespace, project.project);
    const url = `${CONFIG.baseUrl}${basePath}/agents`;

    const payload = JSON.stringify({
        name: agent.name,
        status: 'success',
        type: 'default',
        load_duration: 10000000,
        version: 2,
    });

    const response = http.post(url, payload, {
        headers: buildAuthHeaders(token),
        tags: { name: 'agent_post', namespace: project.namespace, project: project.project, agent: agent.name },
    });

    agentPostRequests.add(1);
    agentPostDuration.add(response.timings.duration);
    check(response, {
        'agent_post: status is 200': (r) => r.status === 200,
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
    agentPostTest(data);
}
