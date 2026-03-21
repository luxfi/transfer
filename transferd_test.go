package transfer_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/luxfi/transfer/pkg/comms"
	"github.com/luxfi/transfer/pkg/disclosures"
	"github.com/luxfi/transfer/pkg/dividends"
	"github.com/luxfi/transfer/pkg/filings"
	"github.com/luxfi/transfer/pkg/ledger"
	"github.com/luxfi/transfer/pkg/restrictions"
	"github.com/luxfi/transfer/pkg/router"
	"github.com/luxfi/transfer/pkg/shareholder"
	"github.com/luxfi/transfer/pkg/store"
	"github.com/luxfi/transfer/pkg/voting"
)

func setup() *httptest.Server {
	st := store.NewMemoryStore()
	r := router.New(
		shareholder.New(st),
		ledger.New(st),
		restrictions.New(st),
		disclosures.New(st),
		comms.New(st),
		dividends.New(st),
		filings.New(st),
		voting.New(st),
	)
	return httptest.NewServer(r)
}

func TestHealthz(t *testing.T) {
	ts := setup()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body["status"])
	}
}

func TestShareholderCRUD(t *testing.T) {
	ts := setup()
	defer ts.Close()

	// Create
	sh := map[string]interface{}{
		"name":       "Alice Smith",
		"email":      "alice@example.com",
		"type":       "individual",
		"accredited": true,
	}
	resp := postJSON(t, ts, "/shareholders", sh)
	if resp.StatusCode != 201 {
		t.Fatalf("create: expected 201, got %d", resp.StatusCode)
	}
	var created store.Shareholder
	readJSON(t, resp, &created)
	if created.ID == "" {
		t.Fatal("expected ID to be generated")
	}
	if created.Name != "Alice Smith" {
		t.Fatalf("expected name Alice Smith, got %s", created.Name)
	}

	// Get
	resp2, err := http.Get(ts.URL + "/shareholders/" + created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resp2.StatusCode != 200 {
		t.Fatalf("get: expected 200, got %d", resp2.StatusCode)
	}

	// List
	resp3, err := http.Get(ts.URL + "/shareholders")
	if err != nil {
		t.Fatal(err)
	}
	var list []store.Shareholder
	readJSON(t, resp3, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 shareholder, got %d", len(list))
	}

	// Not found
	resp4, err := http.Get(ts.URL + "/shareholders/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if resp4.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp4.StatusCode)
	}
}

func TestTransferLedger(t *testing.T) {
	ts := setup()
	defer ts.Close()

	// Create two shareholders
	sh1 := createShareholder(t, ts, "Issuer Corp", "issuer@example.com", "entity")
	sh2 := createShareholder(t, ts, "Bob Jones", "bob@example.com", "individual")

	// Issue shares to Issuer Corp
	issuance := map[string]interface{}{
		"to_shareholder_id": sh1.ID,
		"share_class_id":    "common-a",
		"quantity":          1000,
		"price_per_share":   10.00,
		"type":              "issuance",
		"reason":            "initial issuance",
	}
	resp := postJSON(t, ts, "/transfers", issuance)
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("issuance: expected 201, got %d: %s", resp.StatusCode, string(body))
	}

	// Transfer shares from Issuer to Bob
	transfer := map[string]interface{}{
		"from_shareholder_id": sh1.ID,
		"to_shareholder_id":   sh2.ID,
		"share_class_id":      "common-a",
		"quantity":            100,
		"price_per_share":     12.00,
		"type":                "transfer",
	}
	resp2 := postJSON(t, ts, "/transfers", transfer)
	if resp2.StatusCode != 201 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("transfer: expected 201, got %d: %s", resp2.StatusCode, string(body))
	}

	// Check ledger
	resp3, err := http.Get(ts.URL + "/ledger/common-a")
	if err != nil {
		t.Fatal(err)
	}
	var transfers []store.Transfer
	readJSON(t, resp3, &transfers)
	if len(transfers) != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", len(transfers))
	}
}

