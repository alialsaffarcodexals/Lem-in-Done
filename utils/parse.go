package utils

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ParseInput(path string) (*Graph, []string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer file.Close()

	g := &Graph{Rooms: make(map[string]*Room)}
	scanner := bufio.NewScanner(file)
	var lines []string
	var pendingStart, pendingEnd bool
	var links [][2]string
	parsedAnts := false
	coords := map[[2]int]bool{}
	linkSeen := map[string]struct{}{}

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.HasPrefix(line, "#") {
			isStart, isEnd := CheckStartOrEnd(line, pendingStart, pendingEnd, g)
			if isStart {
				pendingStart = true
			} else if isEnd {
				pendingEnd = true
			}
			continue
		}

		if !parsedAnts {
			g.Ants, parsedAnts = CheckAnts(line)
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 3 {
			isStart, isEnd, isCoords, newG := CheckRoom(pendingStart, pendingEnd, g, fields, coords)
			if !isStart {
				pendingStart = false
			}
			if !isEnd {
				pendingEnd = false
			}
			if isCoords {
				x, _ := strconv.Atoi(fields[1])
				y, _ := strconv.Atoi(fields[2])
				coords[[2]int{x, y}] = true
			}
			g = newG
			continue
		}

		if strings.Count(line, "-") == 1 && !strings.Contains(line, " ") {
			parts := strings.Split(line, "-")
			newLinks, newSeen := CheckLink(linkSeen, links, parts[0], parts[1])
			linkSeen = newSeen
			links = newLinks
			continue
		}

		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: invalid line")
		os.Exit(1)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if g.Start == nil || g.End == nil {
		fmt.Println("ERROR: invalid data format")
		fmt.Println("Reason: missing start or end")
		os.Exit(1)
	}

	for _, l := range links {
		a, ok1 := g.Rooms[l[0]]
		b, ok2 := g.Rooms[l[1]]
		if !ok1 || !ok2 {
			fmt.Println("ERROR: invalid data format")
			fmt.Println("Reason: unknown room in link '" + l[0])
			os.Exit(1)
		}
		if !hasNeighbor(a, b) {
			a.Links = append(a.Links, b)
		}
		if !hasNeighbor(b, a) {
			b.Links = append(b.Links, a)
		}
	}
	return g, lines
}

func hasNeighbor(r, other *Room) bool {
	for _, nb := range r.Links {
		if nb == other {
			return true
		}
	}
	return false
}
