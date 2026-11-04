# Validation failures and conflicts must preserve identities, profiles and credentials.

POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"Taken_Name","password":"owner password","display_name":"Owner profile"}
HTTP 200
[Captures]
owner_token: jsonpath "$.token"
owner_id: jsonpath "$.user.id"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.username" == "Taken_Name"
jsonpath "$.user.display_name" == "Owner profile"
jsonpath "$.user.is_guest" == false

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
jsonpath "$.user.username" == null
jsonpath "$.user.is_guest" == true

PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"display_name":"Guest profile"}
HTTP 204

# A guest cannot gain a username by editing their profile.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"Guest_Rename"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# A guest has no password credentials and remains a usable guest after rejection.
POST {{base_url}}/api/v1/auth/password
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"old_password":"owner password","new_password":"guest password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == null
jsonpath "$.user.display_name" == "Guest profile"
jsonpath "$.user.is_guest" == true

# An upgrade conflict must preserve the guest and the existing account.
POST {{base_url}}/api/v1/auth/upgrade
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"taken_name","password":"guest password"}
HTTP 409
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "username_taken"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == null
jsonpath "$.user.display_name" == "Guest profile"
jsonpath "$.user.is_guest" == true

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{owner_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{owner_id}}"
jsonpath "$.user.username" == "Taken_Name"
jsonpath "$.user.display_name" == "Owner profile"
jsonpath "$.user.is_guest" == false

# The same guest token can retry successfully without changing the user UUID.
POST {{base_url}}/api/v1/auth/upgrade
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"Upgraded_User","password":"guest password"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == "Upgraded_User"
jsonpath "$.user.display_name" == "Guest profile"
jsonpath "$.user.is_guest" == false

# A conflicting rename must not partially save the display name.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"TAKEN_NAME","display_name":"Must not be saved"}
HTTP 409
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "username_taken"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == "Upgraded_User"
jsonpath "$.user.display_name" == "Guest profile"
jsonpath "$.user.is_guest" == false

# Malformed JSON is rejected.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
```
{"display_name":
```
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Wrong JSON field types are rejected.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"display_name":123}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Invalid display names are rejected at the HTTP boundary.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"display_name":"invalid\u0000name"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

# Null fields preserve their previous values.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":null,"display_name":null}
HTTP 204

# A registered account cannot be upgraded again or have its credentials replaced.
POST {{base_url}}/api/v1/auth/upgrade
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"Replacement_User","password":"replacement password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

# An invalid new password must not change credentials or revoke the current session.
POST {{base_url}}/api/v1/auth/password
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"old_password":"guest password","new_password":"short"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == "Upgraded_User"
jsonpath "$.user.display_name" == "Guest profile"
jsonpath "$.user.is_guest" == false

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"upgraded_user","password":"guest password"}
HTTP 200
[Captures]
upgraded_token: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{guest_id}}"

# A failed upgrade must never replace the conflicting account password.
POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"taken_name","password":"guest password"}
HTTP 401
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "unauthorized"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"taken_name","password":"owner password"}
HTTP 200
[Captures]
owner_fresh_token: jsonpath "$.token"
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.token" matches "^[A-Za-z0-9_-]{43}$"
jsonpath "$.token_type" == "Bearer"
jsonpath "$.expires_in" == 2592000
jsonpath "$.user.id" == "{{owner_id}}"

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{owner_fresh_token}}
HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
header "Cache-Control" == "no-store"
jsonpath "$.user.id" == "{{owner_id}}"
jsonpath "$.user.username" == "Taken_Name"
jsonpath "$.user.display_name" == "Owner profile"
jsonpath "$.user.is_guest" == false

# Representative request validation complements the exhaustive Go boundary tests.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"Missing_Password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":123,"password":"valid password"}
HTTP 400
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.error" == "invalid_request"
