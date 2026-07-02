package dto

// USSDRequest is the gateway-agnostic inbound payload.
// Africa's Talking sends application/x-www-form-urlencoded; other
// gateways may send JSON. ShouldBind handles both via the tagged fields.
type USSDRequest struct {
	SessionID   string `json:"sessionId"   form:"sessionId"`
	ServiceCode string `json:"serviceCode" form:"serviceCode"`
	PhoneNumber string `json:"phoneNumber" form:"phoneNumber"`
	Text        string `json:"text"        form:"text"`
	NetworkCode string `json:"networkCode" form:"networkCode"`
}

