package repository

import (
	"testing"

	analytics "google.golang.org/api/analyticsdata/v1beta"
)

func row(title, host, path, pv string) *analytics.Row {
	return &analytics.Row{
		DimensionValues: []*analytics.DimensionValue{
			{Value: title},
			{Value: host},
			{Value: path},
		},
		MetricValues: []*analytics.MetricValue{
			{Value: pv},
		},
	}
}

func TestAggregateRows(t *testing.T) {
	tests := []struct {
		name       string
		rows       []*analytics.Row
		titleSplit string
		want       map[string]int // タイトル -> PV
		wantPath   map[string]string
		wantErr    bool
	}{
		{
			name: "sums PV for duplicate titles",
			rows: []*analytics.Row{
				row("記事A | サイト", "example.com", "/blog/a", "10"),
				row("記事A | サイト", "example.com", "/blog/a", "5"),
			},
			titleSplit: " | ",
			want:       map[string]int{"記事A": 15},
			wantPath:   map[string]string{"記事A": "example.com/blog/a"},
		},
		{
			name: "skips top-level path",
			rows: []*analytics.Row{
				row("トップ", "example.com", "/", "100"),
				row("記事B", "example.com", "/blog/b", "3"),
			},
			titleSplit: " | ",
			want:       map[string]int{"記事B": 3},
		},
		{
			name: "splits title on separator",
			rows: []*analytics.Row{
				row("見出し - サイト名", "example.com", "/blog/c", "7"),
			},
			titleSplit: " - ",
			want:       map[string]int{"見出し": 7},
		},
		{
			name:       "non-numeric PV is an error",
			rows:       []*analytics.Row{row("記事C", "example.com", "/blog/d", "abc")},
			titleSplit: " | ",
			wantErr:    true,
		},
		{
			name:       "empty titleSplit keeps the full title without panicking",
			rows:       []*analytics.Row{row("記事D", "example.com", "/blog/e", "4")},
			titleSplit: "",
			want:       map[string]int{"記事D": 4},
		},
		{
			name:       "empty title with empty titleSplit does not panic",
			rows:       []*analytics.Row{row("", "example.com", "/blog/f", "2")},
			titleSplit: "",
			want:       map[string]int{"": 2},
		},
		{
			name: "skips malformed rows",
			rows: []*analytics.Row{
				nil,
				{DimensionValues: []*analytics.DimensionValue{{Value: "x"}}}, // ディメンション不足
				row("記事E", "example.com", "/blog/g", "9"),
			},
			titleSplit: " | ",
			want:       map[string]int{"記事E": 9},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pageMap := make(map[string]*Page)
			err := aggregateRows(pageMap, tt.rows, tt.titleSplit)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for title, pv := range tt.want {
				p, ok := pageMap[title]
				if !ok {
					t.Fatalf("title %q missing from result", title)
				}
				if p.PV != pv {
					t.Errorf("title %q PV = %d, want %d", title, p.PV, pv)
				}
			}
			for title, path := range tt.wantPath {
				if got := pageMap[title].Path; got != path {
					t.Errorf("title %q Path = %q, want %q", title, got, path)
				}
			}
			if len(pageMap) != len(tt.want) {
				t.Errorf("got %d pages, want %d", len(pageMap), len(tt.want))
			}
		})
	}
}

func TestSortPages(t *testing.T) {
	pageMap := map[string]*Page{
		"low":  {Title: "low", PV: 1},
		"high": {Title: "high", PV: 100},
		"mid":  {Title: "mid", PV: 50},
	}

	got := sortPages(pageMap)

	want := []string{"high", "mid", "low"}
	if len(got) != len(want) {
		t.Fatalf("got %d pages, want %d", len(got), len(want))
	}
	for i, title := range want {
		if got[i].Title != title {
			t.Errorf("position %d = %q, want %q", i, got[i].Title, title)
		}
	}
}
