/*
Usage:
DOC_API_BASE_URL=<URL> k6 run loadtest.js
*/

import http from 'k6/http';
import { check, sleep, group } from 'k6';

// Load Test Configuration
export const options = {
    stages: [
        { duration: '1m', target: 20 }, // ramp up to 20 VUs
        { duration: '3m', target: 20 }, // hold 20 VUs
        { duration: '1m', target: 0 },  // ramp down to 0 VUs
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests must be under 500ms
        http_req_failed: ['rate<0.01'],   // error rate must be under 1%
    },
};

// Configure BASE_URL from environment variable
const BASE_URL = __ENV.DOC_API_BASE_URL || 'http://localhost:8080';

// In-memory file content (simple byte array)
const fileContent = new Uint8Array([0x68, 0x65, 0x6c, 0x6c, 0x6f]).buffer;

export default function () {
    let uploadedDocId = null;

    // 1) Upload Document
    group('Upload Document', function () {
        const data = {
            file: http.file(fileContent, 'test_document.txt', 'text/plain'),
        };

        const res = http.post(`${BASE_URL}/documents`, data, {
            tags: { name: 'UploadDocument' },
        });

        const pass = check(res, {
            'upload status is 201': (r) => r.status === 201,
            'has document id': (r) => r.json() && r.json().id !== undefined,
        });

        if (pass) {
            uploadedDocId = res.json().id;
        }
    });

    // 2) List Documents (always performed)
    group('List Documents', function () {
        const res = http.get(`${BASE_URL}/documents?limit=10&offset=0`, {
            tags: { name: 'ListDocuments' },
        });

        check(res, {
            'list status is 200': (r) => r.status === 200,
            'list is array': (r) => Array.isArray(r.json()),
        });
    });

    // 3) Get & 4) Delete (only if upload succeeded)
    if (uploadedDocId) {
        group('Get Document by ID', function () {
            const res = http.get(`${BASE_URL}/documents/${uploadedDocId}`, {
                tags: { name: 'GetDocument' },
            });

            check(res, {
                'get status is 200': (r) => r.status === 200,
                'correct document id': (r) => r.json() && r.json().id === uploadedDocId,
            });
        });

        group('Delete Document', function () {
            const res = http.del(`${BASE_URL}/documents/${uploadedDocId}`, null, {
                tags: { name: 'DeleteDocument' },
            });

            check(res, {
                'delete status is 204': (r) => r.status === 204,
            });
        });
    }

    // End of iteration
    sleep(1);
}
