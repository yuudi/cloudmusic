# Current-session logout, password changes and all-session logout must stay within the current account.

POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"Scope_A","password":"original password","display_name":"Account A"}
HTTP 200
[Captures]
a1: jsonpath "$.token"
a_id: jsonpath "$.user.id"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.username" == "Scope_A"
jsonpath "$.user.display_name" == "Account A"
jsonpath "$.user.is_guest" == false

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_a","password":"original password"}
HTTP 200
[Captures]
a2: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{a_id}}"
jsonpath "$.token" != "{{a1}}"

POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"Scope_B","password":"other password","display_name":"Account B"}
HTTP 200
[Captures]
b1: jsonpath "$.token"
b_id: jsonpath "$.user.id"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.username" == "Scope_B"
jsonpath "$.user.display_name" == "Account B"
jsonpath "$.user.is_guest" == false

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{b1}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{b_id}}"
jsonpath "$.user.username" == "Scope_B"
jsonpath "$.user.display_name" == "Account B"

# Only A1 is revoked; A2 and the unrelated account B must still work.
POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer {{a1}}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a1}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a2}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{a_id}}"
jsonpath "$.user.username" == "Scope_A"
jsonpath "$.user.display_name" == "Account A"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{b1}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{b_id}}"
jsonpath "$.user.username" == "Scope_B"
jsonpath "$.user.display_name" == "Account B"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_a","password":"original password"}
HTTP 200
[Captures]
a1: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{a_id}}"
jsonpath "$.token" != "{{a2}}"

# A rejected password change must preserve both sessions and the original password.
POST {{base_url}}/api/v1/auth/password
Authorization: Bearer {{a1}}
Content-Type: application/json
{"old_password":"wrong password","new_password":"updated password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a1}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{a_id}}"
jsonpath "$.user.username" == "Scope_A"
jsonpath "$.user.display_name" == "Account A"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a2}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{a_id}}"
jsonpath "$.user.username" == "Scope_A"
jsonpath "$.user.display_name" == "Account A"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_a","password":"original password"}
HTTP 200
[Captures]
a3: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{a_id}}"

# A successful password change revokes every A session, including the caller, while preserving B.
POST {{base_url}}/api/v1/auth/password
Authorization: Bearer {{a1}}
Content-Type: application/json
{"old_password":"original password","new_password":"updated password"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a1}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a2}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a3}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{b1}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{b_id}}"
jsonpath "$.user.username" == "Scope_B"
jsonpath "$.user.display_name" == "Account B"

# The old password is rejected and the new password creates independent sessions.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_a","password":"original password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_a","password":"updated password"}
HTTP 200
[Captures]
a1: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{a_id}}"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_a","password":"updated password"}
HTTP 200
[Captures]
a2: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{a_id}}"
jsonpath "$.token" != "{{a1}}"

# All-session logout revokes the caller as well as the second A session, without affecting B.
POST {{base_url}}/api/v1/auth/logout-all
Authorization: Bearer {{a1}}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a1}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{a2}}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{b1}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{b_id}}"
jsonpath "$.user.username" == "Scope_B"
jsonpath "$.user.display_name" == "Account B"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"scope_b","password":"other password"}
HTTP 200
[Captures]
b2: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{b_id}}"
