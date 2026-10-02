package apicomposition

import (
	"encoding/json"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

// errorBody matches the OData V4 error response format (message as a plain
// string, unlike OData V2's nested {"lang","value"} object). SAP's
// documentation only shows success responses; the shape was confirmed by a
// tenant's 403 answer on 2026-09-29 and 2026-10-01:
//
//	{"error": {"code": "2707", "message": "You don't have permission ...",
//	  "@Graph.traceId": "<uuid>", "@Common.numericSeverity": 4}}
//
// The trace ID identifies the request for SAP support. If body does not
// parse this way, the resulting error still carries statusCode so callers
// can act on IsNotFound and similar checks.
type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		TraceID string `json:"@Graph.traceId"`
	} `json:"error"`
}

// ParseError builds an apierror.Error from an API Composition Configuration
// API error response body.
func ParseError(statusCode int, body []byte) *apierror.Error {
	result := &apierror.Error{StatusCode: statusCode}

	var parsed errorBody
	if err := json.Unmarshal(body, &parsed); err != nil {
		return result
	}

	result.Code = parsed.Error.Code
	result.Message = parsed.Error.Message
	result.RequestID = parsed.Error.TraceID
	return result
}
