# Missing, forged and revoked tokens must be rejected by every protected operation.

POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"Guard_Member","password":"member password","display_name":"Original profile"}
HTTP 200
[Captures]
member_token: jsonpath "$.token"
member_id: jsonpath "$.user.id"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.username" == "Guard_Member"
jsonpath "$.user.display_name" == "Original profile"
jsonpath "$.user.is_guest" == false

POST {{base_url}}/api/v1/auth/guest
HTTP 200
[Captures]
revoked_token: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000

POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer {{revoked_token}}
HTTP 204

# Missing authorization.

GET {{base_url}}/api/v1/auth/me
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

PATCH {{base_url}}/api/v1/auth/me
Content-Type: application/json
{"username":"Hijacked","display_name":"Changed by unauthorized request"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/upgrade
Content-Type: application/json
{"username":"Hijacked","password":"hijacked password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/password
Content-Type: application/json
{"old_password":"member password","new_password":"hijacked password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/logout
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/logout-all
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# A forged token with the correct length.

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
Content-Type: application/json
{"username":"Hijacked","display_name":"Changed by unauthorized request"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/upgrade
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
Content-Type: application/json
{"username":"Hijacked","password":"hijacked password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/password
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
Content-Type: application/json
{"old_password":"member password","new_password":"hijacked password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/logout-all
Authorization: Bearer AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# A previously valid, now revoked token.

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{revoked_token}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{revoked_token}}
Content-Type: application/json
{"username":"Hijacked","display_name":"Changed by unauthorized request"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/upgrade
Authorization: Bearer {{revoked_token}}
Content-Type: application/json
{"username":"Hijacked","password":"hijacked password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/password
Authorization: Bearer {{revoked_token}}
Content-Type: application/json
{"old_password":"member password","new_password":"hijacked password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer {{revoked_token}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/logout-all
Authorization: Bearer {{revoked_token}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# A valid token under the wrong authentication scheme is still rejected.
GET {{base_url}}/api/v1/auth/me
Authorization: Basic {{member_token}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# Rejected write operations must not revoke the real member session or alter their profile.
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{member_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{member_id}}"
jsonpath "$.user.username" == "Guard_Member"
jsonpath "$.user.display_name" == "Original profile"
jsonpath "$.user.is_guest" == false

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"guard_member","password":"member password"}
HTTP 200
[Captures]
fresh_token: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{member_id}}"
