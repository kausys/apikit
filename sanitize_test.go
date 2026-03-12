package apikit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize_NilAndNonStruct(t *testing.T) {
	assert.Nil(t, Sanitize(nil))
	assert.Equal(t, 42, Sanitize(42))
	assert.Equal(t, "hello", Sanitize("hello"))
	assert.Equal(t, true, Sanitize(true))
}

func TestSanitize_SimpleStruct(t *testing.T) {
	type Payload struct {
		Name     string `json:"name"`
		Password string `json:"password" log:"sensitive"`
		Internal string `json:"-" log:"-"`
	}

	result := Sanitize(Payload{
		Name:     "alice",
		Password: "secret123",
		Internal: "hidden",
	})

	m := result.(map[string]any)
	assert.Equal(t, "alice", m["name"])
	assert.Equal(t, "[REDACTED]", m["password"])
	assert.NotContains(t, m, "Internal")
}

func TestSanitize_OmittedField(t *testing.T) {
	type Payload struct {
		Visible string `json:"visible"`
		Hidden  string `json:"hidden" log:"-"`
	}

	m := Sanitize(Payload{Visible: "yes", Hidden: "no"}).(map[string]any)
	assert.Equal(t, "yes", m["visible"])
	assert.NotContains(t, m, "hidden")
}

func TestSanitize_NoJsonTag(t *testing.T) {
	type Payload struct {
		FieldName string
	}

	m := Sanitize(Payload{FieldName: "value"}).(map[string]any)
	assert.Equal(t, "value", m["FieldName"])
}

