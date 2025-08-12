package utils

import (
	"fmt"
	"sort"
	"strings"
)

// helpers for simulation to keep SimulateMulti easy to read

// move shows one ant stepping into a room.
type move struct {
	ant  int
	room *Room
}

// moveAnts moves ants already on their paths.
func moveAnts(g *Graph, paths [][]*Room, loc []int, going []bool, busy map[*Room]int, done *int, plan []int) []move {
	var ms []move
	for ant := 0; ant < len(plan); ant++ {
		if !going[ant] {
			continue
		}
		path := paths[plan[ant]]
		if loc[ant] < len(path)-1 {
			next := path[loc[ant]+1]
			if next == g.End || busy[next] == 0 {
				if path[loc[ant]] != g.Start {
					delete(busy, path[loc[ant]])
				}
				loc[ant]++
				if next != g.End {
					busy[next] = ant + 1
				} else {
					*done++
				}
				ms = append(ms, move{ant: ant + 1, room: next})
			}
		}
	}
	return ms
}

// startAnts starts new ants if the next room is free.
func startAnts(g *Graph, paths [][]*Room, wait [][]int, going []bool, loc []int, busy map[*Room]int, done *int) []move {
	var ms []move
	for i, q := range wait {
		if len(q) == 0 {
			continue
		}
		ant := q[0]
		next := paths[i][1]
		if next == g.End || busy[next] == 0 {
			going[ant] = true
			loc[ant] = 1
			if next != g.End {
				busy[next] = ant + 1
			} else {
				*done++
			}
			wait[i] = q[1:]
			ms = append(ms, move{ant: ant + 1, room: next})
		}
	}
	return ms
}

// formatMoves turns a slice of moves into output text.
func formatMoves(ms []move) string {
	sort.Slice(ms, func(i, j int) bool { return ms[i].ant < ms[j].ant })
	line := make([]string, len(ms))
	for i, m := range ms {
		line[i] = fmt.Sprintf("L%d-%s", m.ant, m.room.Name)
	}
	return strings.Join(line, " ")
}
