import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';
import { CONFIG, SUMMARY_TREND_STATS } from '../config/config.js';
import { getRandomProject, getRandomAgent, buildProjectApiPath, buildAuthHeaders } from '../common/utils.js';
import { login } from '../common/auth.js';

const agentHitRequests = new Counter('agent_hit_requests');
const agentHitDuration = new Trend('agent_hit_duration', true);

/**
 * Generate k6 options for the agent hit scenario
 */
export function generateOptions() {
    const cfg = CONFIG.scenarios.agentHit;
    if (!cfg || !cfg.enabled) {
        return { scenarios: {}, thresholds: {} };
    }

    return {
        scenarios: {
            agentHit: {
                executor: cfg.executor,
                rate: cfg.rate,
                timeUnit: cfg.timeUnit,
                duration: cfg.duration,
                preAllocatedVUs: cfg.preAllocatedVUs,
                maxVUs: cfg.maxVUs,
                exec: 'agentHitTest',
                tags: { test: 'agentHit' },
            },
        },
        thresholds: {
            'agent_hit_duration': [`p(95)<${cfg.thresholds.responseTime}`],
            'agent_hit_requests': ['count>0'],
        },
    };
}

/**
 * PATCH agent hit endpoint test
 * @param {Object} data - Data from setup function containing token
 */
export function agentHitTest(data) {
    const token = data.token;
    const project = getRandomProject();
    const agent = getRandomAgent();
    const basePath = buildProjectApiPath(project.namespace, project.project);
    const url = `${CONFIG.baseUrl}${basePath}/agents/${agent.name}/hit`;

    const response = http.patch(url, null, {
        headers: buildAuthHeaders(token),
        tags: { name: 'agent_hit', namespace: project.namespace, project: project.project, agent: agent.name },
    });

    agentHitRequests.add(1);
    agentHitDuration.add(response.timings.duration);
    check(response, {
        'agent_hit: status is 200': (r) => r.status === 200,
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
    agentHitTest(data);
}