func TestSanitize_NestedStruct(t *testing.T) {
	type Address struct {
		City string `json:"city"`
		SSN  string `json:"ssn" log:"sensitive"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	m := Sanitize(Person{
		Name:    "bob",
		Address: Address{City: "NYC", SSN: "123-45-6789"},
	}).(map[string]any)

	assert.Equal(t, "bob", m["name"])
	addr := m["address"].(map[string]any)
	assert.Equal(t, "NYC", addr["city"])
	assert.Equal(t, "[REDACTED]", addr["ssn"])
}

func TestSanitize_Pointer(t *testing.T) {
	type Payload struct {
		Token string `json:"token" log:"sensitive"`
	}
	p := &Payload{Token: "abc"}
	m := Sanitize(p).(map[string]any)
	assert.Equal(t, "[REDACTED]", m["token"])
}

func TestSanitize_NilPointer(t *testing.T) {
	type Payload struct {
		Token string `json:"token"`
	}
	var p *Payload
	assert.Nil(t, Sanitize(p))
}

func TestSanitize_SliceOfStructs(t *testing.T) {
	type Item struct {
		ID     int    `json:"id"`
		Secret string `json:"secret" log:"sensitive"`
	}

	items := []Item{
		{ID: 1, Secret: "a"},
		{ID: 2, Secret: "b"},
	}

	result := Sanitize(items).([]any)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].(map[string]any)["id"])
	assert.Equal(t, "[REDACTED]", result[0].(map[string]any)["secret"])
}

func TestSanitize_EmbeddedStruct(t *testing.T) {
	type Base struct {
		ID int `json:"id"`
	}
	type Extended struct {
		Base
		Name  string `json:"name"`
		Token string `json:"token" log:"sensitive"`
	}

	m := Sanitize(Extended{
		Base:  Base{ID: 1},
		Name:  "test",
		Token: "secret",
	}).(map[string]any)

	assert.Equal(t, 1, m["id"])
	assert.Equal(t, "test", m["name"])
	assert.Equal(t, "[REDACTED]", m["token"])
}

func TestSanitize_UnexportedFields(t *testing.T) {
	type Payload struct {
		Public  string `json:"public"`
		private string //nolint:unused
	}

	m := Sanitize(Payload{Public: "yes"}).(map[string]any)
	assert.Equal(t, "yes", m["public"])
	assert.NotContains(t, m, "private")
}

func TestSanitize_EmptyStruct(t *testing.T) {
	type Empty struct{}
	m := Sanitize(Empty{}).(map[string]any)
	assert.Empty(t, m)
}

func TestSanitize_JsonOmitempty(t *testing.T) {
	type Payload struct {
		Name string `json:"name,omitempty"`
	}

	m := Sanitize(Payload{Name: "test"}).(map[string]any)
	assert.Equal(t, "test", m["name"])
}

// Benchmark types that mirror real blockwyre payloads

type benchDocumentID struct {
	IDNumber         string `json:"idNumber" log:"sensitive"`
	IDDocumentType   string `json:"idDocumentType"`
	IDIssueDate      string `json:"idIssueDate"`
	IDExpiryDate     string `json:"idExpiryDate"`
	IDCountryOfIssue string `json:"idCountryOfIssue"`
	Nationality      string `json:"nationality"`
}

type benchAddress struct {
	Address1   string `json:"address1"`
	Address2   string `json:"address2"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
	Country    string `json:"country"`
}

type benchKYCPayload struct {
	AccountID   string          `json:"accountId"`
	DateOfBirth string          `json:"dateOfBirth" log:"sensitive"`
	Phone       string          `json:"phone" log:"sensitive"`
	Address     benchAddress    `json:"address"`
	DocumentID  benchDocumentID `json:"documentId"`
}

type benchBankingPayload struct {
	RoutingNumber       string       `json:"routingNumber" log:"sensitive"`
	AccountNumber       string       `json:"accountNumber" log:"sensitive"`
	BankName            string       `json:"bankName"`
	BankAccountType     string       `json:"bankAccountType"`
	RdfiNumberQualifier string       `json:"rdfiNumberQualifier"`
	GatewayRoutingNumber string      `json:"gatewayRoutingNumber" log:"sensitive"`
	Address             benchAddress `json:"address"`
}

type benchLoginPayload struct {
	Origin    string `json:"-"`
	UserAgent string `json:"-"`
	Body      struct {
		Token string `json:"token" log:"sensitive"`
	}
}

// BenchmarkSanitize_NoTags measures overhead on structs with no log tags
func BenchmarkSanitize_NoTags(b *testing.B) {
	payload := benchAddress{
		Address1: "123 Main St", Address2: "Apt 4",
		City: "New York", State: "NY", PostalCode: "10001", Country: "US",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sanitize(payload)
	}
}

// BenchmarkSanitize_KYCPayload measures a realistic KYC payload with nested structs
func BenchmarkSanitize_KYCPayload(b *testing.B) {
	payload := benchKYCPayload{
		AccountID:   "550e8400-e29b-41d4-a716-446655440000",
		DateOfBirth: "1990-05-15",
		Phone:       "+1234567890",
		Address: benchAddress{
			Address1: "123 Main St", City: "New York", State: "NY",
			PostalCode: "10001", Country: "US",
		},
		DocumentID: benchDocumentID{
			IDNumber: "ABC123456", IDDocumentType: "passport",
			IDIssueDate: "2020-01-01", IDExpiryDate: "2030-01-01",
			IDCountryOfIssue: "US", Nationality: "US",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sanitize(payload)
	}
}

// BenchmarkSanitize_BankingPayload measures a banking payload
func BenchmarkSanitize_BankingPayload(b *testing.B) {
	payload := benchBankingPayload{
		RoutingNumber: "021000021", AccountNumber: "123456789",
		BankName: "BlockWyre Bank", BankAccountType: "checking",
		Address: benchAddress{
			Address1: "456 Wall St", City: "New York", State: "NY",
			PostalCode: "10005", Country: "US",
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sanitize(payload)
	}
}

// BenchmarkSanitize_LoginPayload measures a small auth payload
func BenchmarkSanitize_LoginPayload(b *testing.B) {
	payload := benchLoginPayload{
		Origin:    "https://app.blockwyre.com",
		UserAgent: "Mozilla/5.0",
	}
	payload.Body.Token = "token_MRe0FMIzQLLTiRVp6E6QK0Lsk"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Sanitize(payload)
	}
}

// BenchmarkSanitize_NonStruct measures passthrough for non-struct types
func BenchmarkSanitize_NonStruct(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Sanitize("just a string")
	}
}
