import http from 'k6/http';
import { CONFIG } from '../config/config.js';

/**
 * Login and get JWT token
 * @returns {string} JWT access token
 */
export function login() {
    const loginUrl = `${CONFIG.baseUrl}/auth/login`;
    const payload = JSON.stringify({
        username: CONFIG.auth.username,
        password: CONFIG.auth.password,
    });

    console.log(`Attempting login to ${loginUrl} with user ${CONFIG.auth.username}`);

    const response = http.post(loginUrl, payload, {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'auth_login' },
    });

    console.log(`Login response status: ${response.status}`);

    if (response.status !== 200) {
        console.error(`Login failed: ${response.status} - ${response.body}`);
        throw new Error(`Login failed with status ${response.status}: ${response.body}`);
    }

    const body = JSON.parse(response.body);
    if (!body.tokens || !body.tokens.accessToken) {
        console.error(`Invalid login response: ${response.body}`);
        throw new Error('Invalid login response: missing tokens.accessToken');
    }

    console.log('Login successful');
    return body.tokens.accessToken;
}
