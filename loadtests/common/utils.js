import { CONFIG } from '../config/config.js';

/**
 * Get a random project from the configured projects list
 * @returns {{namespace: string, project: string}}
 */
export function getRandomProject() {
    const projects = CONFIG.projects;
    if (!projects || projects.length === 0) {
        throw new Error('No projects configured in config.json');
    }
    const index = Math.floor(Math.random() * projects.length);
    return projects[index];
}

/**
 * Get a random agent from the configured agents list
 * @returns {{name: string, hostname: string}}
 */
export function getRandomAgent() {
    const agents = CONFIG.agents;
    if (!agents || agents.length === 0) {
        throw new Error('No agents configured in config.json');
    }
    const index = Math.floor(Math.random() * agents.length);
    return agents[index];
}

/**
 * Build the API base path for a project
 * @param {string} namespace
 * @param {string} project
 * @returns {string}
 */
export function buildProjectApiPath(namespace, project) {
    return `/api/namespace/${namespace}/project/${project}`;
}

/**
 * Build request headers with authorization
 * @param {string} token
 * @returns {Object}
 */
export function buildAuthHeaders(token) {
    return {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
    };
}