func TestRestrictions(t *testing.T) {
	ts := setup()
	defer ts.Close()

	sh := createShareholder(t, ts, "Charlie Restricted", "charlie@example.com", "individual")

	// Issue shares
	postJSON(t, ts, "/transfers", map[string]interface{}{
		"to_shareholder_id": sh.ID,
		"share_class_id":    "common-a",
		"quantity":          500,
		"price_per_share":   10.00,
		"type":              "issuance",
	})

	// Add lockup restriction
	rst := map[string]interface{}{
		"shareholder_id": sh.ID,
		"share_class_id": "common-a",
		"type":           "lockup",
		"description":    "12 month lockup from issuance",
	}
	resp := postJSON(t, ts, "/restrictions", rst)
	if resp.StatusCode != 201 {
		t.Fatalf("create restriction: expected 201, got %d", resp.StatusCode)
	}

	// Check transfer — should be blocked
	check := map[string]interface{}{
		"from_shareholder_id": sh.ID,
		"to_shareholder_id":   "someone",
		"share_class_id":      "common-a",
		"quantity":            100,
	}
	resp2 := postJSON(t, ts, "/restrictions/check", check)
	var result store.RestrictionCheck
	readJSON(t, resp2, &result)
	if result.Allowed {
		t.Fatal("expected transfer to be blocked by lockup")
	}
	if len(result.Violations) == 0 {
		t.Fatal("expected violation messages")
	}
}

func TestDividendLifecycle(t *testing.T) {
	ts := setup()
	defer ts.Close()

	sh := createShareholder(t, ts, "Dividend Holder", "div@example.com", "individual")

	// Issue shares
	postJSON(t, ts, "/transfers", map[string]interface{}{
		"to_shareholder_id": sh.ID,
		"share_class_id":    "preferred-b",
		"quantity":          200,
		"price_per_share":   25.00,
		"type":              "issuance",
	})

	// Declare dividend
	now := time.Now()
	div := map[string]interface{}{
		"share_class_id":   "preferred-b",
		"type":             "cash",
		"amount_per_share": 1.50,
		"record_date":      now.Format(time.RFC3339),
		"payment_date":     now.Add(30 * 24 * time.Hour).Format(time.RFC3339),
	}
	resp := postJSON(t, ts, "/dividends", div)
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("declare dividend: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created store.Dividend
	readJSON(t, resp, &created)

	// Calculate
	resp2 := postJSON(t, ts, "/dividends/"+created.ID+"/calculate", nil)
	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("calculate: expected 200, got %d: %s", resp2.StatusCode, string(body))
	}
	var calculated store.Dividend
	readJSON(t, resp2, &calculated)
	if calculated.Status != "calculated" {
		t.Fatalf("expected status calculated, got %s", calculated.Status)
	}
	if len(calculated.Distributions) != 1 {
		t.Fatalf("expected 1 distribution, got %d", len(calculated.Distributions))
	}
	if calculated.Distributions[0].Amount != 300.00 { // 200 shares * $1.50
		t.Fatalf("expected amount 300.00, got %f", calculated.Distributions[0].Amount)
	}

	// Pay
	resp3 := postJSON(t, ts, "/dividends/"+created.ID+"/pay", nil)
	if resp3.StatusCode != 200 {
		body, _ := io.ReadAll(resp3.Body)
		t.Fatalf("pay: expected 200, got %d: %s", resp3.StatusCode, string(body))
	}
	var paid store.Dividend
	readJSON(t, resp3, &paid)
	if paid.Status != "paid" {
		t.Fatalf("expected status paid, got %s", paid.Status)
	}
}

func TestVotingLifecycle(t *testing.T) {
	ts := setup()
	defer ts.Close()

	sh1 := createShareholder(t, ts, "Voter A", "a@example.com", "individual")
	sh2 := createShareholder(t, ts, "Voter B", "b@example.com", "individual")

	// Issue shares
	postJSON(t, ts, "/transfers", map[string]interface{}{
		"to_shareholder_id": sh1.ID,
		"share_class_id":    "common-a",
		"quantity":          600,
		"type":              "issuance",
	})
	postJSON(t, ts, "/transfers", map[string]interface{}{
		"to_shareholder_id": sh2.ID,
		"share_class_id":    "common-a",
		"quantity":          400,
		"type":              "issuance",
	})

	// Create proposal
	now := time.Now()
	prop := map[string]interface{}{
		"title":           "Elect Board Member",
		"description":     "Vote to elect Jane Doe to the board",
		"type":            "board_election",
		"share_class_ids": []string{"common-a"},
		"record_date":     now.Format(time.RFC3339),
		"deadline":        now.Add(14 * 24 * time.Hour).Format(time.RFC3339),
		"quorum_percent":  50.0,
		"status":          "open",
	}
	resp := postJSON(t, ts, "/proposals", prop)
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create proposal: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created store.Proposal
	readJSON(t, resp, &created)

	// Cast votes
	postJSON(t, ts, "/proposals/"+created.ID+"/vote", map[string]interface{}{
		"shareholder_id": sh1.ID,
		"choice":         "for",
		"shares_voted":   600,
	})
	postJSON(t, ts, "/proposals/"+created.ID+"/vote", map[string]interface{}{
		"shareholder_id": sh2.ID,
		"choice":         "against",
		"shares_voted":   400,
	})

	// Get results
	resp4, err := http.Get(ts.URL + "/proposals/" + created.ID + "/results")
	if err != nil {
		t.Fatal(err)
	}
	var results store.VoteResults
	readJSON(t, resp4, &results)
	if results.For != 600 {
		t.Fatalf("expected 600 for, got %d", results.For)
	}
	if results.Against != 400 {
		t.Fatalf("expected 400 against, got %d", results.Against)
	}
	if !results.QuorumMet {
		t.Fatal("expected quorum to be met (100% voted, 50% required)")
	}
}

