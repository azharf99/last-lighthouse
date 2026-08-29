package main

import (
	"fmt"
	"os"

	"github.com/lastlighthouse/lastlighthouse/bot"
	"github.com/lastlighthouse/lastlighthouse/core"
)

// traceGame mencetak setiap aksi dari satu partai.
//
// Angka agregat memberi tahu BAHWA ada masalah; trace memberi tahu DI MANA.
// Tanpa ini, temuan seperti "win rate 0%" hanya bisa ditanggapi dengan
// menebak-nebak konstanta mana yang salah.
func traceGame(seed int64, players int, level bot.Difficulty, c *core.Content) {
	setups := make([]core.PlayerSetup, 0, players)
	for i := range players {
		setups = append(setups, core.PlayerSetup{
			ID:        core.PlayerID(fmt.Sprintf("p%d", i+1)),
			Name:      fmt.Sprintf("Bot %d", i+1),
			Character: characters[i%len(characters)],
		})
	}

	s, _, err := core.NewGame(fmt.Sprintf("trace-%d", seed), seed, setups, c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	rng := core.NewRNG(s.RNGState)
	pick := core.NewRNG(seed ^ 0x2545F4914F6CDD1D)
	ai := bot.New(level)

	fmt.Printf("TRACE partai seed %d, %d pemain\n", seed, players)
	round := 0

	for range maxStepsPerGame {
		if s.Over() {
			break
		}
		if s.Round != round {
			round = s.Round
			fmt.Printf("\n== Ronde %d | Darkness %d | %s ==\n",
				round, s.Darkness, describeNextComponent(s))
		}

		actor := actingPlayer(s)
		if actor == "" {
			break
		}
		p := s.Player(actor)
		view := core.Project(s, actor)
		cmd, ok := ai.Choose(view, core.LegalCommands(view, c), pick)
		if !ok {
			break
		}

		fmt.Printf("  %-3s @%-11s AP%d inv%v  %s%s\n",
			actor, p.At, p.AP, p.Inventory, cmd.Kind, describeCommand(cmd))

		events, err := core.Decide(s, cmd, c, rng)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  DITOLAK: %v\n", err)
			return
		}
		core.ApplyAll(s, events)
		s.RNGState = rng.Seed()
	}

	fmt.Printf("\nSELESAI: status=%s ronde=%d darkness=%d\n", s.Status, s.Round, s.Darkness)
	for i := range s.Lighthouse {
		comp := &s.Lighthouse[i]
		mark := " "
		if comp.Repaired {
			mark = "x"
		}
		fmt.Printf("  [%s] %-13s butuh %v punya %v\n", mark, comp.ID, comp.Cost, comp.Progress)
	}
}

func describeNextComponent(s *core.State) string {
	comp := s.NextComponent()
	if comp == nil {
		return "semua komponen selesai"
	}
	return fmt.Sprintf("%s masih butuh %v", comp.ID, comp.Progress.Missing(comp.Cost))
}

func describeCommand(cmd core.Command) string {
	switch cmd.Kind {
	case core.CmdMove, core.CmdExplore:
		return " -> " + string(cmd.To)
	case core.CmdGather:
		return " " + cmd.Resource.String()
	case core.CmdRepair:
		return fmt.Sprintf(" bayar %v", cmd.Pay)
	case core.CmdTrade:
		return fmt.Sprintf(" %v ke %s", cmd.Give, cmd.Target)
	case core.CmdChoose:
		return " " + cmd.Option + string(cmd.Card)
	}
	return ""
}
