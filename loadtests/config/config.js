const config = JSON.parse(open('./config.json'));

config.baseUrl = __ENV.BASE_URL || config.baseUrl;

if (__ENV.AUTH_USERNAME) {
    config.auth.username = __ENV.AUTH_USERNAME;
}
if (__ENV.AUTH_PASSWORD) {
    config.auth.password = __ENV.AUTH_PASSWORD;
}

export const SUMMARY_TREND_STATS = ['min', 'avg', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'count'];
export const CONFIG = config;
