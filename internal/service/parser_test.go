package service

import "testing"

func TestParseMembers_BulletRoster(t *testing.T) {
	input := `Темер — 1300
• Арман — 1872
• Ержимбо — 1366
• Бека — 900
• Жандос — 974
• Асан — 900
• Мара — 5000
• Газа — 1564
• Дамир — 1066
• Рама — 1000`

	got := ParseMembers(input)

	want := map[string]int{
		"Темер": 1300, "Арман": 1872, "Ержимбо": 1366, "Бека": 900, "Жандос": 974,
		"Асан": 900, "Мара": 5000, "Газа": 1564, "Дамир": 1066, "Рама": 1000,
	}

	if len(got) != len(want) {
		t.Fatalf("parsed %d members, want %d", len(got), len(want))
	}
	for _, m := range got {
		rating, ok := want[m.Name]
		if !ok {
			t.Errorf("unexpected member %q", m.Name)
			continue
		}
		if rating != m.Rating {
			t.Errorf("%s rating = %d, want %d", m.Name, m.Rating, rating)
		}
	}
}

func TestParseMembers_MixedSeparators(t *testing.T) {
	input := `1. Alice - 1100
2) Bob: 1200
- Carol – 1300
Dan 1400
  * Eve — 1500`

	got := ParseMembers(input)
	want := map[string]int{
		"Alice": 1100, "Bob": 1200, "Carol": 1300, "Dan": 1400, "Eve": 1500,
	}
	if len(got) != len(want) {
		t.Fatalf("parsed %d members, want %d: %+v", len(got), len(want), got)
	}
	for _, m := range got {
		if want[m.Name] != m.Rating {
			t.Errorf("%s rating = %d, want %d", m.Name, m.Rating, want[m.Name])
		}
	}
}

func TestParseMembers_SkipsNonRosterLines(t *testing.T) {
	input := `Let's split into 2 teams!
Темер — 1300
• Арман — 1872
good luck`

	got := ParseMembers(input)
	if len(got) != 2 {
		t.Fatalf("parsed %d members, want 2: %+v", len(got), got)
	}
}

func TestParseMembers_EmptyOnGarbage(t *testing.T) {
	got := ParseMembers("hi there, how are you")
	if len(got) != 0 {
		t.Errorf("parsed %d, want 0: %+v", len(got), got)
	}
}