func TestDisclosuresAndComms(t *testing.T) {
	ts := setup()
	defer ts.Close()

	sh := createShareholder(t, ts, "Investor Dana", "dana@example.com", "individual")

	// Create disclosure
	disc := map[string]interface{}{
		"name":         "Q4 2025 PPM",
		"type":         "ppm",
		"document_url": "https://docs.example.com/ppm-q4-2025.pdf",
		"recipients": []map[string]interface{}{
			{"shareholder_id": sh.ID},
		},
	}
	resp := postJSON(t, ts, "/disclosures", disc)
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create disclosure: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created store.Disclosure
	readJSON(t, resp, &created)

	// Deliver
	resp2 := postJSON(t, ts, "/disclosures/"+created.ID+"/deliver", map[string]interface{}{
		"shareholder_id": sh.ID,
	})
	if resp2.StatusCode != 200 {
		body, _ := io.ReadAll(resp2.Body)
		t.Fatalf("deliver: expected 200, got %d: %s", resp2.StatusCode, string(body))
	}

	// Acknowledge
	resp3 := postJSON(t, ts, "/disclosures/"+created.ID+"/acknowledge", map[string]interface{}{
		"shareholder_id": sh.ID,
	})
	if resp3.StatusCode != 200 {
		body, _ := io.ReadAll(resp3.Body)
		t.Fatalf("acknowledge: expected 200, got %d: %s", resp3.StatusCode, string(body))
	}

	// Create and send notice
	notice := map[string]interface{}{
		"subject": "Annual Report Available",
		"body":    "Your annual report is now available.",
		"type":    "general",
		"recipients": []map[string]interface{}{
			{"shareholder_id": sh.ID, "email": "dana@example.com"},
		},
	}
	resp4 := postJSON(t, ts, "/notices", notice)
	if resp4.StatusCode != 201 {
		body, _ := io.ReadAll(resp4.Body)
		t.Fatalf("create notice: expected 201, got %d: %s", resp4.StatusCode, string(body))
	}
	var createdNotice store.Notice
	readJSON(t, resp4, &createdNotice)

	resp5 := postJSON(t, ts, "/notices/"+createdNotice.ID+"/send", nil)
	if resp5.StatusCode != 200 {
		body, _ := io.ReadAll(resp5.Body)
		t.Fatalf("send: expected 200, got %d: %s", resp5.StatusCode, string(body))
	}
}

func TestFilings(t *testing.T) {
	ts := setup()
	defer ts.Close()

	filing := map[string]interface{}{
		"type":         "form_d",
		"jurisdiction": "SEC",
		"data": map[string]interface{}{
			"issuer_name":  "Acme Corp",
			"offering_amt": 5000000,
		},
	}
	resp := postJSON(t, ts, "/filings", filing)
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create filing: expected 201, got %d: %s", resp.StatusCode, string(body))
	}
	var created store.Filing
	readJSON(t, resp, &created)
	if created.Status != "draft" {
		t.Fatalf("expected status draft, got %s", created.Status)
	}

	// List
	resp2, err := http.Get(ts.URL + "/filings?type=form_d")
	if err != nil {
		t.Fatal(err)
	}
	var list []store.Filing
	readJSON(t, resp2, &list)
	if len(list) != 1 {
		t.Fatalf("expected 1 filing, got %d", len(list))
	}
}

// --- helpers ---

func createShareholder(t *testing.T, ts *httptest.Server, name, email, typ string) store.Shareholder {
	t.Helper()
	resp := postJSON(t, ts, "/shareholders", map[string]interface{}{
		"name":  name,
		"email": email,
		"type":  typ,
	})
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("create shareholder %s: expected 201, got %d: %s", name, resp.StatusCode, string(body))
	}
	var sh store.Shareholder
	readJSON(t, resp, &sh)
	return sh
}

func postJSON(t *testing.T, ts *httptest.Server, path string, v interface{}) *http.Response {
	t.Helper()
	var body io.Reader
	if v != nil {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader([]byte("{}"))
	}
	resp, err := http.Post(ts.URL+path, "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func readJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
