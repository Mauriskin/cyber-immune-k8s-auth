import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 1,
  iterations: 3,
};

const url = 'http://INGRESS_HOST:INGRESS_PORT/login'.replace('INGRESS_HOST', __ENV.INGRESS_HOST).replace('INGRESS_PORT', __ENV.INGRESS_PORT);

const payload = JSON.stringify({
  username: 'admin',
  password: 'secret123',
  mfa: '123456'
});

export default function () {
  const res = http.post(url, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'token issued': (r) => JSON.parse(r.body).token !== undefined,
  });

  sleep(1);
}