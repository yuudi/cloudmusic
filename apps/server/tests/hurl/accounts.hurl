# A guest retains their identity and can upgrade without losing the session.
POST {{base_url}}/api/v1/auth/guest
HTTP 200
[Captures]
guest_token: jsonpath "$.token"
guest_id: jsonpath "$.user.id"
[Asserts]
jsonpath "$.user.is_guest" == true
jsonpath "$.token_type" == "Bearer"
header "Cache-Control" == "no-store"

# Guest profile editing counts as activity; checking /me alone does not.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"display_name":"临时昵称"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.display_name" == "临时昵称"
jsonpath "$.user.is_guest" == true

POST {{base_url}}/api/v1/auth/upgrade
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"Test_User","password":"initial password"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
jsonpath "$.user.id" == "{{guest_id}}"
jsonpath "$.user.username" == "Test_User"
jsonpath "$.user.is_guest" == false

# Usernames are unique ignoring case.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"test_user","password":"another password"}
HTTP 409
[Asserts]
jsonpath "$.error" == "username_taken"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"TEST_USER","password":"initial password"}
HTTP 200
[Captures]
second_token: jsonpath "$.token"
[Asserts]
jsonpath "$.user.id" == "{{guest_id}}"

PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"Renamed","display_name":"音乐爱好者"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{second_token}}
HTTP 200
[Asserts]
jsonpath "$.user.username" == "Renamed"
jsonpath "$.user.display_name" == "音乐爱好者"

# Updating one profile field preserves the other; an explicit empty value clears it.
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"username":"Final_Name"}
HTTP 204
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
jsonpath "$.user.username" == "Final_Name"
jsonpath "$.user.display_name" == "音乐爱好者"
PATCH {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"display_name":""}
HTTP 204
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 200
[Asserts]
jsonpath "$.user.username" == "Final_Name"
jsonpath "$.user.display_name" == ""

# Changing the password revokes sessions on both devices.
POST {{base_url}}/api/v1/auth/password
Authorization: Bearer {{guest_token}}
Content-Type: application/json
{"old_password":"initial password","new_password":"new password"}
HTTP 204

GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{guest_token}}
HTTP 401
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{second_token}}
HTTP 401

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"final_name","password":"new password"}
HTTP 200
[Captures]
new_token: jsonpath "$.token"

POST {{base_url}}/api/v1/auth/logout
Authorization: Bearer {{new_token}}
HTTP 204
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{new_token}}
HTTP 401

# Registration, validation, and all-device logout.
POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"Other_User","password":"valid password","display_name":"Other"}
HTTP 200
[Captures]
other_token: jsonpath "$.token"
other_id: jsonpath "$.user.id"
[Asserts]
jsonpath "$.user.is_guest" == false
jsonpath "$.user.id" != "{{guest_id}}"

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"other_user","password":"wrong password"}
HTTP 401

POST {{base_url}}/api/v1/auth/login
Content-Type: application/json
{"username":"OTHER_USER","password":"valid password"}
HTTP 200
[Captures]
other_token_2: jsonpath "$.token"
[Asserts]
jsonpath "$.user.id" == "{{other_id}}"

POST {{base_url}}/api/v1/auth/logout-all
Authorization: Bearer {{other_token}}
HTTP 204
GET {{base_url}}/api/v1/auth/me
Authorization: Bearer {{other_token_2}}
HTTP 401
GET {{base_url}}/api/v1/auth/me
HTTP 401

POST {{base_url}}/api/v1/auth/register
Content-Type: application/json
{"username":"no","password":"valid password"}
HTTP 400
[Asserts]
jsonpath "$.error" == "invalid_request"
