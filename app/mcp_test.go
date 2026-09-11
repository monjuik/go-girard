package app

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/monjuik/go-girard/contacts"
)

func TestMCPHTTP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fixture := newServerFixture(t)
	fixture.personQueries.person = contacts.PersonView{ID: "101", Name: "Anna"}
	server := httptest.NewServer(fixture.handler)
	defer server.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"list_due_intentions": false,
		"get_person_context":  false,
	}
	if len(result.Tools) != len(want) {
		t.Fatalf("got %d tools, want %d", len(result.Tools), len(want))
	}
	for _, tool := range result.Tools {
		seen, exists := want[tool.Name]
		if !exists || seen {
			t.Fatalf("unexpected or duplicate tool %q", tool.Name)
		}
		want[tool.Name] = true
		if tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Errorf("%s: missing input or output schema", tool.Name)
		}
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s: missing read-only annotation", tool.Name)
		}
	}

	for _, call := range []mcp.CallToolParams{
		{Name: "list_due_intentions", Arguments: map[string]any{}},
		{Name: "get_person_context", Arguments: map[string]any{"person_id": "101"}},
	} {
		result, err := session.CallTool(ctx, &call)
		if err != nil {
			t.Fatalf("%s: %v", call.Name, err)
		}
		if result.IsError || result.StructuredContent == nil {
			t.Fatalf("%s: unexpected result %+v", call.Name, result)
		}
	}
	if fixture.enrollmentQueries.dueThrough.String() != "2026-09-03" {
		t.Fatal("MCP did not use the UI clock")
	}
	if fixture.personQueries.id.String() != "101" ||
		fixture.enrollmentQueries.personID.String() != "101" {
		t.Fatal("person context did not query the requested person")
	}
}
