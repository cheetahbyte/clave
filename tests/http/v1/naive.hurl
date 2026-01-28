POST http://localhost:8000/api/v1/activate
{
  "license_key": "DEV-123",
  "hwid": "machine-01"
}
HTTP 200

[Asserts]
jsonpath "$.test" == 1
