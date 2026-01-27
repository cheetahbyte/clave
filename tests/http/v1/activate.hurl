POST http://localhost:8000/api/v1/activate
{
  "license_key": "DEV-123",
  "hwid": "machine-01"
}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.status" == "activated"
jsonpath "$.token" exists
