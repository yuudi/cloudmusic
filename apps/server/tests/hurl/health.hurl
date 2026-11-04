# Health responses are verified over the real HTTP server and reachable dependencies.

GET {{base_url}}/api/v1/health/live
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.status" == "ok"

GET {{base_url}}/api/v1/health/ready
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.status" == "ok"
jsonpath "$.checks.postgres" == "ok"
jsonpath "$.checks.redis" == "ok"

# Unknown API routes return JSON errors instead of an HTML fallback.
GET {{base_url}}/api/v1/not-a-route
HTTP 404
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "not_found"

POST {{base_url}}/api/v1/health/live
HTTP 405
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "method_not_allowed"
