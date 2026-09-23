package bot

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestCommandDefinitions(t *testing.T) {
	cmds := commandDefinitions()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 command definitions, got %d", len(cmds))
	}

	cmdMap := make(map[string]*discordgo.ApplicationCommand)
	for _, cmd := range cmds {
		cmdMap[cmd.Name] = cmd
		if cmd.DMPermission == nil || *cmd.DMPermission != false {
			t.Errorf("command %s should have DMPermission set to false", cmd.Name)
		}
	}

	saham, ok := cmdMap["saham"]
	if !ok {
		t.Fatal("missing /saham command")
	}
	if len(saham.Options) != 1 {
		t.Fatalf("expected 1 option for /saham, got %d", len(saham.Options))
	}
	opt := saham.Options[0]
	if opt.Name != "kode" || !opt.Required || !opt.Autocomplete {
		t.Errorf("unexpected option configuration for /saham: %+v", opt)
	}

	scan, ok := cmdMap["scan"]
	if !ok {
		t.Fatal("missing /scan command")
	}
	if len(scan.Options) != 3 {
		t.Fatalf("expected 3 subcommands for /scan, got %d", len(scan.Options))
	}
	subNames := []string{scan.Options[0].Name, scan.Options[1].Name, scan.Options[2].Name}
	expectedSubs := []string{"ara", "arb", "all"}
	for i, name := range subNames {
		if name != expectedSubs[i] {
			t.Errorf("expected subcommand %s at index %d, got %s", expectedSubs[i], i, name)
		}
		if scan.Options[i].Type != discordgo.ApplicationCommandOptionSubCommand {
			t.Errorf("expected option %s to be SubCommand type", name)
		}
	}

	_, ok = cmdMap["verif"]
	if !ok {
		t.Fatal("missing /verif command")
	}
}

func TestStockSymbol(t *testing.T) {
	tests := []struct {
		input     string
		wantSym   string
		wantValid bool
	}{
		{"BBCA", "BBCA", true},
		{"bbca", "BBCA", true},
		{"bbca.jk", "BBCA", true},
		{"BBRI.JK", "BBRI", true},
		{"ihsg", "IHSG", true},
		{"jkse", "IHSG", true},
		{"^JKSE", "IHSG", true},
		{"", "", false},
		{"A", "A", false}, // too short (< 2 chars)
		{"INVALID!@#", "INVALID!@#", false},
		{"   TLKM   ", "TLKM", true},
	}

	for _, tt := range tests {
		sym, valid := stockSymbol(tt.input)
		if sym != tt.wantSym || valid != tt.wantValid {
			t.Errorf("stockSymbol(%q) = (%q, %v), want (%q, %v)", tt.input, sym, valid, tt.wantSym, tt.wantValid)
		}
	}
}

func TestStockChoices(t *testing.T) {
	// Query starting with BB
	choices := stockChoices("BB")
	if len(choices) == 0 {
		t.Fatal("expected choices for BB, got none")
	}
	for _, c := range choices {
		if !strings.HasPrefix(c.Name, "BB") {
			t.Errorf("expected choice %s to start with BB", c.Name)
		}
	}

	// Empty query returns up to 25 items
	all := stockChoices("")
	if len(all) != 25 {
		t.Errorf("expected 25 choices for empty query, got %d", len(all))
	}

	// Non-matching query
	none := stockChoices("ZZZZ")
	if len(none) != 0 {
		t.Errorf("expected 0 choices for ZZZZ, got %d", len(none))
	}
}

func TestRefreshComponents(t *testing.T) {
	comps := refreshComponents("BBCA", "123456789")
	if len(comps) != 1 {
		t.Fatalf("expected 1 row component, got %d", len(comps))
	}
	row, ok := comps[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatal("expected ActionsRow")
	}
	if len(row.Components) != 1 {
		t.Fatalf("expected 1 component in row, got %d", len(row.Components))
	}
	btn, ok := row.Components[0].(discordgo.Button)
	if !ok {
		t.Fatal("expected Button")
	}
	expectedCustomID := "stock_refresh:123456789:BBCA"
	if btn.CustomID != expectedCustomID {
		t.Errorf("expected CustomID %s, got %s", expectedCustomID, btn.CustomID)
	}
	if btn.Label != "Refresh" {
		t.Errorf("expected Label 'Refresh', got %s", btn.Label)
	}
}

func TestHandleRefreshCustomIDParsing(t *testing.T) {
	// CustomID not starting with stock_refresh: should return false
	nonRefreshInteraction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "other_button",
			},
		},
	}
	if handleRefresh(nil, nonRefreshInteraction) {
		t.Error("expected handleRefresh to return false for other_button")
	}
}
