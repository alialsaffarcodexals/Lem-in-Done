package utils

import "sort"

func allPaths(g *Graph, limit int) [][]*Room {
	var res [][]*Room           // Box to store all paths we find
	path := []*Room{}           // Current path we are walking
	visited := map[*Room]bool{} // List of rooms we already visited

	var dfs func(*Room)
	dfs = func(r *Room) {
		// Stop if we found enough paths
		if len(res) >= limit {
			return
		}

		// Found the exit! Save this path
		if r == g.End {
			p := append(append([]*Room{}, path...), r)
			res = append(res, p)
			return
		}

		// Mark this room as visited and add to current path
		visited[r] = true
		path = append(path, r)

		// Try all connected rooms that we haven't visited
		for _, nb := range r.Links {
			if !visited[nb] {
				dfs(nb) // Go explore that room
			}
		}

		// Backtrack: remove this room from path and mark as unvisited
		path = path[:len(path)-1]
		visited[r] = false
	}
	dfs(g.Start) // Begin from the start room
	return res   // Return all paths found
}

func bestDisjointPaths(all [][]*Room, ants int) [][]*Room {
	bestTurns := int(^uint(0) >> 1) // Very big number (worst case)
	var best [][]*Room              // Best paths we found
	var bestIdx []int               // Which path numbers we picked
	var rec func(int, [][]*Room, []int, map[*Room]bool)

	// This function finds the best combination of paths for ants to use
	rec = func(i int, cur [][]*Room, idxs []int, used map[*Room]bool) {

		// BASE CASE: We've looked at all available paths
		if i == len(all) {
			// Skip empty combinations
			if len(cur) == 0 {
				return
			}

			// Calculate path lengths (subtract 1 because we count steps, not rooms)
			lens := make([]int, len(cur))
			for j, path := range cur {
				lens[j] = len(path) - 1
			}

			// Calculate total turns needed for this combination
			t := countTurns(ants, lens)

			// Check if this combination is better than our current best
			better := false

			// First priority: fewer turns is always better
			if t < bestTurns {
				better = true
			} else if t == bestTurns {
				// Second priority: if same turns, prefer smaller path indexes
				for k := 0; k < len(idxs) && k < len(bestIdx); k++ {
					if idxs[k] < bestIdx[k] {
						better = true
						break
					} else if idxs[k] > bestIdx[k] {
						break
					}
				}
				// Third priority: prefer fewer paths if indexes are the same
				if !better && len(idxs) < len(bestIdx) {
					better = true
				}
			}

			// Save this combination if it's better
			if better {
				bestTurns = t
				best = append([][]*Room{}, cur...) // Copy current paths
				bestIdx = append([]int{}, idxs...) // Copy current indexes
			}
			return
		}

		// RECURSIVE CASE: Try two options for path i

		// Option 1: Skip this path
		rec(i+1, cur, idxs, used)

		// Option 2: Try to use this path
		p := all[i] // Get the current path

		// Check if path is valid (no room conflicts)
		valid := true
		// Only check middle rooms (skip first and last room)
		for _, r := range p[1 : len(p)-1] {
			if used[r] { // Room already used by another path
				valid = false
				break
			}
		}

		// If path is valid, try using it
		if valid {
			// Mark middle rooms as used
			for _, r := range p[1 : len(p)-1] {
				used[r] = true
			}

			// Try this combination recursively
			rec(i+1, append(cur, p), append(idxs, i), used)

			// BACKTRACK: Unmark rooms so we can try other combinations
			for _, r := range p[1 : len(p)-1] {
				delete(used, r)
			}
		}
	}

	rec(0, nil, nil, map[*Room]bool{}) // Start with no paths selected
	return best                        // Return best combination
}

// Main function to find the best paths for ants to use
func FindPaths(g *Graph) [][]*Room {

	// Generate all possible paths from start to end
	// MaxPaths limits how many paths we consider (for performance)
	all := allPaths(g, MaxPaths)

	// Sort paths by length (shortest first)
	sort.SliceStable(all, func(i, j int) bool {
		li, lj := len(all[i]), len(all[j]) // Get lengths of both paths

		// If paths have same length, keep original order (stable sort)
		if li == lj {
			return i < j
		}

		// Otherwise, shorter path comes first
		return li < lj
	})

	// Find the best combination of non-overlapping paths
	return bestDisjointPaths(all, g.Ants)
}

// countTurns returns how many turns are needed for given paths and ants.
func countTurns(ants int, lens []int) int {
	for turns := 1; ; turns++ {
		total := 0
		for _, l := range lens {
			if turns-l >= 0 {
				total += turns - l + 1
			}
		}
		if total >= ants {
			return turns
		}
	}
}

// assignPaths picks a path index for every ant.
func assignPaths(paths [][]*Room, ants int) []int {
	// find length of each path
	lens := getLens(paths)
	// how many turns are needed in total
	turns := countTurns(ants, lens)
	// how many ants can start on each path
	starts := countStarts(lens, turns)
	// trim if we planned for more ants than we have
	starts = trimStarts(starts, ants)
	// make the order in which ants should leave
	return planOrder(starts)
}
