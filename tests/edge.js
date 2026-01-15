import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 1,
  iterations: 15,
};

const url = 'http://INGRESS_HOST:INGRESS_PORT/login'.replace('INGRESS_HOST', __ENV.INGRESS_HOST).replace('INGRESS_PORT', __ENV.INGRESS_PORT);

const validPayload = JSON.stringify({username: 'admin', password: 'secret123', mfa: '123456'});

let token = '';

export default function () {
  if (__ITER < 12) { // 11 успешных + 1 с rate limit
    let res = http.post(url, validPayload, {headers: {'Content-Type': 'application/json'}});
    if (__ITER === 0) {
      token = JSON.parse(res.body).token;
    }
    check(res, { 'request allowed': (r) => r.status === 200 });
  } else {
    // Rate limit test (Ingress + Auth)
    let res = http.post(url, validPayload, {headers: {'Content-Type': 'application/json'}});
    check(res, { 'rate limit triggered': (r) => r.status === 429 || r.status === 503 });
  }

  // Token expiration test (после ~70 секунд)
  if (__ITER === 14) {
    sleep(70);
    let validateRes = http.get('http://token-service-svc.domain3-tcb.svc.cluster.local:8080/validate', {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    // Этот запрос внутри кластера — для проверки expiration
    // В реальности запустить из pod
  }

  sleep(1);
}