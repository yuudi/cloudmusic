# Anonymous guest creation, registration and login share a limit of ten requests per minute.

# Request 1 consumes one slot and provides a token for protected requests.
POST {{base_url}}/api/v1/auth/guest
HTTP 200
[Captures]
guest_token: jsonpath "$.token"
guest_id: jsonpath "$.user.id"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000

# Request 2 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Request 3 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"Unknown_User","password":"wrong password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# Request 4 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Request 5 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"Unknown_User","password":"wrong password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# Request 6 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Request 7 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"Unknown_User","password":"wrong password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# Request 8 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Request 9 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"Unknown_User","password":"wrong password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# Request 10 is rejected by validation or authentication, but still consumes one anonymous slot.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Once exhausted, the shared limit applies to this anonymous endpoint too.
POST {{base_url}}/api/v1/auth/guest
HTTP 429
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "rate_limited"
header "Retry-After" == "60"

# Once exhausted, the shared limit applies to this anonymous endpoint too.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"After_Limit","password":"valid password"}
HTTP 429
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "rate_limited"
header "Retry-After" == "60"

# Once exhausted, the shared limit applies to this anonymous endpoint too.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"Unknown_User","password":"wrong password"}
HTTP 429
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "rate_limited"
header "Retry-After" == "60"

# Anonymous rate limiting must not block a valid authenticated session.
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == null
jsonpath "$.user.is_guest" == true

PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"display_name":"Still usable"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == null
jsonpath "$.user.display_name" == "Still usable"
jsonpath "$.user.is_guest" == true
